# Protocol Specification (v1 draft)

This is **your own protocol**. It is inspired by common LAN-transfer designs, but do not copy another project's spec or code without checking its license.

Convert this document into an **OpenAPI 3** file (`packages/protocol/openapi.yaml`) as soon as the shapes settle, then generate Go and TypeScript code from it.

---

## 1. Constants

| Name | Value |
|---|---|
| Protocol version | `1` |
| Default LAN port (TLS) | `42100` |
| Default control port (localhost) | `42101` |
| mDNS service type | `_lanbeam._tcp` |
| Multicast group (admin-scoped) | `239.255.42.99` |
| Multicast UDP port | `42099` |
| Max concurrent files per session | 3 (configurable) |
| Offer timeout | 60 s |
| Identity | SHA-256 fingerprint of the certificate (DER), lowercase hex |

---

## 2. Discovery

Discovery runs **all layers at the same time**, merging results by fingerprint.

### 2.1 mDNS / DNS-SD (primary)

Announce service `_lanbeam._tcp.local` on the LAN port with TXT records:

| Key | Example | Meaning |
|---|---|---|
| `v` | `1` | Protocol version |
| `id` | `a3f9...` (first 32 hex chars of fingerprint) | Device ID prefix |
| `alias` | `Ravi-Laptop` | Display name |
| `type` | `desktop` | `desktop`, `mobile`, `web`, `other` |

The full fingerprint is confirmed through `GET /api/v1/info` over TLS. Never trust TXT records alone.

### 2.2 UDP multicast (fallback)

Send a JSON datagram to `239.255.42.99:42099` on every active IPv4 interface, on startup and every 5 seconds:

```json
{
  "v": 1,
  "id": "a3f9c1...",
  "alias": "Ravi-Laptop",
  "type": "desktop",
  "port": 42100,
  "announce": true
}
```

On receiving an announcement with `"announce": true` from an unknown device, reply by unicast `HTTP GET /api/v1/info` to the sender (preferred) rather than another multicast. Keep payloads under 512 bytes.

### 2.3 Subnet scan (fallback)

If no peers are found after about 5 seconds, or on user request, probe every host in the local `/24` on the LAN port with a short timeout (300–500 ms) and a bounded worker pool (for example 64 goroutines). A valid response to `GET /api/v1/info` identifies a peer.

### 2.4 Manual entry and QR (always available)

- User types an IP (and optional port).
- Or scans a QR code containing `lanbeam://connect?ip=192.168.1.5&port=42100&fp=<fingerprint>`.

### 2.5 Peer liveness

A peer not seen for 30 s through any method is marked offline in the UI (but remains in the registry). Remove it from the visible list after 2 minutes.

---

## 3. Transport and identity

- All peer and guest traffic uses **HTTPS** (TLS 1.2+, prefer 1.3).
- Each device has a self-signed **ECDSA P-256** certificate generated on first run.
- **Clients pin the fingerprint.** For a known device, the expected fingerprint comes from the registry. For a new device, it is trusted on first use after user confirmation (see `SECURITY.md`).
- HTTP/2 is used automatically by Go's TLS server when enabled.

---

## 4. Common conventions

### 4.1 Error envelope

```json
{
  "error": {
    "code": "REJECTED",
    "message": "Receiver declined the transfer."
  }
}
```

| HTTP | `code` | Meaning |
|---|---|---|
| 400 | `INVALID_REQUEST` | Malformed body or parameters |
| 401 | `UNAUTHORIZED` | Missing/invalid token |
| 403 | `REJECTED` | Receiver rejected, or sender not allowed |
| 404 | `NOT_FOUND` | Unknown session or file |
| 409 | `BUSY` | Receiver is already handling a session, or state conflict |
| 410 | `EXPIRED` | Session expired |
| 413 | `TOO_LARGE` | Exceeds receiver limit or free space |
| 422 | `SIZE_MISMATCH` / `HASH_MISMATCH` | Integrity failure |
| 429 | `RATE_LIMITED` | Too many requests |
| 500 | `INTERNAL` | Unexpected error |

### 4.2 Content types

JSON for control messages (`application/json`). File bytes use `application/octet-stream`.

### 4.3 Versioning

Path-prefixed (`/api/v1`). `GET /api/v1/info` returns `protocolVersion`; peers must refuse to talk if the major versions differ and show a helpful "update the other device" message.

---

## 5. Peer API

### 5.1 `GET /api/v1/info`

Returns the device identity.

```json
{
  "protocolVersion": 1,
  "appVersion": "0.1.0",
  "alias": "Ravi-Laptop",
  "deviceType": "desktop",
  "fingerprint": "a3f9c1d2e4...64 hex chars...",
  "acceptingTransfers": true
}
```

### 5.2 `POST /api/v1/prepare-send`

The sender offers files. The receiver's UI asks the user to accept or reject. The request **blocks** until the user answers or the offer times out.

Request:

```json
{
  "sender": {
    "alias": "Pixel-7",
    "deviceType": "mobile",
    "fingerprint": "7be2..."
  },
  "files": [
    {
      "id": "f1",
      "name": "report.pdf",
      "relPath": "docs/report.pdf",
      "size": 1048576,
      "mime": "application/pdf",
      "hash": "blake3:9a1c..."
    }
  ],
  "text": null
}
```

- `hash` is optional. If absent, the receiver skips verification (or the sender sends the hash at the end, a v1.1 option).
- `relPath` is used for folder transfers (relative, forward slashes, no `..`).
- `text` carries a plain-text/clipboard share instead of files.

Success `200`:

