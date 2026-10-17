"use client";

import React, { useState, useEffect, useCallback } from "react";
import { Navbar } from "@/components/Navbar";
import { PeerCard } from "@/components/PeerCard";
import { Dropzone } from "@/components/Dropzone";
import { TransferQueue } from "@/components/TransferQueue";
import { GuestQrModal } from "@/components/GuestQrModal";
import { usePeerStore, DiscoveredPeer, TransferSession } from "@/store/usePeerStore";
import {
  Wifi,
  RefreshCw,
  Sparkles,
  Check,
  X,
  Plus,
  AlertCircle,
  Laptop,
} from "lucide-react";
import {
  fetchSelf,
  fetchPeers,
  fetchTransfers,
  startSend,
  acceptTransfer,
  rejectTransfer,
  cancelTransfer,
  addManualPeer,
  subscribeToEvents,
} from "@/lib/api";

export default function HomePage() {
  const [isQrOpen, setIsQrOpen] = useState(false);
  const [isAddPeerOpen, setIsAddPeerOpen] = useState(false);
  const [manualPeerAddr, setManualPeerAddr] = useState("");
  const [isAddingPeer, setIsAddingPeer] = useState(false);
  const [addPeerError, setAddPeerError] = useState<string | null>(null);
  const [selectedPeer, setSelectedPeer] = useState<DiscoveredPeer | null>(null);

  const {
    myAlias,
    peers,
    transfers,
    pendingOffer,
    serverConnected,
    setSelf,
    setServerConnected,
    upsertPeer,
    setPeerOffline,
    setPeersList,
    addTransfer,
    setTransfersList,
    updateTransferProgress,
    setTransferStatus,
    setPendingOffer,
  } = usePeerStore();

  // Load real server identity and state
  const refreshData = useCallback(async () => {
    try {
      const self = await fetchSelf();
      setSelf(self.alias, self.fingerprint, self.lanPort);
      setServerConnected(true);

      const peerList = await fetchPeers();
      setPeersList(peerList);

      const transferList = await fetchTransfers();
      setTransfersList(
        transferList.map((t) => ({
          sessionId: t.sessionId,
          peerAlias: t.peerAlias || "Peer",
          peerFingerprint: t.peerFingerprint || "",
          direction: t.direction === "send" ? "outbound" : "inbound",
          fileName: t.fileName || "File",
          bytesDone: t.bytesDone || 0,
          totalBytes: t.totalBytes || 1,
          speedBytesPerSec: t.speedBytesPerSec || 0,
          status: t.status as TransferSession["status"],
        }))
      );
    } catch {
      setServerConnected(false);
    }
  }, [setSelf, setServerConnected, setPeersList, setTransfersList]);

  // Initial fetch and real-time SSE event subscription
  useEffect(() => {
    refreshData();

    // EventSource subscription for real-time push events from Go server
    const unsubscribe = subscribeToEvents(
      (ev) => {
        setServerConnected(true);
        if (ev.type === "peer.up" && ev.data) {
          upsertPeer({
            fingerprint: String(ev.data.Fingerprint || ev.data.fingerprint || ""),
            alias: String(ev.data.Alias || ev.data.alias || "Peer"),
            deviceType: (ev.data.DeviceType || ev.data.deviceType || "desktop") as DiscoveredPeer["deviceType"],
            addrs: Array.isArray(ev.data.addrs) ? (ev.data.addrs as string[]) : [],
            online: true,
            trusted: Boolean(ev.data.trusted),
            via: String(ev.data.via || "manual"),
            lastSeen: new Date().toISOString(),
          });
        } else if (ev.type === "peer.down" && ev.data) {
          setPeerOffline(String(ev.data.fingerprint || ev.data.Fingerprint || ""));
        } else if (ev.type === "offer.incoming" && ev.data) {
          setPendingOffer({
            sessionId: String(ev.data.sessionId || ""),
            peerAlias: String(ev.data.peerAlias || "Remote Peer"),
            peerFingerprint: String(ev.data.peerFp || ""),
            direction: "inbound",
            fileName: String(ev.data.fileName || "Incoming file"),
            bytesDone: 0,
            totalBytes: Number(ev.data.totalBytes) || 0,
            speedBytesPerSec: 0,
            status: "offered",
          });
        } else if (ev.type === "progress" && ev.data) {
          updateTransferProgress(
            String(ev.data.sessionId || ""),
            Number(ev.data.bytesDone) || 0,
            Number(ev.data.bytesPerSec) || 0
          );
        } else if (ev.type === "transfer.done" && ev.data) {
          setTransferStatus(
            String(ev.data.sessionId || ""),
            (ev.data.status || "completed") as TransferSession["status"]
          );
        }
      },
      () => {
        // SSE disconnected, fallback to periodic polling
      }
    );

    // Periodic sync interval (every 4 seconds)
    const interval = setInterval(refreshData, 4000);

    return () => {
      unsubscribe();
      clearInterval(interval);
    };
  }, [refreshData, setServerConnected, upsertPeer, setPeerOffline, setPendingOffer, updateTransferProgress, setTransferStatus]);

  // Start real file send to selected peer
  const handleFilesSelected = async (files: File[]) => {
    const targetPeer = selectedPeer || Object.values(peers)[0];
    if (!targetPeer) return;

    for (const file of files) {
      const fileId = `f-${Date.now()}-${Math.random().toString(36).substring(2, 6)}`;
      try {
        const res = await startSend(targetPeer.fingerprint, [
          {
            id: fileId,
            name: file.name,
            size: file.size,
            mime: file.type || "application/octet-stream",
          },
        ]);

        addTransfer({
          sessionId: res.sessionId,
          peerAlias: targetPeer.alias,
          peerFingerprint: targetPeer.fingerprint,
          direction: "outbound",
          fileName: file.name,
          bytesDone: 0,
          totalBytes: file.size,
          speedBytesPerSec: 0,
          status: "transferring",
        });
      } catch (err) {
        console.error("Failed to start send:", err);
      }
    }
  };

  const handleCancelTransfer = async (sessionId: string) => {
    try {
      await cancelTransfer(sessionId);
    } catch {
      // Ignored
    }
    setTransferStatus(sessionId, "cancelled");
  };

  const handleAcceptOffer = async () => {
    if (!pendingOffer) return;
    try {
      await acceptTransfer(pendingOffer.sessionId);
      addTransfer({
        ...pendingOffer,
        status: "transferring",
      });
    } catch {
      // Ignored
    }
    setPendingOffer(null);
  };

  const handleRejectOffer = async () => {
    if (!pendingOffer) return;
    try {
      await rejectTransfer(pendingOffer.sessionId);
    } catch {
      // Ignored
    }
    setPendingOffer(null);
  };

  const handleAddManualPeer = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!manualPeerAddr.trim()) return;
    setIsAddingPeer(true);
    setAddPeerError(null);
    try {
      const p = await addManualPeer(manualPeerAddr.trim());
      upsertPeer({
        fingerprint: p.fingerprint,
        alias: p.alias,
        deviceType: p.deviceType,
        addrs: p.addrs,
        online: p.online,
        trusted: p.trusted,
        via: p.via,
        lastSeen: p.lastSeen,
      });
      setManualPeerAddr("");
      setIsAddPeerOpen(false);
    } catch (err: unknown) {
      setAddPeerError(err instanceof Error ? err.message : "Failed to connect to peer");
    } finally {
      setIsAddingPeer(false);
    }
  };

  const peerList = Object.values(peers);
  const transferList = Object.values(transfers);

  return (
    <div className="min-h-screen bg-[#090a0f] text-slate-100 flex flex-col">
      <Navbar onOpenQr={() => setIsQrOpen(true)} onOpenSettings={() => {}} />

      {/* Backend connection warning when Go server is not running */}
      {!serverConnected && (
        <div className="bg-amber-500/10 border-b border-amber-500/20 px-4 py-2.5 text-xs text-amber-200">
          <div className="mx-auto max-w-7xl flex items-center justify-between gap-4">
            <div className="flex items-center gap-2">
              <AlertCircle className="h-4 w-4 text-amber-400 shrink-0" />
              <span>
                <strong>Go Core Server Offline:</strong> Listening on localhost:42101 failed. Start the server with{" "}
                <code className="bg-black/40 px-1.5 py-0.5 rounded font-mono text-amber-300">
                  bun run dev:server
                </code>{" "}
                or <code className="bg-black/40 px-1.5 py-0.5 rounded font-mono text-amber-300">make dev-core</code>.
              </span>
            </div>
            <button
              onClick={() => refreshData()}
              className="flex items-center gap-1 font-medium text-amber-300 hover:text-white underline shrink-0"
            >
              <RefreshCw className="h-3 w-3" /> Retry
            </button>
          </div>
        </div>
      )}

      {/* Pending incoming offer banner */}
      {pendingOffer && (
        <div className="bg-indigo-600/20 border-b border-indigo-500/30 px-4 py-3">
          <div className="mx-auto max-w-7xl flex items-center justify-between">
            <div className="flex items-center gap-2 text-xs text-indigo-200">
              <Sparkles className="h-4 w-4 text-indigo-400" />
              <span>
                <strong>{pendingOffer.peerAlias}</strong> wants to send you{" "}
                <strong>{pendingOffer.fileName}</strong>
              </span>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={handleAcceptOffer}
                className="flex items-center gap-1 rounded-lg bg-emerald-600 px-3 py-1 text-xs font-medium text-white hover:bg-emerald-500 transition-colors"
              >
                <Check className="h-3.5 w-3.5" /> Accept
              </button>
              <button
                onClick={handleRejectOffer}
                className="flex items-center gap-1 rounded-lg bg-white/10 px-3 py-1 text-xs font-medium text-slate-300 hover:bg-white/15 transition-colors"
              >
                <X className="h-3.5 w-3.5" /> Reject
              </button>
            </div>
          </div>
        </div>
      )}

      <main className="mx-auto w-full max-w-7xl flex-1 px-4 py-8 sm:px-6 lg:px-8 space-y-8">
        {/* Top Header */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold tracking-tight text-white flex items-center gap-2.5">
              <span>Nearby Devices</span>
              {serverConnected && myAlias && (
                <span className="text-xs font-normal text-slate-400 border border-white/10 rounded-full px-2.5 py-0.5 bg-white/[0.02]">
                  Self: {myAlias}
                </span>
              )}
            </h1>
            <p className="text-xs text-slate-400 mt-1">
              Devices on your Wi-Fi running LanBeam appear automatically below.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setIsAddPeerOpen(true)}
              className="flex items-center gap-1.5 rounded-xl border border-indigo-500/30 bg-indigo-500/10 px-3.5 py-2 text-xs font-medium text-indigo-300 hover:bg-indigo-500/20 hover:text-white transition-colors"
            >
              <Plus className="h-3.5 w-3.5" />
              <span>Add Peer by IP</span>
            </button>
            <button
              onClick={() => refreshData()}
              className="flex items-center gap-2 rounded-xl border border-white/10 bg-white/[0.03] px-3.5 py-2 text-xs font-medium text-slate-300 hover:bg-white/5 hover:text-white transition-colors"
              title="Refresh device status"
            >
              <RefreshCw className="h-3.5 w-3.5" />
              <span>Scan Subnet</span>
            </button>
          </div>
        </div>

        {/* Peer Grid */}
        <section>
          {peerList.length === 0 ? (
            <div className="flex flex-col items-center justify-center rounded-3xl border border-white/5 bg-[#12141e]/40 p-12 text-center">
              <Wifi className="h-10 w-10 text-slate-600 animate-pulse" />
              <h3 className="mt-4 text-sm font-semibold text-slate-300">Searching for peers...</h3>
              <p className="mt-1 text-xs text-slate-500 max-w-sm">
                No peers discovered yet. Ensure both devices are on the same Wi-Fi, or click &quot;Add Peer by IP&quot; to connect directly.
              </p>
              <button
                onClick={() => setIsAddPeerOpen(true)}
                className="mt-4 flex items-center gap-1.5 rounded-xl bg-white/5 border border-white/10 px-3 py-1.5 text-xs text-indigo-300 hover:bg-white/10 hover:text-white transition-colors"
              >
                <Plus className="h-3.5 w-3.5" /> Add Peer by IP
              </button>
            </div>
          ) : (
            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {peerList.map((peer) => (
                <PeerCard
                  key={peer.fingerprint}
                  peer={peer}
                  onSelectPeer={(p) => setSelectedPeer(p)}
                />
              ))}
            </div>
          )}
        </section>

        {/* Dropzone & Active Transfers Split */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-12">
          <div className="lg:col-span-7">
            <Dropzone
              onFilesSelected={handleFilesSelected}
              selectedPeerAlias={selectedPeer?.alias}
            />
          </div>
          <div className="lg:col-span-5">
            <TransferQueue transfers={transferList} onCancel={handleCancelTransfer} />
          </div>
        </div>
      </main>

      {/* Add Peer Modal */}
      {isAddPeerOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm">
          <div className="relative w-full max-w-md rounded-3xl border border-white/10 bg-[#12141e] p-6 shadow-2xl">
            <button
              onClick={() => setIsAddPeerOpen(false)}
              className="absolute right-4 top-4 rounded-xl p-2 text-slate-400 hover:bg-white/5 hover:text-white"
            >
              <X className="h-5 w-5" />
            </button>

            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-600/15 text-indigo-400">
                <Laptop className="h-5 w-5" />
              </div>
              <div>
                <h2 className="text-base font-semibold text-white">Add Peer Manually</h2>
                <p className="text-xs text-slate-400">Directly connect to another LanBeam device by IP</p>
              </div>
            </div>

            <form onSubmit={handleAddManualPeer} className="mt-5 space-y-4">
              <div>
                <label className="block text-xs font-medium text-slate-300 mb-1.5">
                  Peer Address (IP:Port)
                </label>
                <input
                  type="text"
                  placeholder="192.168.1.15:42100"
                  value={manualPeerAddr}
                  onChange={(e) => setManualPeerAddr(e.target.value)}
                  className="w-full rounded-xl border border-white/10 bg-black/40 px-3.5 py-2.5 text-xs text-white placeholder-slate-500 focus:border-indigo-500 focus:outline-none focus:ring-1 focus:ring-indigo-500 font-mono"
                  autoFocus
                />
                <p className="text-[11px] text-slate-500 mt-1">Default LAN port is 42100 if omitted.</p>
              </div>

              {addPeerError && (
                <div className="rounded-xl border border-rose-500/20 bg-rose-500/10 p-2.5 text-xs text-rose-300">
                  {addPeerError}
                </div>
              )}

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setIsAddPeerOpen(false)}
                  className="rounded-xl bg-white/5 px-4 py-2 text-xs font-medium text-slate-300 hover:bg-white/10 hover:text-white"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isAddingPeer || !manualPeerAddr.trim()}
                  className="rounded-xl bg-indigo-600 px-4 py-2 text-xs font-medium text-white hover:bg-indigo-500 disabled:opacity-50"
                >
                  {isAddingPeer ? "Probing..." : "Connect Peer"}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <GuestQrModal isOpen={isQrOpen} onClose={() => setIsQrOpen(false)} />
    </div>
  );
}
