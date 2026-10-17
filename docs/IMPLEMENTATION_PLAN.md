# Implementation Plan

A practical, step-by-step build guide. Where `PHASES.md` says *what* to deliver, this document says *how* to start building it.

Code snippets are **skeletons to learn from and adapt**, not finished production code. Compile and test them before relying on them.

---

## 1. Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.22 or newer | Core (needed for `net/http` pattern routing and `http.FileServerFS`) |
| Node.js | Current LTS | UI build |
| bun| latest | Package manager |
| Git | any | Version control |
| `golangci-lint` | latest | Go linting |
| `oapi-codegen` | latest | Go code from OpenAPI |
| Wireshark | optional | Debug discovery and TLS |
| `iperf3` | optional | Baseline network throughput |

**Test hardware (minimum):** two computers on the same network. Add an Android phone for Phase 4, and an iPhone later.

---

## 2. Repository bootstrap

```bash
mkdir lanbeam && cd lanbeam && git init

mkdir -p core/cmd/lanbeam \
         core/internal/{config,crypto,discovery,peer,transfer,api,store,events,uiassets} \
         ui packages/protocol mobile docs

# Go module
cd core
go mod init github.com/<you>/lanbeam/core
cd ..

# UI
cd ui
npm create vite@latest . -- --template react-ts   # or svelte-ts
npm install
cd ..
```

Add a root `Makefile` (or `justfile`/`Taskfile`) with targets: `gen`, `build-ui`, `build-core`, `test`, `lint`, `dev`.

---

## 3. Build order for the Go core (Phase 1)

### Step 1: Identity (`internal/crypto`)

```go
package crypto

import (
    "crypto/ecdsa"
    "crypto/elliptic"
    "crypto/rand"
    "crypto/sha256"
    "crypto/tls"
    "crypto/x509"
    "crypto/x509/pkix"
    "encoding/hex"
    "math/big"
    "time"
)

type Identity struct {
    Cert        tls.Certificate
    Fingerprint string // hex SHA-256 of DER
}

func NewIdentity() (*Identity, error) {
    key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
    if err != nil {
        return nil, err
    }
    serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 127))
    if err != nil {
        return nil, err
    }
    tmpl := &x509.Certificate{
        SerialNumber:          serial,
        Subject:               pkix.Name{CommonName: "lanbeam"},
        NotBefore:             time.Now().Add(-time.Hour),
        NotAfter:              time.Now().AddDate(10, 0, 0),
        KeyUsage:              x509.KeyUsageDigitalSignature,
        ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
        BasicConstraintsValid: true,
    }
    der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
    if err != nil {
        return nil, err
    }
    sum := sha256.Sum256(der)
    return &Identity{
        Cert:        tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key},
        Fingerprint: hex.EncodeToString(sum[:]),
    }, nil
}
```

To do next: persist the key and cert to disk (PEM, file mode `0600`), load on startup, and write a test that the fingerprint is stable across restarts.

### Step 2: Pinned TLS client

```go
func PinnedTLSConfig(expectedFP string) *tls.Config {
    return &tls.Config{
        MinVersion: tls.VersionTLS12,
        // Normal CA verification cannot work for self-signed certs, so we skip it
        // and verify the fingerprint ourselves below. Do NOT remove the callback.
        InsecureSkipVerify: true,
        VerifyPeerCertificate: func(raw [][]byte, _ [][]*x509.Certificate) error {
            if len(raw) == 0 {
                return errors.New("no peer certificate")
            }
            sum := sha256.Sum256(raw[0])
            if hex.EncodeToString(sum[:]) != expectedFP {
                return errors.New("peer fingerprint mismatch")
            }
            return nil
        },
    }
}
```

For first contact (unknown device), use a variant that **captures** the fingerprint, shows it to the user, and only continues if they confirm.

### Step 3: Server skeleton (`internal/api/peerapi`)

```go
func NewMux(s *Service) *http.ServeMux {
    mux := http.NewServeMux()
    mux.HandleFunc("GET /api/v1/info", s.handleInfo)
    mux.HandleFunc("POST /api/v1/prepare-send", s.handlePrepareSend)
    mux.HandleFunc("PUT /api/v1/send/{session}/{file}", s.handleSend)
    mux.HandleFunc("HEAD /api/v1/send/{session}/{file}", s.handleOffset)
    mux.HandleFunc("POST /api/v1/cancel/{session}", s.handleCancel)
    return mux
}

srv := &http.Server{
    Addr:              ":42100",
    Handler:           middleware.Chain(mux, middleware.Recover, middleware.Log, middleware.Limit),
    TLSConfig:         &tls.Config{Certificates: []tls.Certificate{id.Cert}, MinVersion: tls.VersionTLS12},
    ReadHeaderTimeout: 10 * time.Second,
    IdleTimeout:       2 * time.Minute,
    // No overall ReadTimeout/WriteTimeout: large transfers would be killed.
    // Use a per-transfer stall timeout instead.
}
log.Fatal(srv.ListenAndServeTLS("", ""))
```

