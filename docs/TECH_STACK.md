# Tech Stack

Chosen for a **TypeScript-first developer with working knowledge of Go**.

> **Rule of thumb:** Go does all the networking. TypeScript does all the UI. The UI always talks to the Go core over localhost HTTP/WebSocket.

*Library names below were accurate to my knowledge, but ecosystems move. Verify that each one is still maintained before depending on it.*

---

## 1. Summary

| Layer | Choice | Notes |
|---|---|---|
| Core language | **Go 1.22+** | Single static binary, great concurrency, easy cross-compilation |
| HTTP server | stdlib `net/http` (1.22 pattern routing) | `chi` is a fine alternative if you want middleware helpers |
| TLS / certificates | stdlib `crypto/tls`, `crypto/x509` | Self-signed ECDSA P-256, fingerprint pinning |
| mDNS discovery | `github.com/grandcat/zeroconf` (or the `libp2p/zeroconf` fork) | Check maintenance status; wrap behind an interface so it can be swapped |
| Multicast / scan fallback | stdlib `net` | Hand-written, small |
| WebSocket | `github.com/coder/websocket` (formerly `nhooyr.io/websocket`) | Progress events and incoming-request prompts |
| Database | `modernc.org/sqlite` | Pure Go, no CGO, which matters for mobile and cross-compilation |
| Hashing | `github.com/zeebo/blake3` (or stdlib SHA-256) | Integrity check after transfer |
| Optional WebRTC | `github.com/pion/webrtc` | Phase 8 only |
| API spec | **OpenAPI 3** | Single source of truth for Go and TS |
| Go codegen | `oapi-codegen` | Server stubs and types |
| TS codegen | `openapi-typescript` plus `openapi-fetch` | Typed client |
| UI framework | **React or Svelte** (pick one you're fastest in) | Svelte yields smaller bundles; React has more ecosystem |
| Build tool | **Vite** | Fast dev server, simple production build |
| Styling | **Tailwind CSS** | Quick, consistent UI |
| Client state | TanStack Query plus a small WebSocket store (Zustand or Svelte stores) | |
| UI embedding | Go `embed` package | One binary ships core and UI |
| Desktop shell | **Wails** or **Electron plus Go sidecar** | See section 3 |
| Mobile shell | **Capacitor** plus `gomobile bind` | See section 4 |
| Tests (Go) | `testing`, `testify`, `go test -race` | |
| Tests (UI) | Vitest plus Playwright | Playwright also covers the web guest flow |
| CI | GitHub Actions | Matrix build for Win/macOS/Linux |
| Packaging | GoReleaser (CLI/server), Wails or electron-builder (desktop) | |

---

## 2. Why Go for the core

- Goroutines make "one transfer per goroutine, many in parallel" trivial.
- The standard library covers HTTP/2, TLS, X.509, hashing, and streaming I/O.
- Cross-compiles to every desktop OS with a single command.
- `gomobile bind` can package the same core for Android and iOS.
- You already know some Go, so the learning curve is small.

## 3. Desktop shell options

| Option | Pros | Cons | Verdict |
|---|---|---|---|
| **No shell (MVP)** | Zero extra work. Run the binary, it opens `https://localhost:PORT` in the browser | Not a "real" app feel | Use for Phases 1–4 |
| **Wails** | Go backend plus TS frontend, small binaries, native webview | Wails v3 had a long alpha period. Verify current stability | Preferred if stable |
| **Electron plus Go sidecar** | Most mature, huge community | Large install size, extra process | Safe fallback |
| **Tauri** | Small, secure | Rust-based shell, which you'd rather avoid | Not recommended for this skill set |

## 4. Mobile strategy (future)

**Recommended: Capacitor plus gomobile.**

1. Compile the Go core to an Android `.aar` and iOS `.xcframework` using `gomobile bind`.
2. Wrap it in a small Capacitor plugin (Kotlin and Swift) exposing `start()`, `stop()`, and `getPort()`.
3. The Capacitor WebView loads the same TypeScript UI from the in-process Go server.
4. Discovery uses **native APIs** (Android `NsdManager`, iOS Bonjour) through the plugin, because iOS restricts raw multicast and Android needs multicast locks.

**Alternative:** React Native plus gomobile. Same bridge idea, but you rebuild the UI in native components, which gives up code sharing.

**Keep the gomobile bridge tiny.** It only passes simple types, so expose a handful of functions and let everything else flow over localhost HTTP.

## 5. Web guest client

- Same TypeScript UI, built with a `guest` mode flag.
- Served over HTTPS by a host peer at `/g/`.
- Entry point is a QR code or short URL shown by the host.
- Browser APIs used: Fetch streaming, File API, Streams API, File System Access API (Chromium), OPFS, and a StreamSaver-style fallback for other browsers.
- PWA caching is an optional extra. Chrome may refuse to register service workers on pages with an untrusted certificate, so the app must work without it. The host serves the page every time anyway, so it is already offline-capable.

## 6. Decision log

| # | Decision | Reason | Revisit when |
|---|---|---|---|
| D1 | Go core, TS UI | Matches developer skills, one UI codebase | Never, unless performance requires it |
| D2 | HTTPS plus REST for transfer (not raw TCP) | Easy to debug, browsers can speak it, resumable with Range-style offsets | If throughput benchmarks fall short |
| D3 | Self-signed certs with fingerprint pinning | No CA needed offline | If browser UX proves unacceptable (consider a local CA) |
| D4 | Native discovery on mobile | iOS and Android restrict multicast | If Go-based discovery works reliably in testing |
| D5 | SQLite via `modernc.org/sqlite` | No CGO, easy mobile builds | If performance is poor |
| D6 | Web is a guest, not a peer | Browsers cannot listen or do mDNS | If a WebRTC-only mode proves viable |
| D7 | IPv4 first | Fewer edge cases | After v1 |

## 7. Alternatives considered

| Alternative | Why not (for now) |
|---|---|
| Rust core plus Flutter UI | Best long-term fit for pure native apps, but two new languages to learn |
| Node.js core | Weaker for large streaming and cross-compiling a single binary, no `gomobile` equivalent |
| Pure WebRTC architecture | Needs signaling and does not solve discovery. Great for browser-to-browser, but not as the foundation |
| Flutter plus Go via FFI | Doable, but throws away your TypeScript strength |