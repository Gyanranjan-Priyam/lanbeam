# Risk Register

Review this list at the end of every phase. Add new risks as you discover them, and close the ones that stop mattering.

**Scale:** Likelihood (L) and Impact (I) are rated Low / Medium / High.

---

## 1. Technical risks

| # | Risk | L | I | Mitigation | Early warning sign |
|---|---|---|---|---|---|
| R1 | **AP/client isolation** on many Wi-Fi networks stops devices from reaching each other | High | High | Detect and explain clearly; phone-hotspot fallback; Wi-Fi QR flow; troubleshooting guide | Discovery finds nothing and direct probes time out |
| R2 | **Multicast / mDNS blocked** by routers | High | Medium | Four-layer discovery (mDNS, multicast, scan, manual/QR) | Peers only appear via scan |
| R3 | **iOS background limits** pause transfers when the app is not in the foreground | High | Medium | Foreground-first UX; keep screen awake; set expectations in the UI | Transfers stall when the screen locks |
| R4 | **iOS Local Network permission** and Bonjour rules cause discovery failures | Medium | High | Prototype iOS discovery early (before Phase 7); handle denial gracefully | Peers never appear on iOS after granting permission |
| R5 | **Android OEM battery killers** terminate the foreground service | High | Medium | Foreground service plus in-app guidance for battery exemptions; test on aggressive OEMs | Transfers die when the screen turns off on specific brands |
| R6 | **Android foreground-service rules** change between OS versions | Medium | Medium | Re-read current docs at the start of Phase 6; keep the service layer thin | Build or runtime errors on new Android versions |
| R7 | **gomobile limitations** (type restrictions, binary size, build quirks) | Medium | Medium | Keep the bridge tiny; build for Android early in Phase 6 (and a spike even earlier); fall back to a localhost process model if needed | Build breaks or heavy size growth |
| R8 | **Browser certificate warnings** with self-signed certs hurt guest UX; service workers may not register | High | Medium | Onboarding screen with per-browser steps; do not depend on a service worker; consider an optional local CA mode later | Test users abandon at the warning |
| R9 | **Large-file handling in browsers** varies (memory, streaming upload support, save APIs) | Medium | High | Streams, chunked fallback, File System Access/OPFS where available; documented limits per browser | Tab crashes with multi-GB files |
| R10 | **Throughput lower than expected** (small buffers, per-chunk allocations, hashing overhead) | Medium | Medium | Benchmark early against `iperf3`; profile with `pprof`; tune buffers; consider parallel chunking only if measured | Less than half of the link rate |
| R11 | **Library maintenance:** the mDNS library or others become unmaintained | Medium | Medium | Wrap behind interfaces; keep a replacement plan; vendor if needed | No releases, open critical issues |
| R12 | **Wails v3 / shell maturity** issues | Medium | Low | Start with the "binary plus browser" path; keep Electron plus sidecar as a fallback | Build or packaging blockers |
| R13 | **Network changes / sleep-wake** leave stale listeners or announcements | Medium | Medium | Watch interface changes; re-announce and rebind on resume; test on laptops | Peers vanish after the laptop wakes |
| R14 | **Resumable-transfer correctness** bugs (offset races, partial writes) | Medium | High | State-machine tests; fault-injection tests; atomic rename; verify hashes | Hash mismatches after resume |
| R15 | **Memory/handle leaks** in long sessions | Low | Medium | Soak test; `pprof` goroutine/heap checks in CI | RSS grows over time |

## 2. Security risks

| # | Risk | L | I | Mitigation |
|---|---|---|---|---|
| S1 | Path traversal / malicious filenames | Medium | High | Strict sanitizer, confinement checks, fuzzing (`SECURITY.md` T4, T11) |
| S2 | Unsolicited file spam on public Wi-Fi | High | Medium | Approval for every offer; rate limiting; block list; "visibility" toggle (hide from others) |
| S3 | CSRF / DNS rebinding against the control API | Medium | High | Bound to loopback, startup token, `Host`/`Origin` checks |
| S4 | MITM during first pairing | Low | High | Verification code comparison; pinning afterwards |
| S5 | Guest URL opened by an unintended device | Medium | Medium | One-time token in the URL fragment, short expiry, host approval, confirmation code |
| S6 | Private key theft | Low | High | File permissions; OS keystore later; identity reset option |
| S7 | Hidden outbound traffic (dependency, update check) breaks the "offline" promise | Low | High | No telemetry; CI offline-verification job; dependency review |
| S8 | Supply-chain compromise via dependencies | Low | High | Pin versions; review licenses and changes; use `govulncheck` and `npm audit` in CI; signed releases |

## 3. Product and project risks

| # | Risk | L | I | Mitigation |
|---|---|---|---|---|
| P1 | **Crowded space.** LocalSend, Snapdrop-style tools, and OS-native sharing already exist | High | High | Pick a clear differentiator (for example zero-install web guest, cleaner UX, built-in hotspot helper); validate with real users early; do not try to beat the incumbents on features alone |
| P2 | **Scope creep.** Mobile, WebRTC, folders, and i18n all at once | High | High | Follow the phases strictly; ship `v0.5` desktop first; defer Phase 8 items by user demand |
| P3 | **Solo-developer burnout / underestimation** | Medium | High | Re-estimate after Phase 1; cut scope, not quality; release small milestones |
| P4 | **Platform fees and signing costs** (Apple Developer Program, Google Play, Windows code signing) | High | Low | Budget early; start with direct downloads, F-Droid, and TestFlight |
| P5 | **App store rejection** (iOS local-network justification, Android permission usage) | Medium | Medium | Clear usage descriptions; privacy policy; review guidelines before submission |
| P6 | **Name or trademark conflict** | Medium | Medium | Search before branding; avoid names similar to existing apps |
| P7 | **License issues** if code or protocol text is borrowed | Low | High | Write your own protocol; check licenses before reusing anything; track dependency licenses |
| P8 | **Support burden** from network-specific failures that are not bugs | High | Medium | Diagnostics screen, good error messages, troubleshooting docs, "copy report" button |

## 4. Top five to watch first

1. **R1 + R2:** Prove discovery and fallbacks on hostile networks in Phase 2.
2. **R4 + R7:** Do a small iOS and gomobile spike **before** Phase 6 and 7 to avoid late surprises.
3. **R8 + R9:** Test multi-GB browser uploads and the certificate-warning flow early in Phase 4.
4. **P1:** Talk to potential users before investing in mobile. Confirm the differentiator matters.
5. **P2:** Hold the line on the phase plan.

## 5. Spike list (small experiments to de-risk)

| Spike | Time-box | Question it answers |
|---|---|---|
| Go TLS server with pinning between two laptops | 1 day | Is the core transport approach sound? |
| mDNS on a router that filters multicast | 1 day | How often do fallbacks kick in? |
| 2 GB browser upload through a streaming `fetch` | 1 day | Is the web guest viable for big files? |
| `gomobile bind` hello-world on Android and iOS | 2 days | Does the mobile embedding plan hold? |
| iOS Bonjour browse and advertise with Local Network permission | 2 days | Can iOS be a first-class peer? |
| Throughput test: Go `io.Copy` over TLS versus `iperf3` | 0.5 day | What is the performance ceiling? |

## 6. How to use this register

- Add a **status** column (Open, Mitigating, Closed) when you move this into your tracker.
- Review at each phase exit and after any production incident or user complaint.
- If a risk's impact is High and its mitigation is unproven, build the spike first.