### Step 4: Streaming receive handler (sketch)

```go
func (s *Service) handleSend(w http.ResponseWriter, r *http.Request) {
    sid, fid := r.PathValue("session"), r.PathValue("file")

    f, ok := s.sessions.AuthorizedFile(sid, fid, bearer(r))
    if !ok {
        writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
        return
    }

    offset, _ := strconv.ParseInt(r.Header.Get("Upload-Offset"), 10, 64)
    if offset != f.BytesDone() {
        writeError(w, http.StatusConflict, "BUSY", "offset mismatch")
        return
    }

    out, err := os.OpenFile(f.PartPath, os.O_CREATE|os.O_WRONLY, 0o644)
    if err != nil { /* 500 */ return }
    defer out.Close()
    if _, err := out.Seek(offset, io.SeekStart); err != nil { /* 500 */ return }

    remaining := f.Size - offset
    body := http.MaxBytesReader(w, r.Body, remaining)

    pw := &progressWriter{dst: out, onProgress: f.ReportProgress}
    if _, err := io.Copy(pw, body); err != nil {
        f.MarkFailedResumable(err)
        return
    }

    if f.BytesDone() == f.Size {
        if err := f.VerifyAndFinalize(); err != nil { // hash check, then atomic rename
            writeError(w, http.StatusUnprocessableEntity, "HASH_MISMATCH", err.Error())
            return
        }
    }
    writeJSON(w, http.StatusOK, map[string]any{"fileId": fid, "status": "completed", "savedAs": f.SavedName})
}
```

Notes:
- Hashing during a resumed upload is awkward because the hasher state is lost. The simple approach: **hash the finished file once** by re-reading it, then compare. Optimize later.
- Stream directly from `r.Body`. Never call `io.ReadAll`.
- Add a stall detector: if no bytes arrive for N seconds, cancel the request context.

### Step 5: Sender client (sketch)

```go
func (c *Client) SendFile(ctx context.Context, peer Peer, sess SessionInfo, file LocalFile) error {
    httpc := &http.Client{Transport: &http.Transport{TLSClientConfig: PinnedTLSConfig(peer.Fingerprint)}}

    // 1. HEAD to learn current offset (for resume)
    offset := c.headOffset(ctx, httpc, peer, sess, file)

    // 2. Open the local file and seek
    fh, _ := os.Open(file.Path)
    defer fh.Close()
    fh.Seek(offset, io.SeekStart)

    // 3. Stream
    req, _ := http.NewRequestWithContext(ctx, http.MethodPut, peer.URL("/api/v1/send/"+sess.ID+"/"+file.ID), fh)
    req.ContentLength = file.Size - offset
    req.Header.Set("Authorization", "Bearer "+sess.Token(file.ID))
    req.Header.Set("Upload-Offset", strconv.FormatInt(offset, 10))
    resp, err := httpc.Do(req)
    // handle resp, retry with backoff on transient errors
    return err
}
```

### Step 6: CLI and tests

- `lanbeam receive` starts the server and prompts on stdin to accept offers.
- `lanbeam send <ip>[:port] <paths...>` fetches `/info`, shows the fingerprint, asks to trust it, then sends.
- Integration test: start two `Service` instances on random loopback ports with temp dirs, send a file with random bytes, compare hashes. Add tests for resume, cancel, rejection, and corrupted data.

**Phase 1 is done when** the exit criteria in `PHASES.md` are met.

---

## 4. Discovery implementation (Phase 2)

Define the abstraction first so implementations can be swapped (and so mobile can inject native discovery):

```go
type Peer struct {
    Fingerprint string
    Alias       string
    DeviceType  string
    Addrs       []netip.AddrPort
    LastSeen    time.Time
    Via         string // "mdns" | "multicast" | "scan" | "manual" | "native"
}

type Discoverer interface {
    Start(ctx context.Context, self SelfInfo, out chan<- Peer) error
}
```

