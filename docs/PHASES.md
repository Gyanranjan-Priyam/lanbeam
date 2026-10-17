# Project Phases

Nine phases, from foundations to public release. Each phase has a goal, tasks, deliverables, and **exit criteria**: do not start the next phase until the exit criteria are met.

> **About the estimates:** durations assume one focused developer working close to full time. If you work part-time, expect roughly 2–3x. The mobile phases (6 and 7) are the least predictable, so pad them.

---

## Timeline overview

| Phase | Name | Est. duration | Output |
|---|---|---|---|
| 0 | Foundations and spec | 1 week | Repo, tooling, protocol draft, OpenAPI skeleton |
| 1 | Core transfer engine | 2 weeks | Go binary that sends/receives files between two machines |
| 2 | Discovery | 1–2 weeks | Peers appear automatically, with fallbacks |
| 3 | Local UI | 2–3 weeks | Full desktop-in-browser experience, embedded in the binary |
| 4 | Web guest and QR flow | 1–2 weeks | Phone browser can exchange files with a host |
| 5 | Desktop packaging | 1–2 weeks | Installable apps for Windows, macOS, Linux |
| 6 | Android app | 3–4 weeks | Android full peer |
| 7 | iOS app | 3–4 weeks | iOS full peer (foreground) |
| 8 | Advanced features | ongoing | Folders, clipboard, WebRTC mode, i18n, polish |
| 9 | Hardening and release | 2 weeks | Security review, docs, signed releases |

Phase 8 can overlap with Phases 6–7 for desktop-only features.

---

## Phase 0: Foundations and spec

**Goal:** Remove ambiguity before writing real code.

**Tasks**
- [ ] Choose a final project name; check name/domain/trademark conflicts
- [ ] Create the monorepo (`core`, `ui`, `packages/protocol`, `mobile`, `docs`)
- [ ] Set up Go (1.22+), Node (LTS), pnpm or npm, and an editor with linting
- [ ] Add formatters and linters: `gofmt`, `golangci-lint`, ESLint, Prettier
- [ ] Add GitHub Actions: build and test on Linux, macOS, Windows
- [ ] Finalize `PROTOCOL.md` v1 and convert it to `openapi.yaml`
- [ ] Set up code generation scripts (`oapi-codegen`, `openapi-typescript`)
- [ ] Decide the license; add `LICENSE` and `CONTRIBUTING.md`
- [ ] Write the first three Architecture Decision Records (ADRs)

**Deliverables:** Empty-but-building monorepo, CI green, OpenAPI file, ADRs.

**Exit criteria:** `make build` (or equivalent) succeeds on all three desktop OSes in CI.

---

## Phase 1: Core transfer engine

**Goal:** Two machines on the same network can transfer a file securely using only the Go core.

**Tasks**
- [ ] Identity: generate/load ECDSA P-256 key and self-signed cert; compute fingerprint
- [ ] TLS server with the LAN listener; configurable port with fallback
- [ ] `GET /api/v1/info`
- [ ] `POST /api/v1/prepare-send` with an accept/reject hook (CLI prompt for now)
- [ ] `PUT /api/v1/send/{session}/{file}` streaming to `.part`, rename on completion
- [ ] `HEAD` endpoint and `Upload-Offset` resume support
- [ ] Sender client with fingerprint pinning (`VerifyPeerCertificate`)
- [ ] Safe filename and path handling plus collision renaming
- [ ] Integrity check (BLAKE3 or SHA-256) after transfer
- [ ] Session state machine and cancel support
- [ ] Minimal CLI: `lanbeam send <ip> <file>` and `lanbeam receive`
- [ ] Unit and integration tests (two in-process instances over loopback)

**Deliverables:** Working CLI; test suite; throughput benchmark.

**Exit criteria**
- Transfer a 5 GB file between two real machines with constant memory use
- Kill the connection mid-transfer and resume successfully
- Hash mismatch is detected when you deliberately corrupt a byte
- `go test -race ./...` passes

---

## Phase 2: Discovery

**Goal:** Devices find each other without typing IPs.

