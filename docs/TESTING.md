# Testing Strategy

Networking software fails in ways that unit tests never reveal. The plan below combines automated tests with a **real-network test matrix**.

---

## 1. Test pyramid

| Level | Tooling | What it covers |
|---|---|---|
| Unit | Go `testing`, `testify`; Vitest | Pure logic: filename sanitizer, state machine, fingerprint, SAS code, formatters |
| Integration (in-process) | Go, loopback, temp dirs | Two core instances exchanging files; resume; cancel; rejection; corruption |
| API contract | OpenAPI validation, generated clients | Go server and TS client stay in sync |
| End-to-end | Playwright plus built binaries | UI flows: send, accept, guest pairing, history |
| Real-network | Manual plus scripted, multiple devices | Discovery, throughput, platform quirks |
| Non-functional | `go test -race`, fuzzing, soak, benchmarks | Concurrency safety, robustness, performance, leaks |

---

## 2. Core test cases

### Transfer engine
- [ ] Send a small file, a zero-byte file, a 1-byte file, and a file exactly at buffer boundaries
- [ ] Send a very large file (5–10 GB) with constant memory use
- [ ] Many small files (10,000) in one session
- [ ] Resume after the connection drops at 0%, 50%, and 99%
- [ ] Resume after the receiver restarts
- [ ] Corrupt one byte in transit: `HASH_MISMATCH` is reported and the file is discarded
- [ ] Wrong declared size: `SIZE_MISMATCH`
- [ ] Receiver rejects; sender sees `REJECTED`
- [ ] Offer timeout expires
- [ ] Receiver is busy: `BUSY`
- [ ] Cancel from the sender, from the receiver, and during verification
- [ ] Disk full mid-transfer: clean failure, `.part` handled
- [ ] Destination folder is not writable
- [ ] Two senders to one receiver at once
- [ ] Same file sent twice: collision rename

### Security
- [ ] Filename `../../etc/passwd`, absolute paths, `C:\Windows\...`, null bytes, very long names, Unicode tricks (RTL override), reserved Windows names
- [ ] Folder transfer with `..` segments (zip-slip style)
- [ ] Wrong fingerprint: client refuses to connect
- [ ] Fingerprint change after trust: warning is shown
- [ ] Expired, reused, or forged guest token rejected
- [ ] Guest cannot reach peer or control endpoints
- [ ] Control API rejects requests with a wrong `Host`, a foreign `Origin`, or no token
- [ ] Oversized request bodies rejected
- [ ] Rate limits work under connection floods
- [ ] XSS payloads in alias and filename render as plain text

### Discovery
- [ ] mDNS announce/browse works between two instances
- [ ] Multicast works with and without mDNS
- [ ] Scanner finds a peer when both other methods are disabled
- [ ] Peers marked offline after the timeout and removed later
- [ ] Interface change (Wi-Fi switch) triggers rediscovery
- [ ] Sleep/wake recovers

---

## 3. Automated integration test pattern (Go)

```go
func TestSendReceive_Resume(t *testing.T) {
    recv := startTestPeer(t, WithAutoAccept())
    send := startTestPeer(t)

    src := writeRandomFile(t, 200<<20) // 200 MiB
    // Drop the connection after about 40% of the bytes
    err := send.SendFile(ctxWithFault(DropAfter(80<<20)), recv.Peer(), src)
    require.Error(t, err)

    // Resume
    require.NoError(t, send.SendFile(context.Background(), recv.Peer(), src))

    require.Equal(t, sha256File(t, src), sha256File(t, recv.Received(src)))
}
```

Use a fault-injecting `net.Conn` wrapper or `io.Reader` to simulate drops, slow links, and corruption.

---

## 4. UI and end-to-end (Playwright)

Scenarios:
1. Launch two binaries (different config dirs and ports), open both UIs, verify each sees the other.
2. Send a file from A to B, accept on B, verify the received file hash.
3. Reject flow and cancel flow.
4. Guest flow: A creates an invite, a second browser context opens the guest URL (ignoring the certificate error in test config), confirmation codes match, A approves, the guest uploads, A receives.
5. Reload the UI mid-transfer: state recovers through WebSocket and the API.
6. Keyboard-only run through the main flow (accessibility smoke test).