```json
{
  "sessionId": "s_8f2a...",
  "files": {
    "f1": { "token": "tok_...", "offset": 0 }
  }
}
```

- `files` contains only files the receiver accepted (the user may untick some).
- `offset` is greater than 0 when a partial `.part` file already exists (resume).

Errors: `403 REJECTED`, `409 BUSY`, `413 TOO_LARGE`, `400 INVALID_REQUEST`.

### 5.3 `PUT /api/v1/send/{sessionId}/{fileId}`

Streams the file bytes.

Headers:

| Header | Meaning |
|---|---|
| `Authorization: Bearer <token>` | Token from `prepare-send` for this file |
| `Upload-Offset: <n>` | Byte offset at which this body starts (default 0) |
| `Content-Length` | Number of bytes in this body |

Behaviour:
- The receiver writes to `<name>.part`, seeking to `Upload-Offset`.
- When total bytes written equals the declared `size`, the receiver verifies the hash (if provided), renames the file into place, and returns `200`.
- On a dropped connection, the sender calls `HEAD` to find the offset and resumes.

Success `200`:

```json
{ "fileId": "f1", "status": "completed", "savedAs": "report.pdf" }
```

`savedAs` may differ from the name if a collision forced a rename.

### 5.4 `HEAD /api/v1/send/{sessionId}/{fileId}`

Returns `Upload-Offset: <n>` telling the sender how many bytes the receiver already has. Used for resume.

### 5.5 `POST /api/v1/cancel/{sessionId}`

Either side may cancel. Idempotent. Returns `204`.

### 5.6 Text/clipboard shares

Sent via `prepare-send` with `text` set and `files` empty. The receiver shows the text with Copy and Dismiss buttons. No `PUT` step is needed.

---

## 6. Control API (localhost only)

Base: `http://127.0.0.1:42101/control/v1`. Every request needs the startup token (`Authorization: Bearer ...`), and the server must validate `Host` and `Origin` (defence against DNS rebinding and cross-site requests).

| Method and path | Purpose |
|---|---|
| `GET /self` | Own identity, IPs, ports, settings |
| `GET /peers` | Known peers with online status |
| `POST /peers/manual` | Add a peer by IP |
| `POST /send` | Start a send: `{ peerFingerprint, files: [paths or upload handles] }` |
| `GET /transfers` | Active and historical transfers |
| `POST /transfers/{id}/accept` | Accept an incoming offer (with per-file selection) |
| `POST /transfers/{id}/reject` | Reject an incoming offer |
| `POST /transfers/{id}/cancel` | Cancel any transfer |
| `PATCH /settings` | Update settings |
| `POST /guest/invite` | Create a one-time guest pairing token and QR payload |
| `POST /guest/{id}/approve` | Approve a guest connection |
| `GET /diagnostics` | Network interfaces, discovery status, recent errors |
| `GET /events` (WebSocket) | Live events (below) |

For browser-based UIs that cannot hand over a filesystem path, the control API accepts streamed uploads (`PUT /send/upload/{handle}`), and the core forwards bytes to the peer without writing to disk.

### 6.1 WebSocket events

```json
{ "type": "peer.up",        "data": { "fingerprint": "...", "alias": "..." } }
{ "type": "peer.down",      "data": { "fingerprint": "..." } }
{ "type": "offer.incoming", "data": { "sessionId": "...", "sender": {...}, "files": [...] } }
{ "type": "progress",       "data": { "sessionId": "...", "fileId": "...", "bytesDone": 123, "bytesTotal": 456, "bytesPerSec": 7890 } }
{ "type": "transfer.done",  "data": { "sessionId": "...", "status": "completed" } }
{ "type": "transfer.failed","data": { "sessionId": "...", "error": {...} } }
{ "type": "guest.request",  "data": { "guestId": "...", "ua": "...", "code": "4821" } }
```

---

## 7. Guest API (web clients)

Base: `https://<host-ip>:42100/guest/v1`.

| Step | Request | Notes |
|---|---|---|
| 1. Pair | `POST /pair` with `{ "token": "<from QR>" }` | Token is one-time and expires in about 2 minutes. Response includes a 4-digit confirmation code, shown on both host and guest |
| 2. Wait for approval | `GET /pair/status` (long-poll or WS) | Resolves when the host user taps Allow or Deny |
| 3. Receive bearer | Returned on approval | Short-lived, stored in memory (not localStorage) |
| 4. Upload to host | `PUT /upload/{fileId}` (streamed) with `Upload-Offset` | Same resume semantics as the peer API |
| 5. List shared files | `GET /files` | Files the host chose to share with this guest |
| 6. Download | `GET /files/{id}` with `Range` support | |

---

## 8. File handling rules (receiver side)

- Treat all names as hostile: strip directories, reject `..`, drop control characters, handle Windows reserved names (`CON`, `NUL`, ...), cap length.
- Write to `*.part` in the destination directory, rename atomically on success.
- Collisions: append ` (1)`, ` (2)`, and so on.
- Check free disk space before accepting.
- Folder transfers: recreate structure from `relPath` after sanitizing every segment.
- Never execute or open received files automatically.

---

## 9. Performance notes

- One streaming `PUT` per file is usually enough to saturate a Wi-Fi link. **Measure before adding parallel chunking.**
- Parallelism across files (default 3) helps with many small files.
- Use `io.Copy` with a large buffer (256 KiB to 1 MiB) and avoid per-chunk allocations.
- Hash while writing where possible. For resumed transfers, re-hash the existing `.part` content (or verify the whole file once at the end).