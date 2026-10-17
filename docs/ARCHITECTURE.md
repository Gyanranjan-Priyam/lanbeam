# Architecture

## 1. High-level view

```
┌──────────────────────────────────────────────────────────┐
│                    TypeScript UI (one codebase)           │
│   Desktop shell │ Capacitor WebView │ Browser (guest)     │
└───────────────┬──────────────────────────────────────────┘
                │ HTTP + WebSocket (localhost or LAN guest API)
┌───────────────▼──────────────────────────────────────────┐
│                         Go Core                           │
│                                                           │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐ ┌───────────┐ │
│  │ Discovery │ │ Transfer  │ │  Crypto / │ │  Store    │ │
│  │  service  │ │  engine   │ │  Identity │ │ (SQLite)  │ │
│  └───────────┘ └───────────┘ └───────────┘ └───────────┘ │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐              │
│  │ Peer API  │ │ Control   │ │ Guest API │              │
│  │ (LAN TLS) │ │ API (lo)  │ │ (LAN TLS) │              │
│  └───────────┘ └───────────┘ └───────────┘              │
└───────────────┬──────────────────────────────────────────┘
                │ LAN: mDNS, UDP multicast, HTTPS
          Other peers and web guests
```

## 2. Two listeners, three APIs

The core runs two network listeners, which keeps the attack surface small.

| Listener | Binds to | Serves | Who connects |
|---|---|---|---|
| **LAN listener** | `0.0.0.0:42100` (TLS) | Peer API, Guest API, embedded guest web app | Other peers, web guests |
| **Control listener** | `127.0.0.1:42101` | Control API plus WebSocket events, embedded main UI | The local UI only |

| API | Purpose | Auth |
|---|---|---|
| **Peer API** (`/api/v1/...`) | Peer-to-peer discovery info and file transfer | TLS fingerprint pinning plus per-session tokens |
| **Control API** (`/control/v1/...`) | Local UI commands: list peers, start sends, accept/reject, settings | Random startup token, `Host`/`Origin` checks |
| **Guest API** (`/guest/v1/...`) | Web guest uploads/downloads via the host | One-time pairing token → short-lived bearer token, plus host approval |

## 3. Go module layout

```
core/
  cmd/lanbeam/            main.go: wiring, flags, signal handling
  internal/
    config/               config load/save, defaults, ports
    crypto/               identity (cert/key), fingerprint, SAS code
    discovery/            Discoverer interface and implementations
      mdns/
      multicast/
      scan/
    peer/                 peer registry (in-memory plus persisted)
    transfer/
      session.go          session state machine
      sender.go           outbound transfers
      receiver.go         inbound transfers
      files.go            safe filenames, .part files, collisions
    api/
      peerapi/            LAN-facing handlers
      controlapi/         localhost handlers plus WebSocket hub
      guestapi/           guest handlers
      middleware/         logging, limits, auth
    store/                SQLite access, migrations
    events/               in-process pub/sub bus
    uiassets/             go:embed of built UI
  mobile/                 gomobile-exposed façade (Start/Stop/Port)
```

### Responsibilities

| Package | Responsibility | Depends on |
|---|---|---|
| `crypto` | Create/load identity, compute fingerprints, derive verification codes | stdlib only |
| `discovery` | Announce self, find peers, emit `PeerEvent`s | `crypto` (for info), `events` |
| `peer` | Single source of truth for known peers | `store`, `events` |
| `transfer` | Business logic of sessions, files, progress, integrity | `peer`, `store`, `events` |
| `api/*` | Translate HTTP to calls into `transfer`/`peer` | `transfer`, `peer`, `crypto` |
| `store` | Persistence only, no business logic | `modernc.org/sqlite` |
| `events` | Decouple producers (transfer, discovery) from consumers (WebSocket) | none |

**Dependency rule:** dependencies point inward. `api` depends on `transfer`; `transfer` never imports `api`.

## 4. Transfer session state machine

```
            ┌─────────┐
 offer ───► │ Offered │
            └────┬────┘
        accept   │   reject / timeout
      ┌──────────┴───────────┐
      ▼                      ▼
┌──────────────┐        ┌──────────┐
│ Transferring │        │ Rejected │
└──┬───┬───┬───┘        └──────────┘
   │   │   └──── cancel ───► ┌───────────┐
   │   └──── error ────────► │  Failed   │ (resumable)
   │                         └───────────┘
   ▼
┌───────────┐  hash ok   ┌───────────┐
│ Verifying │ ─────────► │ Completed │
└─────┬─────┘            └───────────┘
      │ hash mismatch
      ▼
 ┌───────────┐
 │  Failed   │
 └───────────┘
```

