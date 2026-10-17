# Platform Notes

Per-platform requirements, quirks, and limitations. OS rules change often, so **verify the specifics against current official documentation** before you implement each phase.

---

## 1. Desktop

### Windows
- First run triggers the **Windows Defender Firewall** prompt for inbound connections. Explain it in onboarding ("Allow on private networks").
- Network profile matters: a network marked *Public* blocks inbound traffic by default. Show a hint if peers cannot connect.
- Multiple adapters (Wi-Fi, Ethernet, Hyper-V, WSL, VPN) are common. Select interfaces carefully and let the user pick.
- Reserved filenames (`CON`, `NUL`, `COM1`...), trailing dots/spaces, and the 260-character path limit need handling.
- Code signing: Authenticode certificate to avoid SmartScreen warnings.

### macOS
- Recent macOS versions require the **Local Network** permission for apps that use mDNS or discover LAN devices. Handle the "denied" state with clear guidance.
- Gatekeeper requires **Developer ID signing plus notarization** for smooth installs.
- Sandboxed (App Store) builds need network client/server entitlements; consider distributing outside the App Store first.
- Menu bar app behaviour (no Dock icon) is a nice touch.

### Linux
- Firewalls (`ufw`, `firewalld`) may block the LAN port and UDP multicast. Provide copy-paste commands in the docs.
- Avahi may or may not be running. Using a pure-Go mDNS implementation avoids needing it, but conflicts with a running Avahi are possible, so test.
- Distribution formats: AppImage, `.deb`, `.rpm`, and later Flatpak (sandbox permissions needed for network and the downloads folder).

### All desktops
- Handle sleep/wake: re-announce and rebind after resume.
- Handle network changes (Wi-Fi switch, IP change): watch for interface changes and restart discovery.
- Choose a sensible default download folder and let the user change it.

---

## 2. Android

| Topic | Notes |
|---|---|
| Hosting the core | Run the Go core inside a **foreground service** with a persistent notification during transfers. Verify current foreground-service type requirements and any runtime limits for the `dataSync` type on the latest Android versions |
| Discovery | Use `NsdManager` (mDNS). Hold a **`WifiManager.MulticastLock`** while discovering or listening for multicast. Feed results to the core through `InjectPeer` |
| Permissions | Network and Wi-Fi state, multicast state change, and notification permission on newer versions. Check the current docs for anything Nearby-related if you add hotspot or Wi-Fi Direct features |
| Saving files | Use **MediaStore / Storage Access Framework**. Scoped storage restricts raw paths on modern Android |
| Receiving shares | Intent filters for `ACTION_SEND` and `ACTION_SEND_MULTIPLE` |
| Battery | OEM battery optimizers (Xiaomi, Oppo, Samsung, and others) may kill background services. Provide in-app guidance to exempt the app |
| Hotspot | `LocalOnlyHotspot` can create a local network with no internet. It is a good offline story, but test across devices |
| Wi-Fi QR | Show `WIFI:T:WPA;S:<ssid>;P:<password>;;` so another phone can join by scanning |
| Cleartext | The WebView loads `http://127.0.0.1`, so allow cleartext **only** for loopback via a network security config |

---

## 3. iOS

| Topic | Notes |
|---|---|
| Local Network permission | Required for Bonjour and LAN connections. Add `NSLocalNetworkUsageDescription` and list your service in `NSBonjourServices` (`_lanbeam._tcp`). Handle denial with a clear explanation and a deep link to Settings |
| Discovery | Use `NWBrowser`/`NWListener` (or `NetService`) natively and feed results to the Go core. Do not rely on raw UDP multicast; it may need a special entitlement |
| Background | Transfers effectively need the app in the **foreground**. Keep the screen awake during a transfer and tell the user. A short background grace period may be available, but do not depend on it |
| Receiving files | Save into the app's documents (exposed through the Files app) or offer "Save to Photos" for media |
| Sending | Document picker, Photos picker, and a **Share extension** |
| Hotspot | iOS cannot create hotspots programmatically. Guide the user to Personal Hotspot in Settings |
| Wi-Fi join | `NEHotspotConfiguration` can join a known network. Scanning a Wi-Fi QR with the Camera also works |
| App Review | Explain local network use and why there is no account or server |
| Go runtime | Test `gomobile` builds on real devices and watch memory limits |

---

## 4. Web browsers (guest client)

The web client is a **guest**: it cannot discover devices or listen for connections.

| Capability | Chrome / Edge | Safari | Firefox |
|---|---|---|---|
| Streaming upload (`fetch` with stream body) | Yes | Verify | Verify |
| File System Access API (`showSaveFilePicker`) | Yes (desktop) | No | No |
| OPFS (origin private file system) | Yes | Yes | Yes |
| Downloads via normal links with streaming | Yes | Yes | Yes |
| Service worker on self-signed HTTPS | Often blocked | Verify | Verify |

*Treat the table as a starting point and check current compatibility data (for example MDN) when you implement Phase 4.*

Guidance:
- **Downloads** are easy and memory-safe: use `Content-Disposition: attachment` links that the browser streams to disk.
- **Uploads** of large files must use streams or chunked requests. Never read whole files into memory.
- **Self-signed certificate**: users must accept a warning once per browser. Provide per-browser instructions with screenshots.
- **Secure context** is required for several APIs (service workers, File System Access), which is why the guest page is served over HTTPS rather than plain HTTP.
- Keep the guest UI lightweight: it may be opened on a very old phone.

---

## 5. Network environments

| Situation | Impact | What to do |
|---|---|---|
| **AP / client isolation** (guest Wi-Fi, café, hotel, some campus networks) | Devices cannot reach each other at all | Detect (discovery finds nothing and probes time out), then explain clearly. Suggest a phone hotspot or a different network |
| **Multicast blocked or filtered** | mDNS and multicast fail | Rely on subnet scan, manual IP, or QR |
| **Multiple VLANs / subnets** | Peers on another subnet are unreachable by scan | Support manual IP; document mDNS reflectors for admins |
| **VPN active** | May hijack routing or the chosen interface | Prefer the physical Wi-Fi/Ethernet interface; allow manual selection |
| **Different Wi-Fi bands (2.4 / 5 GHz)** on the same router | Usually fine, but some routers isolate bands | Note in troubleshooting |
| **Phone hotspot** | Works well; host device is the gateway | Recommended offline fallback |
| **No router at all** | Needs a hotspot or local-only hotspot | Provide Wi-Fi QR sharing flow |
| **IPv6-only or link-local** | Not supported in v1 | Document the limitation |
| **Captive portals** | Block traffic until login | Out of scope; tell users to use a different network |

### Troubleshooting guide (for end users)

Ship a "Can't see my device?" help page covering: same Wi-Fi network, disable VPN, firewall allow, router client isolation, try manual IP, try QR, restart the app on both devices, and the Diagnostics screen.