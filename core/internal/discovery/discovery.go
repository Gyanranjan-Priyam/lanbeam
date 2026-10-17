// Package discovery defines the Discoverer interface and shared peer types
// used by all discovery method implementations (mDNS, multicast, scan, manual).
package discovery

import (
	"context"
	"net/netip"
	"time"
)

// SelfInfo describes this device as announced to the network.
type SelfInfo struct {
	Fingerprint string
	Alias       string
	DeviceType  string
	Port        int
}

// DiscoveredPeer is a raw discovery result before registry lookup/merge.
type DiscoveredPeer struct {
	Fingerprint string
	Alias       string
	DeviceType  string
	Addr        netip.AddrPort
	Via         string
	SeenAt      time.Time
}

// Discoverer is the interface that every discovery method implements.
// Each implementation announces self and pushes DiscoveredPeer events on out.
type Discoverer interface {
	// Start begins advertising and listening.
	// Discovered peers are sent on out.
	// The method returns when ctx is cancelled.
	Start(ctx context.Context, self SelfInfo, out chan<- DiscoveredPeer) error
}
