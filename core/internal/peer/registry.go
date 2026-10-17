// Package peer manages the known-peers registry.
// It is the single source of truth for peer state (online/offline/trusted).
package peer

import (
	"net/netip"
	"sync"
	"time"
)

// DeviceType describes the kind of device.
type DeviceType string

const (
	DeviceDesktop DeviceType = "desktop"
	DeviceMobile  DeviceType = "mobile"
	DeviceWeb     DeviceType = "web"
	DeviceOther   DeviceType = "other"
)

// Peer represents a remote device.
type Peer struct {
	Fingerprint string
	Alias       string
	DeviceType  DeviceType
	Addrs       []netip.AddrPort
	Trusted     bool
	FirstSeen   time.Time
	LastSeen    time.Time
	Online      bool
	Via         string // "mdns" | "multicast" | "scan" | "manual" | "native"
}

// Registry holds all known peers in memory.
// Persistence is delegated to the store package (Phase 2).
type Registry struct {
	mu    sync.RWMutex
	peers map[string]*Peer // keyed by fingerprint
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{peers: make(map[string]*Peer)}
}

// Upsert adds or updates a peer. Returns true if the peer was newly added.
func (r *Registry) Upsert(p Peer) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, exists := r.peers[p.Fingerprint]
	if exists {
		existing := r.peers[p.Fingerprint]
		existing.Alias = p.Alias
		existing.DeviceType = p.DeviceType
		existing.Addrs = p.Addrs
		existing.LastSeen = p.LastSeen
		existing.Online = true
		existing.Via = p.Via
	} else {
		p.Online = true
		if p.FirstSeen.IsZero() {
			p.FirstSeen = time.Now()
		}
		r.peers[p.Fingerprint] = &p
	}
	return !exists
}

// MarkOffline marks a peer as offline by fingerprint.
func (r *Registry) MarkOffline(fingerprint string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.peers[fingerprint]; ok {
		p.Online = false
	}
}

// Get returns the peer by fingerprint, or false if not found.
func (r *Registry) Get(fingerprint string) (Peer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.peers[fingerprint]
	if !ok {
		return Peer{}, false
	}
	return *p, true
}

// List returns all known peers.
func (r *Registry) List() []Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Peer, 0, len(r.peers))
	for _, p := range r.peers {
		out = append(out, *p)
	}
	return out
}

// ListOnline returns only peers currently marked online.
func (r *Registry) ListOnline() []Peer {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []Peer
	for _, p := range r.peers {
		if p.Online {
			out = append(out, *p)
		}
	}
	return out
}
