# ADR 0002: Streaming Transport via HTTP with Resumable Offsets

## Status
Accepted

## Context
Transferring multi-gigabyte files between devices over local Wi-Fi or Ethernet requires high throughput, resume capability on network drops, flow control, and broad platform compatibility (desktop, mobile, and web browsers).
Alternatives considered:
- Custom binary TCP protocol
- WebRTC DataChannels
- QUIC / HTTP/3
- Standard HTTP/1.1 and HTTP/2 over TLS

## Decision
We choose **standard HTTP/1.1 and HTTP/2 over TLS with streaming body and `Upload-Offset` headers** for LAN file transfers:
1. `PUT /api/v1/send/{sessionId}/{fileId}` streams raw binary data directly from disk to socket on the sender, and socket to `.part` disk file on the receiver, bypassing memory buffering (`io.Copy` with constant bounded memory).
2. Resumability is achieved via `HEAD /api/v1/send/{sessionId}/{fileId}` and the `Upload-Offset` request header. If a transfer drops mid-stream, the sender issues a `HEAD` request to query the receiver's written byte count on the `.part` file, seeks the local file descriptor to that offset, and resumes upload without retransmitting already-received bytes.
3. Integrity is verified upon completion by calculating the full file BLAKE3 or SHA-256 hash before atomically renaming from `.part` to final destination filename.

## Consequences
### Positive
- Works seamlessly across all targets: Go backend, browser fetch/XHR, and mobile WebViews without requiring specialized kernel drivers or third-party binary protocols.
- Native HTTP flow control (TCP windowing / HTTP/2 multiplexing) prevents memory exhaustion and saturates Gigabit Wi-Fi / LAN links.
- Simpler debugging, testing, and proxying using standard HTTP diagnostic tooling.
- Resuming interrupted transfers requires zero protocol state negotiation beyond offset seeking.

### Negative / Trade-offs
- Web browsers cannot currently initiate arbitrary chunked streaming PUT requests easily without modern fetch stream duplex APIs, though downloads (`GET`) and multi-part uploads work reliably.
