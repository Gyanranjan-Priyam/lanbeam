import { create } from "zustand";

export interface DiscoveredPeer {
  fingerprint: string;
  alias: string;
  deviceType: "desktop" | "mobile" | "web" | "other";
  addrs: string[];
  online: boolean;
  trusted: boolean;
  via: string;
  lastSeen: string;
}

export interface TransferSession {
  sessionId: string;
  peerAlias: string;
  peerFingerprint: string;
  direction: "inbound" | "outbound";
  fileName: string;
  bytesDone: number;
  totalBytes: number;
  speedBytesPerSec: number;
  status: "pending" | "transferring" | "verifying" | "completed" | "failed" | "cancelled" | "rejected" | "offered";
}

interface PeerStore {
  myAlias: string;
  myFingerprint: string;
  lanPort: number;
  serverConnected: boolean;
  peers: Record<string, DiscoveredPeer>;
  transfers: Record<string, TransferSession>;
  pendingOffer: TransferSession | null;
  setSelf: (alias: string, fingerprint: string, port: number) => void;
  setServerConnected: (connected: boolean) => void;
  upsertPeer: (peer: DiscoveredPeer) => void;
  setPeerOffline: (fingerprint: string) => void;
  setPeersList: (peers: DiscoveredPeer[]) => void;
  addTransfer: (session: TransferSession) => void;
  setTransfersList: (transfers: TransferSession[]) => void;
  updateTransferProgress: (sessionId: string, bytesDone: number, speed: number) => void;
  setTransferStatus: (sessionId: string, status: TransferSession["status"]) => void;
  setPendingOffer: (offer: TransferSession | null) => void;
}

export const usePeerStore = create<PeerStore>((set) => ({
  myAlias: "",
  myFingerprint: "",
  lanPort: 42100,
  serverConnected: false,
  peers: {},
  transfers: {},
  pendingOffer: null,

  setSelf: (alias, fingerprint, port) =>
    set({ myAlias: alias, myFingerprint: fingerprint, lanPort: port }),

  setServerConnected: (connected) => set({ serverConnected: connected }),

  upsertPeer: (peer) =>
    set((state) => ({
      peers: {
        ...state.peers,
        [peer.fingerprint]: { ...peer, online: true },
      },
    })),

  setPeersList: (peerList) =>
    set(() => {
      const newPeers: Record<string, DiscoveredPeer> = {};
      for (const p of peerList) {
        newPeers[p.fingerprint] = p;
      }
      return { peers: newPeers };
    }),

  setPeerOffline: (fingerprint) =>
    set((state) => {
      const existing = state.peers[fingerprint];
      if (!existing) return state;
      return {
        peers: {
          ...state.peers,
          [fingerprint]: { ...existing, online: false },
        },
      };
    }),

  addTransfer: (session) =>
    set((state) => ({
      transfers: { ...state.transfers, [session.sessionId]: session },
    })),

  setTransfersList: (transferList) =>
    set(() => {
      const newTransfers: Record<string, TransferSession> = {};
      for (const t of transferList) {
        newTransfers[t.sessionId] = t;
      }
      return { transfers: newTransfers };
    }),

  updateTransferProgress: (sessionId, bytesDone, speed) =>
    set((state) => {
      const existing = state.transfers[sessionId];
      if (!existing) return state;
      const status = bytesDone >= existing.totalBytes ? "completed" : "transferring";
      return {
        transfers: {
          ...state.transfers,
          [sessionId]: {
            ...existing,
            bytesDone,
            speedBytesPerSec: speed,
            status: existing.status === "cancelled" ? "cancelled" : status,
          },
        },
      };
    }),

  setTransferStatus: (sessionId, status) =>
    set((state) => {
      const existing = state.transfers[sessionId];
      if (!existing) return state;
      return {
        transfers: {
          ...state.transfers,
          [sessionId]: { ...existing, status },
        },
      };
    }),

  setPendingOffer: (offer) => set({ pendingOffer: offer }),
}));
