# LanBeam (working name)

> Offline-first file transfer over your local network. Think **LocalSend + AirDrop + Quick Share**, with a **zero-install web client**. No internet, no accounts, no cloud.

*"LanBeam" is a placeholder. Rename it, and check that the name and domain are free before you publish anything.*

---

## 1. Vision

Send any file, folder, text, or clipboard content between any two devices on the same Wi-Fi network (or a phone hotspot, or a router with no internet) with one tap and no setup.

## 2. Goals

- **Fully offline.** Nothing ever leaves the local network. No STUN/TURN, no relay servers, no analytics, no accounts.
- **Cross-platform.** Windows, macOS, Linux, Android, iOS, and any modern browser.
- **Fast.** Limited by the Wi-Fi link, not by the app.
- **Private and secure.** Encrypted transport, explicit receiver approval, device identity via certificate fingerprints.
- **Simple UX.** Nearby devices appear automatically; transfer takes two taps.
- **One codebase per concern.** Go for the core, TypeScript for every UI.

## 3. Non-goals (for v1)

- Interoperating with Apple AirDrop or Google Quick Share. Both are closed protocols (AWDL, BLE, Wi-Fi Direct). "Hybrid" here means matching their UX, not their wire protocol.
- Transfer over the internet or across different networks.
- Cloud sync, accounts, or backup.
- IPv6-only networks (IPv4 first; IPv6 later).
- Background transfers on iOS (the platform restricts this heavily).

## 4. Platform roles

| Platform | Role | Can discover peers | Send | Receive | Planned phase |
|---|---|---|---|---|---|
| Windows / macOS / Linux | Full peer | Yes | Yes | Yes | 1–5 |
| Android | Full peer | Yes (native NSD plus Go fallbacks) | Yes | Yes | 6 |
| iOS | Full peer (foreground) | Yes (Bonjour) | Yes | Yes | 7 |
| Web browser | Guest (via QR/URL to a host peer) | No | Via host | Via host | 4 |

## 5. Guiding principles

1. **The Go core owns all networking.** UIs only talk to the core over local HTTP/WebSocket.
2. **Always have a fallback.** Discovery has four layers (mDNS, multicast, subnet scan, manual IP/QR) because networks are hostile.
3. **Stream, never buffer.** Constant memory use regardless of file size.
4. **Receiver is always in control.** No file is written without approval (unless the sender is explicitly trusted).
5. **Spec first.** The protocol and API are documented and versioned before code is written.

## 6. Document index

| File | Purpose |
|---|---|
| [`docs/TECH_STACK.md`](docs/TECH_STACK.md) | Technology choices, alternatives, and decision log |
| [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) | System design, components, data model, state machines |
| [`docs/PROTOCOL.md`](docs/PROTOCOL.md) | Discovery and transfer protocol, API reference |
| [`docs/SECURITY.md`](docs/SECURITY.md) | Threat model, identity, pairing, hardening |
| [`docs/PHASES.md`](docs/PHASES.md) | All project phases with tasks, deliverables, and exit criteria |
| [`docs/IMPLEMENTATION_PLAN.md`](docs/IMPLEMENTATION_PLAN.md) | Step-by-step build guide with code skeletons and first sprints |
| [`docs/PLATFORM_NOTES.md`](docs/PLATFORM_NOTES.md) | Per-platform requirements, permissions, and limitations |
| [`docs/UX_DESIGN.md`](docs/UX_DESIGN.md) | Screens, flows, states, and microcopy |
| [`docs/TESTING.md`](docs/TESTING.md) | Test strategy, network matrix, performance targets |
| [`docs/RISKS.md`](docs/RISKS.md) | Risk register and mitigations |
| [`docs/adr/`](docs/adr/) | Architecture Decision Records (ADRs) |

## 7. Quickstart

### Prerequisites
- **Go**: 1.24+ (1.27+ supported)
- **Node.js**: v20+ / v24+
- **bun**: package manager

### Build and Run
```bash
# Install dependencies
bun install

# Generate TypeScript & Go code from OpenAPI specification
make gen

# Build UI and core Go binary
make build

# Run unit tests across core Go modules and UI
make test

# Start UI development server
make dev-ui

# Start Core Go daemon
make dev-core
```

## 8. Status

- **Phase 0 (Foundations and spec):** Completed ✅
- **Phase 1 (Core transfer engine):** In progress 🚀

## 9. License

MIT License — see [LICENSE](LICENSE) for details.