**Tasks**
- [ ] Define the `Discoverer` interface and the peer registry
- [ ] mDNS announce and browse
- [ ] UDP multicast announce and listen
- [ ] Subnet scanner with a bounded worker pool
- [ ] Manual IP entry support
- [ ] Network interface selection (ignore loopback, VPN, and virtual adapters by default)
- [ ] Peer liveness (online/offline timeouts) and de-duplication by fingerprint
- [ ] Persist known devices in SQLite
- [ ] Diagnostics endpoint: interfaces, method status, last error

**Deliverables:** `lanbeam peers` lists nearby devices; events published on the bus.

**Exit criteria**
- Discovery works on: home router, phone hotspot, Windows-to-macOS-to-Linux combinations
- On a network that blocks multicast, scan or manual fallback still finds the peer
- Peer appears within about 3 seconds of launch on a typical home network

---

## Phase 3: Local UI

**Goal:** A polished UI that exercises the entire core, served by the Go binary.

**Tasks**
- [ ] Set up the Vite plus TypeScript project, Tailwind, routing
- [ ] Generate the typed API client from OpenAPI
- [ ] Control API: peers, send, accept/reject, transfers, settings, diagnostics
- [ ] WebSocket events client with reconnect
- [ ] Screens: Nearby devices, Send, Incoming offer, Progress, History, Settings, Diagnostics
- [ ] Drag-and-drop and file picker; folder picker where supported
- [ ] Streamed upload path for browser-origin files (no temp copies)
- [ ] Embed the built UI with `go:embed`; SPA fallback routing
- [ ] Control API hardening (token, `Host`/`Origin` checks)
- [ ] Empty states, error states, and loading states (see `UX_DESIGN.md`)
- [ ] Dark mode and basic accessibility (keyboard, contrast, labels)

**Deliverables:** Single binary that opens the full UI.

**Exit criteria**
- A new user can send a file to another device in under 30 seconds with no instructions
- Progress, speed, and ETA are accurate; cancel works
- UI survives core restart and WebSocket reconnect

---

## Phase 4: Web guest and QR flow

**Goal:** Any phone or laptop with a browser can exchange files with a host, with nothing to install.

**Tasks**
- [ ] Host creates a guest invite: token plus QR code (`/g/#t=...`)
- [ ] Guest API: pair, approval status, upload, list, download
- [ ] Host approval prompt with the 4-digit confirmation code
- [ ] Guest UI build mode (`guest` flag): simplified send/receive screens
- [ ] Large-file support: streamed upload, File System Access API where available, StreamSaver-style fallback elsewhere
- [ ] Certificate warning onboarding screen with browser-specific instructions
- [ ] Token expiry and revocation; guest session list in host UI
- [ ] Optional service worker or PWA manifest (only as an enhancement)
- [ ] Browser test matrix (see `TESTING.md`)

**Deliverables:** Phone scans QR, approves on the host, and sends and receives files.

**Exit criteria**
- Works on Chrome/Edge (desktop and Android) and Safari (iOS/macOS)
- 2 GB file upload from a phone browser succeeds without crashing the tab
- Revoking a guest immediately blocks further requests

---

## Phase 5: Desktop packaging

**Goal:** Real installable apps with a native feel.

**Tasks**
- [ ] Evaluate Wails (check current stability) versus Electron plus Go sidecar; record the decision as an ADR
- [ ] System tray/menu bar icon, "receive" status, quick-send
- [ ] Auto-start on login (optional)
- [ ] Native file dialogs and "Open folder" actions
- [ ] OS integration: "Share/Send to" context menu where feasible
- [ ] Firewall prompts and first-run guidance (Windows, macOS, Linux)
- [ ] Installers: MSI/EXE, DMG, AppImage/deb/rpm
- [ ] Code signing and macOS notarization (needs paid developer accounts)
- [ ] Auto-update strategy that does **not** break the offline promise (manual download or opt-in LAN update check)

**Deliverables:** Installable builds for all three desktop OSes via CI.

**Exit criteria**
- Clean-machine install to first successful transfer in under 3 minutes
- No scary OS security warnings on signed builds

---

## Phase 6: Android app

**Goal:** Android full peer with native discovery and background-safe transfers.