Additional rules:
- Offers expire after a configurable timeout (default 60 s) if the receiver does not respond.
- A `Failed` session keeps its `.part` files for a limited time so the sender can resume.
- Each file inside a session has its own sub-state (`Pending → Transferring → Verifying → Done/Failed`).

## 5. Data model (SQLite)

```sql
CREATE TABLE devices (
  fingerprint  TEXT PRIMARY KEY,   -- SHA-256 of cert DER, hex
  alias        TEXT NOT NULL,
  device_type  TEXT NOT NULL,      -- desktop | mobile | web | other
  trusted      INTEGER NOT NULL DEFAULT 0,
  first_seen   INTEGER NOT NULL,   -- unix seconds
  last_seen    INTEGER NOT NULL,
  last_address TEXT
);

CREATE TABLE transfers (
  id           TEXT PRIMARY KEY,   -- session id
  direction    TEXT NOT NULL,      -- send | receive
  peer_fp      TEXT REFERENCES devices(fingerprint),
  status       TEXT NOT NULL,
  total_bytes  INTEGER NOT NULL,
  started_at   INTEGER NOT NULL,
  finished_at  INTEGER
);

CREATE TABLE transfer_files (
  id           TEXT PRIMARY KEY,
  transfer_id  TEXT NOT NULL REFERENCES transfers(id) ON DELETE CASCADE,
  name         TEXT NOT NULL,
  rel_path     TEXT,
  size         INTEGER NOT NULL,
  hash         TEXT,
  bytes_done   INTEGER NOT NULL DEFAULT 0,
  status       TEXT NOT NULL,
  saved_path   TEXT
);

CREATE TABLE settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
```

Use numbered migrations (`001_init.sql`, `002_...sql`) applied at startup inside a transaction.

## 6. Concurrency model

- One goroutine per active file transfer, bounded by a semaphore (default 3 concurrent files per session).
- Every transfer receives a `context.Context` tied to the session. Cancel the context to abort everything cleanly.
- Progress is reported through a throttled channel (at most about 10 updates per second per file) into the event bus.
- The event bus fans out to WebSocket clients. Slow consumers are dropped rather than allowed to block producers.
- Shared maps (sessions, peers) are protected by `sync.RWMutex`, or by a single owner goroutine if that proves cleaner.

## 7. Web guest architecture

```
Host app (Go)                          Guest browser
─────────────                          ─────────────
Shows QR: https://IP:42100/g/#t=TOKEN ──► scans, opens page
                                       Page POSTs token to /guest/v1/pair
Prompt: "Pixel-Chrome wants to connect.
Code 4821. Allow?"                     Page shows code 4821
User taps Allow ───────────────────────► receives bearer token
                                       Upload: PUT /guest/v1/upload/...
                                       Download: GET /guest/v1/files/...
```

- The guest can send files **to the host** and download files the **host chooses to share**.
- Relaying guest files on to other peers is a v1.1 enhancement (host acts as a bridge).
- The guest page is the same TypeScript UI built with a `guest` flag.

## 8. Mobile embedding

```
Capacitor app
 ├─ WebView ──► loads http://127.0.0.1:PORT (control listener)
 └─ Native plugin
      ├─ calls Go via gomobile: Start(cfg) / Stop() / Port()
      ├─ native discovery (NsdManager / Bonjour) → posts peers to Go core
      └─ foreground service (Android) / foreground-only (iOS)
```

The Go core and UI are identical to desktop. Only the thin native plugin differs.

## 9. Configuration

| Setting | Default | Notes |
|---|---|---|
| `alias` | Device hostname | Shown to other devices |
| `lan_port` | 42100 | Falls back to a random free port if taken, announced via discovery |
| `control_port` | 42101 | Localhost only |
| `download_dir` | OS downloads folder | Per-platform default |
| `auto_accept_trusted` | false | |
| `max_concurrent_files` | 3 | |
| `offer_timeout_s` | 60 | |
| `interface` | auto | Allow the user to pick if several networks are active |

## 10. Observability

- Structured logging with `log/slog`; levels via flag or env var.
- A "Diagnostics" screen in the UI shows: chosen network interface, local IP, discovery method status, open ports, last errors. This is what you will want when users report "I can't see my other device."
- No telemetry. Logs stay on the device.

## 11. Error-handling principles

- Every error returned over HTTP uses the common envelope defined in `PROTOCOL.md`.
- Network errors during transfer mark the file `Failed (resumable)` rather than discarding progress.
- Never crash the process on a single bad peer request. Recover from panics in middleware and log them.