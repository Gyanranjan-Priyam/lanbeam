# UX Design

Goal: **a first-time user sends a file to a nearby device in under 30 seconds, with no instructions.**

---

## 1. Design principles

1. **Zero setup.** No sign-up, no pairing ceremony for the common case.
2. **Two taps to send.** Pick files, pick a device.
3. **Always show state.** Searching, connecting, waiting for approval, transferring, done, failed. Never leave the user guessing.
4. **Explain failures in human terms**, with a next action.
5. **Same UI everywhere.** Desktop, mobile, and web guest share components, adapted by layout.
6. **Receiver stays in control.** Incoming offers are clear, safe, and easy to refuse.

---

## 2. Screen map

```
Home (Nearby)
 ├─ Send flow (files selected → choose device → progress)
 ├─ Incoming offer (modal)
 ├─ History
 ├─ Settings
 │    ├─ Device name
 │    ├─ Download folder
 │    ├─ Trusted devices
 │    ├─ Network / interface
 │    └─ Privacy and about
 ├─ Connect a browser (guest invite + QR)
 └─ Diagnostics / Help
```

Guest web app: **Pair → Waiting for approval → Send/Receive → Done**.

---

## 3. Key screens

### 3.1 Home (Nearby devices)

- Top: your device name and a status chip (*Visible to nearby devices*).
- Center: a grid or list of **device cards** (icon by type, alias, "last seen").
- Large drop zone: "Drop files here or tap to choose".
- Bottom actions: *Add by IP*, *Connect a browser (QR)*, *Help*.
- **Empty state:** friendly illustration, "Looking for nearby devices…", with tips: same Wi-Fi, firewall allowed, try Add by IP. Link to Diagnostics after about 10 seconds.

### 3.2 Send flow

1. Files chosen (drag, picker, or share sheet) → summary strip (count, total size).
2. Tap a device card to send. No extra confirm for a single selection.
3. Status steps on the card: *Waiting for [name] to accept…* → *Sending* → *Done*.
4. Cancel is always visible.

### 3.3 Incoming offer (modal)

- Header: "[Alias] wants to send you…"
- File list with names, sizes, and optional checkboxes.
- Total size, free-space check ("Enough space" or warning).
- First-contact devices: show **verification code** with a note: "Check that this code matches on the other device."
- Buttons: **Accept**, **Decline**; secondary: "Always accept from this device", "Block".
- Countdown or timeout hint.

### 3.4 Progress

- Per-file progress bars, overall bar, speed, ETA.
- States: queued, sending, verifying, done, failed (with *Retry/Resume*).
- On completion: **Open file**, **Show in folder**, **Send more**.

### 3.5 History

- Reverse-chronological list: direction icon, peer, file names, size, time, status.
- Actions: open, show in folder, resend, delete record.
- Option to clear history.

### 3.6 Connect a browser (guest invite)

- Big QR code plus the URL in text (for typing).
- Short instructions: "Scan with your phone camera. If the browser shows a security warning, tap Advanced → Continue. This is expected because there is no internet certificate authority on your local network."
- List of connected guests with **Disconnect** buttons.
- When a guest requests access: modal with device name, **confirmation code**, Allow/Deny.

### 3.7 Settings

Device name, download folder, ask-before-save toggle, auto-accept trusted, interface selector, theme, language, "Reset device identity" (with warning), version info.

### 3.8 Diagnostics

Show local IPs, chosen interface, port status, discovery methods (mDNS / multicast / scan: working or not), peers seen, recent errors, and a **Copy report** button. This screen saves you hours of support.

---

## 4. Guest web app

- Minimal, light, fast on old phones.
- Step 1: "Connecting to [host alias]…" with the **confirmation code** large on screen.
- Step 2: after approval, two big actions: **Send files** and **Receive files**.
- Show a "Keep this tab open" hint during transfers.
- If the browser lacks streaming upload for very large files, show a clear limit message rather than failing silently.

---

## 5. Microcopy

| Situation | Copy |
|---|---|
| Searching | "Looking for nearby devices…" |
| Nothing found after 10 s | "Can't see any devices? Make sure both are on the same Wi-Fi, then check Help." |
| Waiting for accept | "Waiting for Ravi's laptop to accept…" |
| Declined | "Ravi's laptop declined the transfer." |
| Busy | "That device is busy with another transfer. Try again in a moment." |
| Connection lost | "Connection lost. Your progress is saved. Tap Resume to continue." |
| Hash mismatch | "The file arrived damaged and was discarded. Please send it again." |
| Fingerprint changed | "This device's identity has changed. It may be a different device. Only continue if you trust it." |
| Not enough space | "Not enough free space for these files." |
| Isolation suspected | "This network may block devices from talking to each other. Try a phone hotspot." |

Keep copy short, calm, and free of jargon. Avoid "fingerprint" and "TLS" in primary UI. Use "verification code" and "secure connection".

---

## 6. Interaction details

- Drag-and-drop anywhere on the window (desktop).
- Paste to send: pasting an image or text into the window offers to send it.
- Keyboard: tab order, Enter to accept, Esc to cancel dialogs.
- Haptics and a subtle sound on mobile when a transfer completes (user setting).
- Notifications for incoming offers when the app is in the background (desktop and Android).
- Throttle progress rendering to avoid UI jank on large transfers.

---

## 7. Accessibility checklist

- [ ] Full keyboard navigation and visible focus
- [ ] Screen-reader labels for icons, progress bars (`role="progressbar"` with values), and dialogs
- [ ] Contrast ratio at least WCAG AA in light and dark themes
- [ ] Respect reduced-motion settings
- [ ] Touch targets at least 44×44 px
- [ ] Don't rely on color alone for status
- [ ] Text scales with system font size

---

## 8. Visual direction (suggestion)

- Clean, friendly, slightly playful, closer to AirDrop than to a sysadmin tool.
- A distinct accent color and a simple logomark (for example a beam or arc connecting two dots).
- Device icons by type: laptop, phone, tablet, browser.
- Smooth, restrained animation: cards appear with a short fade; progress bars ease.

Create low-fidelity wireframes for Home, Incoming offer, and Guest pairing before building Phase 3. A short Figma or Excalidraw session saves rework.