**Tasks**
- [ ] Set up Capacitor project; load the existing UI
- [ ] `gomobile bind` the Go core into an `.aar`; keep the bridge to `Start/Stop/Port`
- [ ] Capacitor plugin (Kotlin) hosting the Go core
- [ ] Native discovery with `NsdManager`; hold a multicast lock; feed peers to the core
- [ ] Foreground service with a notification during transfers (verify current Android foreground-service type rules and limits)
- [ ] Share-sheet integration: "Share to LanBeam" from other apps
- [ ] Storage: Storage Access Framework / MediaStore for saving received files
- [ ] Permissions flow and Wi-Fi/hotspot guidance
- [ ] Optional: Local-only hotspot and Wi-Fi QR join
- [ ] Battery and Doze behaviour testing on multiple vendors (Samsung, Xiaomi, and others are aggressive)

**Deliverables:** Signed APK/AAB and Play-Store-ready build.

**Exit criteria**
- Phone ↔ desktop transfers both directions at near Wi-Fi line rate
- Transfer continues with the screen off (within platform limits)
- Share-sheet send works from Gallery and Files apps

---

## Phase 7: iOS app

**Goal:** iOS full peer (foreground-first).

**Tasks**
- [ ] `gomobile bind` to an `.xcframework`; Capacitor Swift plugin
- [ ] Declare `NSLocalNetworkUsageDescription` and `NSBonjourServices` (`_lanbeam._tcp`)
- [ ] Native Bonjour browse/advertise via `NWBrowser`/`NWListener` (or `NetService`); feed the core
- [ ] Handle the Local Network permission prompt gracefully
- [ ] Share extension and Files app integration
- [ ] Foreground-only transfers with a clear "keep the app open" UX; evaluate `beginBackgroundTask` for short grace periods
- [ ] TestFlight distribution; App Store review preparation (explain local-network use)

**Deliverables:** TestFlight build.

**Exit criteria**
- iPhone ↔ desktop and iPhone ↔ Android transfers succeed
- Permission-denied state is understandable and recoverable

---

## Phase 8: Advanced features

**Goal:** Differentiate from existing tools.

**Candidates (prioritize by user feedback)**
- [ ] Folder transfer with structure
- [ ] Text and clipboard sharing
- [ ] Guest relay: web guest sends to other peers through the host
- [ ] WebRTC DataChannel mode (`pion/webrtc`) for browser-to-browser, with the host as signaling
- [ ] Parallel chunking if benchmarks show a benefit
- [ ] Transfer queue, pause/resume UI
- [ ] Trusted-device auto-accept
- [ ] Wi-Fi QR sharing and in-app hotspot helper
- [ ] Localization (i18n) and RTL support
- [ ] Optional local CA mode to remove browser certificate warnings
- [ ] Image previews and received-file quick actions

**Exit criteria:** Each feature ships behind a setting with tests and docs.

---

## Phase 9: Hardening and release

**Goal:** Something you'd trust with your own files.

**Tasks**
- [ ] Full security checklist pass (`SECURITY.md`)
- [ ] Fuzz the JSON parsers and the filename sanitizer
- [ ] Soak test: 24-hour repeated transfers, check for memory/handle leaks
- [ ] Verify offline claim: run the app with no internet and confirm zero outbound connections
- [ ] Accessibility audit
- [ ] Write user docs: install, first transfer, troubleshooting (AP isolation, firewalls, VPNs)
- [ ] Privacy policy (short and true)
- [ ] Signed, reproducible release pipeline and checksums
- [ ] Beta program with real users on varied networks
- [ ] Decide on distribution: GitHub Releases, app stores, F-Droid, Flathub, winget, Homebrew

**Exit criteria:** Open-issues triage complete, no known data-loss or security bugs, v1.0 tag.

---

## Dependency map

```
0 ──► 1 ──► 2 ──► 3 ──┬──► 4 ──► 8
                      ├──► 5
                      └──► 6 ──► 7
                  All ──► 9
```

Phases 4, 5, and 6 are independent of each other once Phase 3 is done. If you are solo, do 4, then 5, then 6, then 7.

---

## Milestones (suggested releases)

| Version | Contents |
|---|---|
| `v0.1` | Phases 0–2: CLI, discovery |
| `v0.2` | Phase 3: local UI |
| `v0.3` | Phase 4: web guest |
| `v0.5` | Phase 5: desktop installers (first public alpha) |
| `v0.8` | Phase 6: Android beta |
| `v0.9` | Phase 7: iOS TestFlight |
| `v1.0` | Phase 9: hardened release |