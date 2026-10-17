# ADR 0003: Multi-Tier Hybrid LAN Discovery

## Status
Accepted

## Context
Devices running LanBeam must discover each other automatically on a local network without manual IP configuration. However, home and enterprise local networks behave very differently:
- Some networks support mDNS (Bonjour / Avahi) cleanly.
- Some Wi-Fi routers (guest networks, corporate isolation, or buggy consumer firmware) drop or filter multicast DNS (UDP 5353) or multicast groups (224.0.0.251).
- Some platforms restrict raw broadcast or multicast sockets.
- When multicast is filtered, devices on the same subnet cannot see each other unless an active fallback exists.

## Decision
We choose a **multi-tiered hybrid discovery architecture**:
1. **Tier 1 (Primary): mDNS**: Announce and browse `_lanbeam._tcp` with TXT records encoding device alias, fingerprint, and version. Fast, low overhead, standard across OSes.
2. **Tier 2 (Fallback): UDP Multicast & Broadcast**: Periodic announce beacons to UDP multicast (`239.255.42.100:42102`) and standard subnet broadcast (`255.255.255.255:42102`) for environments where mDNS is blocked but generic UDP is permitted.
3. **Tier 3 (Active Fallback): Subnet Sweep**: Optional/on-demand active TCP scan across the local `/24` subnet testing port 42100 with bounded worker pools.
4. **Tier 4 (Manual): Direct IP Entry & QR Pairing**: Manual IP entry and QR code direct connection for edge cases or non-routable Wi-Fi client isolation.

All discovery providers feed into a unified thread-safe `PeerRegistry` that deduplicates peer entries by their unique TLS fingerprint and tracks liveness timestamps.

## Consequences
### Positive
- High resilience: devices discover each other even under degraded network configurations or multicast-hostile routers.
- Pluggable interface: mobile platforms can swap Go discovery routines for native Android/iOS NSD (Network Service Discovery) APIs without affecting core business logic.
- Fast discovery time: typically under 1 second via mDNS, with automatic graceful fallback.

### Negative / Trade-offs
- Multiple discovery routines active simultaneously require careful deduplication and rate limiting to avoid network flooding.