Plan:
1. Implement `mdns.Discoverer` using the zeroconf library; verify `/info` over TLS before registering the peer.
2. Implement `multicast.Discoverer` with per-interface sockets and a 5-second announce timer.
3. Implement `scan.Discoverer` with a bounded worker pool (`errgroup` plus a semaphore).
4. A `peer.Registry` merges events by fingerprint, tracks `LastSeen`, and publishes `peer.up`/`peer.down` events.
5. Add `InjectPeer(Peer)` so the mobile native layer can push peers in.

Tips:
- Enumerate interfaces yourself and skip loopback, down, and (by default) virtual or VPN adapters.
- Always verify identity via `/info` before showing a device.
- Log which methods work on each network. It makes bug reports far easier to diagnose.

---

## 5. OpenAPI-first workflow

1. Write `packages/protocol/openapi.yaml` for the Peer, Control, and Guest APIs.
2. Generate Go types and server interfaces with `oapi-codegen`.
3. Generate TypeScript types and a client with `openapi-typescript` plus `openapi-fetch`.
4. Add a CI step that regenerates both and fails if the git diff is non-empty. This guarantees the spec stays authoritative.

WebSocket events are not covered well by OpenAPI. Define them as JSON Schema or a shared TypeScript `types.ts` plus Go structs, and test both against the same fixtures.

---

## 6. UI implementation (Phase 3)

### Structure (`ui/src`)

```
api/           generated client, ws.ts (reconnecting WebSocket)
state/         peers store, transfers store, settings store
components/    PeerCard, DropZone, TransferRow, ProgressBar, OfferDialog, QRCode
screens/       Nearby, Send, History, Settings, Diagnostics
guest/         guest-only screens (built when VITE_MODE=guest)
lib/           format (bytes, speed, ETA), a11y helpers
```

### Build order

1. Typed API client and an auth wrapper that attaches the control token.
2. WebSocket client with exponential backoff; dispatch events into stores.
3. **Nearby** screen: peer cards, live updates, manual-add dialog.
4. **Send** flow: drop zone, picker, selected-files list, send button.
5. **Incoming offer** modal: sender, file list with checkboxes, Accept/Reject, countdown.
6. **Progress** view: per-file bars, speed, ETA, cancel. Smooth with `requestAnimationFrame` and throttle updates.
7. **History**, **Settings**, **Diagnostics**.
8. Accessibility pass, dark mode, responsive layout (the same UI serves phones).

### Browser-origin files

The browser cannot give the core a filesystem path. Stream via `fetch` with `duplex: 'half'` and a `ReadableStream` body (or chunked `PUT` requests on browsers without streaming upload) to the control API, which pipes bytes straight to the target peer. Test this early with multi-GB files.

---

## 7. Embedding the UI in the binary

```go
//go:embed all:dist
var uiFS embed.FS

func UIHandler() http.Handler {
    sub, _ := fs.Sub(uiFS, "dist")
    fileServer := http.FileServerFS(sub)
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // SPA fallback: serve index.html for unknown, non-file routes
        if _, err := fs.Stat(sub, strings.TrimPrefix(path.Clean(r.URL.Path), "/")); err != nil {
            r.URL.Path = "/"
        }
        fileServer.ServeHTTP(w, r)
    })
}
```

Build order: `npm run build` in `ui/`, copy `dist` into `core/internal/uiassets/dist`, then `go build`. Automate in the Makefile. Keep a separate guest build output (`dist-guest`) served under `/g/`.

For development, run Vite's dev server with a proxy to the Go control port for hot reload.

---

## 8. Web guest (Phase 4)

1. Control API `POST /guest/invite` generates a 128-bit token, stores it with an expiry, and returns the URL plus a QR payload.
2. UI renders the QR using a small library (for example `qrcode`).
3. Guest page reads `#t=...`, calls `/guest/v1/pair`, shows the confirmation code.
4. Host UI shows an approval dialog with the same code. On approval, the guest receives a bearer token.
5. Guest upload uses streaming `fetch` where available, with a chunked fallback. Downloads use `Content-Disposition: attachment` links, which stream natively with no memory problem.
6. Add a "certificate warning" help screen reachable from the QR instructions.

---

## 9. Desktop packaging (Phase 5)

- **Quick path:** ship the binary plus a launcher that opens the default browser at the control URL. Good enough for alpha testers.
- **Wails path:** wrap the UI and core in one app, expose native dialogs through Wails bindings, keep the HTTP interface unchanged so web and desktop share code.
- **Electron path:** spawn the Go binary as a child process on a random control port, pass the token through an environment variable, and load the URL in a `BrowserWindow`. Kill the child on quit.
- Use GoReleaser for CLI artifacts and the shell's builder (Wails/electron-builder) for installers.
- Windows: sign with an Authenticode certificate. macOS: Developer ID plus notarization. Without signing, users will see strong warnings.

