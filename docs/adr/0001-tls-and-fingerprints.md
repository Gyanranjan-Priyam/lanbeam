# ADR 0001: Self-Signed TLS with SHA-256 Fingerprint Pinning

## Status
Accepted

## Context
LanBeam facilitates direct peer-to-peer file transfers across local area networks (LANs). In local network environments:
- Public Certificate Authorities (WebPKI / Let's Encrypt) cannot issue certificates for private IP spaces (`192.168.x.x`, `10.x.x.x`) or local hostnames (`.local`) without public DNS control and internet connectivity.
- Unencrypted transport (raw HTTP or TCP) exposes data, filenames, and tokens to passive eavesdropping, packet sniffing, and tampering on shared LANs (e.g. coffee shops, public Wi-Fi, offices).
- Setting up a custom local Certificate Authority (CA) requires root/admin privilege and OS keystore installation on every user machine and mobile phone, creating excessive friction for casual file transfers.

## Decision
We choose **self-signed ECDSA P-256 certificates with out-of-band SHA-256 certificate fingerprint pinning**:
1. Every LanBeam node generates a persistent ECDSA P-256 private key and self-signed X.509 certificate on initial setup.
2. The device identity is defined by the SHA-256 digest of the DER-encoded certificate (`Fingerprint`).
3. Outbound peer TLS connections use `InsecureSkipVerify: true` coupled with a mandatory `VerifyPeerCertificate` callback:
   - On connecting to a known peer, the client verifies that the server's leaf certificate matches the pinned fingerprint stored in the peer registry. If it does not match, connection is aborted immediately.
   - On first contact with an unknown peer, the fingerprint is verified out-of-band via user confirmation or a Short Authentication String (SAS / emoji / QR code) before data transfer occurs.
4. Per-session bearer tokens authenticate individual file transfer requests within the encrypted channel.

## Consequences
### Positive
- Fully zero-configuration: works 100% offline without internet access or domain names.
- Zero administrative permissions required: no root certificates need to be installed into OS trust stores.
- Strong cryptographic encryption in transit prevents passive eavesdropping.
- Mutual out-of-band verification prevents Man-in-the-Middle (MitM) attacks.

### Negative / Trade-offs
- Browsers connecting in guest mode (`/g/...`) will show a self-signed TLS warning unless accessed via HTTP or paired via trusted QR code. Web guest architecture mitigates this by allowing local HTTP paired access with temporary session tokens when necessary.
