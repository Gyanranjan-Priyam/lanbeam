# Security

Offline does not mean safe. Anyone on the same Wi-Fi (a café, a campus, a hotel) can reach your LAN port. This document defines what we protect, from whom, and how.

---

## 1. Assets

- The user's files in transit and at rest in the download folder
- The user's filesystem (no unwanted writes, no path escapes)
- Device identity (private key)
- User privacy (device name, transfer history)

## 2. Threat model

| # | Threat | Example | Mitigation |
|---|---|---|---|
| T1 | Eavesdropping on LAN | Another device sniffs Wi-Fi traffic | TLS on all peer and guest traffic |
| T2 | Man-in-the-middle | Attacker impersonates a peer | Certificate fingerprint pinning, plus a visual verification code on first pairing |
| T3 | Unsolicited file push (spam) | Stranger sends unwanted files | Receiver approval required for every offer; rate limiting; block list |
| T4 | Malicious filenames | `../../.bashrc`, `CON`, huge names | Strict sanitization and path confinement (`PROTOCOL.md` section 8) |
| T5 | Disk exhaustion | Offer a 500 GB file | Free-space check before accept, per-session size limit |
| T6 | Unauthorized guest access | Random LAN device opens the guest URL | One-time token, short expiry, host approval, confirmation code |
| T7 | Local web attacks on the control API | A malicious website calls `127.0.0.1:42101` (CSRF, DNS rebinding) | Startup bearer token, `Host`/`Origin` validation, no permissive CORS |
| T8 | XSS via device alias or filename | Alias `<script>...` | Escape all untrusted strings, strict CSP, no `innerHTML` |
| T9 | Resource exhaustion (DoS) | Flood of connections or offers | Connection limits, request body limits, per-IP rate limits, offer queue cap |
| T10 | Stolen private key | Malware reads the key file | File permissions `0600`, OS keystore where available (later phase) |
| T11 | Zip-slip style folder attack | Folder transfer with `../` segments | Sanitize every path segment; verify the resolved path stays inside the destination |
| T12 | Silent exfiltration by the app itself | Accidental outbound calls | No analytics, no update pings; CI test that blocks all non-LAN egress |

**Out of scope for v1:** a fully compromised device, nation-state adversaries, and malicious Wi-Fi infrastructure that actively blocks traffic (denial of service).

---

## 3. Device identity

- On first launch, generate an **ECDSA P-256** key pair and a self-signed X.509 certificate.
- **Fingerprint** = SHA-256 of the certificate's DER bytes, as lowercase hex. This is the device ID.
- Store the key at a permission-restricted path (`0600`). Consider the OS keystore later (Windows DPAPI, macOS Keychain, Android Keystore, iOS Keychain).
- Certificates are long-lived (for example 10 years). The identity is the key, not the certificate's dates.
- Regenerating the identity ("reset device") makes the device look new to all peers.

## 4. Trust model (TOFU plus verification)

1. **First contact.** The client sees a new fingerprint. The UI shows the peer alias and a **verification code**. Both devices display the same code.
2. **Verification code (SAS).** Derive from both fingerprints so each side computes the same value:
   `code = first 6 decimal digits of SHA-256(sorted(fpA, fpB) concatenated)`.
   The user compares the codes on both screens. A man-in-the-middle would present a different certificate to each side, so the codes would differ.
3. **Trust levels:**
   - *Unknown:* every transfer needs explicit approval.
   - *Trusted:* user marked "always accept from this device" (optional setting).
   - *Blocked:* connections refused.
4. **Pinning.** Once known, the client verifies the peer's certificate fingerprint on every connection, and refuses on mismatch with a clear warning ("This device's identity changed").

Comparing the code is optional for casual use, but the UI should encourage it on first pairing and always allow it.

## 5. Transport

- TLS 1.2 minimum, TLS 1.3 preferred.
- Restrict cipher suites to the Go defaults (which are sensible).
- Because certificates are self-signed, normal CA validation is replaced by **fingerprint verification** in `VerifyPeerCertificate`. Never ship a client that simply disables verification without pinning.

## 6. Web guest security

- Browsers cannot pin certificates. On first visit the user must accept the self-signed certificate warning. Document this clearly in the UI and onboarding.
- **Pairing token:** 128-bit random value, single use, expires in about 2 minutes, carried in the URL fragment (`#t=...`) so it is not sent in HTTP request lines or logs.
- **Confirmation code:** a 4-digit code displayed on both the host and the guest page. The host user taps Allow only if they recognize the device and the codes match.
- **Bearer token:** issued after approval, held in memory only, expires after inactivity (for example 30 minutes), revocable from the host UI.
- Guests can only reach `/guest/v1/*`. They can never call peer or control APIs.
- Limit guest upload size and concurrent uploads.

## 7. Control API hardening

- Bind to `127.0.0.1` only.
- Random per-run token, delivered to the UI through the shell (Wails/Electron/Capacitor injection) or embedded in the URL printed/opened at startup.
- Reject any request whose `Host` header is not `127.0.0.1:<port>` or `localhost:<port>`.
- Reject cross-origin requests; do not enable permissive CORS.
- Do not expose the control listener on mobile beyond the app's own WebView.

## 8. Input handling checklist

- [ ] Request body size limits on every endpoint (`http.MaxBytesReader`)
- [ ] JSON decoding with `DisallowUnknownFields` where appropriate
- [ ] Filename and path sanitization (see `PROTOCOL.md`)
- [ ] Limit on files per offer (for example 10,000) and total offer size
- [ ] Timeouts: `ReadHeaderTimeout`, idle timeouts, per-transfer stall timeout
- [ ] Panic recovery middleware
- [ ] Escape all strings in the UI; add a strict Content-Security-Policy to served pages

## 9. Privacy

- No telemetry, no crash reporting that leaves the device, no update checks that call out.
- Logs stay local; avoid logging full file paths at default log level.
- Let the user hide their real device name (custom alias).
- Add a CI check (for example running tests in a network namespace with no default route) to prove the app works with zero internet and makes no outbound connections.

## 10. Hardening backlog (post-v1)

- Optional app-level encryption with a PAKE-based pairing (for example SPAKE2) for stronger guest authentication
- OS keystore integration for the private key
- Signed releases and reproducible builds
- Optional local CA mode to remove browser certificate warnings
- Third-party security review before a public 1.0