Add axe-core checks for accessibility in CI.

---

## 5. Real-network test matrix

Fill this in as you test. Each cell records pass/fail and notes.

| Network type | mDNS | Multicast | Scan | Manual IP | Throughput (MB/s) | Notes |
|---|---|---|---|---|---|---|
| Home router (5 GHz) | | | | | | |
| Home router (2.4 GHz) | | | | | | |
| Ethernet ↔ Wi-Fi bridge | | | | | | |
| Phone hotspot | | | | | | |
| Android local-only hotspot | | | | | | |
| Guest Wi-Fi (isolated) | | | | | | Expected to fail; verify messaging |
| Corporate/campus Wi-Fi | | | | | | |
| VPN active on one device | | | | | | |
| Router with no internet | | | | | | Core offline claim |

### Device combinations

| Sender → Receiver | Windows | macOS | Linux | Android | iOS | Web guest |
|---|---|---|---|---|---|---|
| Windows | | | | | | |
| macOS | | | | | | |
| Linux | | | | | | |
| Android | | | | | | |
| iOS | | | | | | |
| Web guest | | | | | | n/a |

### Browser matrix (guest)

Chrome (desktop, Android), Edge, Safari (macOS, iOS), Firefox (desktop, Android). Test: pairing, 100 MB upload, 2 GB upload, download, certificate-warning flow, tab backgrounding.

---

## 6. Performance targets

Measure and record rather than guess. Suggested goals:

| Metric | Target |
|---|---|
| Throughput | At least about 80–90% of the link's measured `iperf3` rate on the same pair of devices |
| Memory | Roughly constant (tens of MB) regardless of file size |
| Discovery time | New peer visible within about 3 s on a typical home network |
| Time to first byte after Accept | Under 1 s |
| Many small files | No more than a modest overhead per file (profile, then optimize) |
| CPU while idle | Near zero (announce timers only) |

Tools: `iperf3` baseline, `pprof` (CPU, heap, goroutines), `go test -bench`, and `tc netem` (Linux) to simulate latency, jitter, and loss.

---

## 7. Robustness

- **Fuzzing:** Go native fuzzing on the filename sanitizer, JSON request decoding, and multicast datagram parsing.
- **Race detector:** run `go test -race ./...` in CI always.
- **Soak test:** 24 hours of repeated transfers with random cancels and reconnects. Watch goroutine count, open file handles, and RSS.
- **Chaos:** randomly kill the peer process mid-transfer; kill the network (disable Wi-Fi) mid-transfer.

---

## 8. Offline verification

A dedicated CI job proves the app is truly offline:
1. Run the binary in a network namespace (or container) with **no default route and no DNS**.
2. Run the transfer integration tests between two instances inside the namespace.
3. Capture connection attempts (`ss`, `conntrack`, or `tcpdump`) and fail the job if anything tries to leave the local subnet.

---

## 9. Mobile testing

- Use real devices, not just emulators. Emulators handle multicast and Wi-Fi unrealistically.
- Android: at least one Pixel/stock device and one aggressive-battery OEM device (for example Xiaomi or Samsung). Test with screen off and Doze.
- iOS: test on at least two iOS versions; test denied and granted Local Network permission; test backgrounding mid-transfer.
- Test share-sheet and share-extension flows with Photos, Files, and a browser.

---

## 10. Release checklist

- [ ] All automated suites pass on Windows, macOS, and Linux
- [ ] Real-network matrix filled in for the supported combinations
- [ ] Offline verification job green
- [ ] Security checklist in `SECURITY.md` reviewed
- [ ] Accessibility checks pass
- [ ] Upgrade path tested (config and database migrations from the previous version)
- [ ] Fresh-install test on a clean VM for each OS
- [ ] Release artifacts verified by checksum and signature