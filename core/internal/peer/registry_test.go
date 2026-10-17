package peer

import (
	"net/netip"
	"testing"
	"time"
)

func TestRegistryUpsertAndList(t *testing.T) {
	reg := NewRegistry()

	addr := netip.MustParseAddrPort("192.168.1.50:42100")
	p1 := Peer{
		Fingerprint: "aaaa1111",
		Alias:       "MacBook Pro",
		DeviceType:  DeviceDesktop,
		Addrs:       []netip.AddrPort{addr},
		LastSeen:    time.Now(),
		Via:         "mdns",
	}

	// First insert
	isNew := reg.Upsert(p1)
	if !isNew {
		t.Fatal("expected isNew=true on first insert")
	}

	retrieved, ok := reg.Get("aaaa1111")
	if !ok {
		t.Fatal("expected peer to be found")
	}
	if retrieved.Alias != "MacBook Pro" || !retrieved.Online {
		t.Errorf("unexpected peer data: %+v", retrieved)
	}

	// Update existing peer alias
	p1.Alias = "MacBook Pro M3"
	isNew = reg.Upsert(p1)
	if isNew {
		t.Fatal("expected isNew=false on update")
	}

	retrieved, _ = reg.Get("aaaa1111")
	if retrieved.Alias != "MacBook Pro M3" {
		t.Errorf("expected updated alias, got %s", retrieved.Alias)
	}

	// Mark offline
	reg.MarkOffline("aaaa1111")
	online := reg.ListOnline()
	if len(online) != 0 {
		t.Errorf("expected 0 online peers, got %d", len(online))
	}
}
