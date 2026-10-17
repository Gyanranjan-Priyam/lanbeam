import { describe, it, expect, beforeEach } from "vitest";
import { usePeerStore, DiscoveredPeer } from "../store/usePeerStore";

describe("usePeerStore", () => {
  beforeEach(() => {
    usePeerStore.setState({
      myAlias: "Test Device",
      myFingerprint: "test-fp",
      lanPort: 42100,
      peers: {},
      transfers: {},
      pendingOffer: null,
    });
  });

  it("should upsert a discovered peer", () => {
    const peer: DiscoveredPeer = {
      fingerprint: "peer-1-fp",
      alias: "Workstation",
      deviceType: "desktop",
      addrs: ["192.168.1.10:42100"],
      online: true,
      trusted: false,
      via: "mdns",
      lastSeen: new Date().toISOString(),
    };

    usePeerStore.getState().upsertPeer(peer);
    const peers = usePeerStore.getState().peers;

    expect(peers["peer-1-fp"]).toBeDefined();
    expect(peers["peer-1-fp"].alias).toBe("Workstation");
    expect(peers["peer-1-fp"].online).toBe(true);
  });

  it("should mark a peer as offline", () => {
    const peer: DiscoveredPeer = {
      fingerprint: "peer-2-fp",
      alias: "Laptop",
      deviceType: "desktop",
      addrs: ["192.168.1.20:42100"],
      online: true,
      trusted: true,
      via: "multicast",
      lastSeen: new Date().toISOString(),
    };

    usePeerStore.getState().upsertPeer(peer);
    usePeerStore.getState().setPeerOffline("peer-2-fp");

    const peers = usePeerStore.getState().peers;
    expect(peers["peer-2-fp"].online).toBe(false);
  });

  it("should track transfer progress and status transitions", () => {
    usePeerStore.getState().addTransfer({
      sessionId: "sess-1",
      peerAlias: "Laptop",
      peerFingerprint: "peer-2-fp",
      direction: "outbound",
      fileName: "archive.zip",
      bytesDone: 0,
      totalBytes: 1000,
      speedBytesPerSec: 0,
      status: "pending",
    });

    usePeerStore.getState().updateTransferProgress("sess-1", 500, 250);
    let transfer = usePeerStore.getState().transfers["sess-1"];
    expect(transfer.bytesDone).toBe(500);
    expect(transfer.speedBytesPerSec).toBe(250);
    expect(transfer.status).toBe("transferring");

    usePeerStore.getState().updateTransferProgress("sess-1", 1000, 0);
    transfer = usePeerStore.getState().transfers["sess-1"];
    expect(transfer.status).toBe("completed");
  });
});