---

## 10. Mobile (Phases 6 and 7)

### Go façade (`core/mobile`)

Keep it tiny and use only gomobile-compatible types:

```go
package mobile

func Start(configJSON string) (controlPort int, err error)
func Stop()
func InjectPeer(peerJSON string) error   // native discovery pushes peers in
func SetDownloadDir(path string)
```

### Build

```bash
go install golang.org/x/mobile/cmd/gomobile@latest
gomobile init
gomobile bind -target=android -o lanbeam.aar ./mobile
gomobile bind -target=ios     -o Lanbeam.xcframework ./mobile
```

### Capacitor plugin responsibilities

| Concern | Android (Kotlin) | iOS (Swift) |
|---|---|---|
| Host the Go core | Foreground service calls `Start/Stop` | App lifecycle calls `Start/Stop` |
| Discovery | `NsdManager` plus multicast lock → `InjectPeer` | `NWBrowser`/`NWListener` → `InjectPeer` |
| File access | Storage Access Framework / MediaStore | Document picker, security-scoped URLs |
| Share into app | Intent filter `SEND`/`SEND_MULTIPLE` | Share extension |
| Permissions | Wi-Fi state, multicast, notifications, as required by OS version | Local Network usage description, Bonjour services |

Build the Android app first. Verify permission and foreground-service requirements against the **current** Android and iOS documentation before you start, because they change often.

---

## 11. CI/CD

GitHub Actions workflows:

1. **`ci.yml`**: on every PR: lint, `go test -race ./...`, UI tests, codegen drift check, build on Linux/macOS/Windows.
2. **`e2e.yml`**: Playwright against a built binary (two instances, send a file, verify hash; web guest flow).
3. **`release.yml`**: on tag: build artifacts, sign where possible, generate checksums, publish a GitHub Release.
4. **`offline-check.yml`**: run the binary inside a network namespace without a default route and assert the transfer test still passes and no outbound connection attempts occur.

---

## 12. Engineering practices

- **Branching:** trunk-based with short-lived feature branches; squash merge.
- **Commits:** Conventional Commits (`feat:`, `fix:`, `docs:`).
- **Reviews:** even solo, open PRs for yourself so CI runs and history stays tidy.
- **Definition of done** for any feature: code, tests, docs updated, no new lint warnings, works on at least two OSes, error and empty states handled.
- **ADRs:** write one short markdown file per significant decision in `docs/adr/`.
- **Logging:** use `log/slog` with consistent keys (`peer`, `session`, `file`).
- **Dependencies:** pin versions; review licenses (prefer MIT/BSD/Apache-2.0).

---

## 13. First two weeks: day-by-day

| Day | Task |
|---|---|
| 1 | Repo bootstrap, tooling, CI skeleton |
| 2 | Draft OpenAPI for `info`, `prepare-send`, `send`, `cancel` |
| 3 | Identity generation, persistence, fingerprint test |
| 4 | TLS server, `GET /info`, pinned TLS client |
| 5 | Session model and `prepare-send` with CLI accept |
| 6 | Streaming receive handler, `.part` to final rename |
| 7 | Sender client; first end-to-end loopback file transfer |
| 8 | Filename sanitization, collision handling, disk-space check |
| 9 | Resume (`HEAD` plus offset), retry with backoff |
| 10 | Integrity hashing, cancel flow |
| 11 | Integration tests (resume, reject, corruption) |
| 12 | Real two-machine test, 5 GB file, memory profiling |
| 13 | Benchmark with `iperf3` comparison; fix bottlenecks |
| 14 | Cleanup, docs, tag `v0.0.1`, retrospective |

---

## 14. Common pitfalls to avoid

- Setting `http.Server.ReadTimeout`/`WriteTimeout` too low and killing big transfers
- Buffering whole files in memory (on either side or in the browser)
- Trusting mDNS TXT data without verifying over TLS
- Forgetting the Windows firewall prompt, and testing only on machines where it is already allowed
- Testing only on loopback, which hides MTU, Wi-Fi, and multicast problems
- Listening on `0.0.0.0` for the control API
- Letting `localhost` pages call the control API without a token
- Building UI screens before the API contract is stable
- Leaving iOS to the very end: test the Local Network permission and Bonjour behaviour early with a tiny prototype