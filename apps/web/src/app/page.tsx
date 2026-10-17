"use client";

import React, { useState, useEffect } from "react";
import { Navbar } from "@/components/Navbar";
import { PeerCard } from "@/components/PeerCard";
import { Dropzone } from "@/components/Dropzone";
import { TransferQueue } from "@/components/TransferQueue";
import { GuestQrModal } from "@/components/GuestQrModal";
import { usePeerStore, DiscoveredPeer } from "@/store/usePeerStore";
import { Wifi, RefreshCw, Sparkles, Check, X } from "lucide-react";

export default function HomePage() {
  const [isQrOpen, setIsQrOpen] = useState(false);
  const [selectedPeer, setSelectedPeer] = useState<DiscoveredPeer | null>(null);

  const {
    peers,
    transfers,
    pendingOffer,
    upsertPeer,
    addTransfer,
    updateTransferProgress,
    setTransferStatus,
    setPendingOffer,
  } = usePeerStore();

  // Populate initial demo peers on mount for immediate interactive preview
  useEffect(() => {
    if (Object.keys(peers).length === 0) {
      upsertPeer({
        fingerprint: "7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b",
        alias: "Alex's MacBook Pro",
        deviceType: "desktop",
        addrs: ["192.168.1.145:42100"],
        online: true,
        trusted: true,
        via: "mdns",
        lastSeen: new Date().toISOString(),
      });
      upsertPeer({
        fingerprint: "11223344556677889900aabbccddeeff11223344556677889900aabbccddeeff",
        alias: "Pixel 9 Pro",
        deviceType: "mobile",
        addrs: ["192.168.1.189:42100"],
        online: true,
        trusted: false,
        via: "multicast",
        lastSeen: new Date().toISOString(),
      });
    }
  }, [peers, upsertPeer]);

  const handleFilesSelected = (files: File[]) => {
    const targetPeer = selectedPeer || Object.values(peers)[0];
    if (!targetPeer) return;

    files.forEach((file) => {
      const sessionId = `tx-${Date.now()}-${Math.random().toString(36).substring(2, 7)}`;
      addTransfer({
        sessionId,
        peerAlias: targetPeer.alias,
        peerFingerprint: targetPeer.fingerprint,
        direction: "outbound",
        fileName: file.name,
        bytesDone: 0,
        totalBytes: file.size || 1024 * 1024 * 50,
        speedBytesPerSec: 0,
        status: "transferring",
      });

      // Simulate streaming transfer progress
      let currentBytes = 0;
      const total = file.size || 1024 * 1024 * 50;
      const chunkSize = Math.max(1024 * 256, Math.floor(total / 20));

      const interval = setInterval(() => {
        currentBytes += chunkSize;
        const speed = chunkSize * 4; // ~1-4 MB/s simulation
        if (currentBytes >= total) {
          clearInterval(interval);
          updateTransferProgress(sessionId, total, 0);
          setTransferStatus(sessionId, "verifying");
          setTimeout(() => {
            setTransferStatus(sessionId, "completed");
          }, 600);
        } else {
          updateTransferProgress(sessionId, currentBytes, speed);
        }
      }, 250);
    });
  };

  const handleCancelTransfer = (sessionId: string) => {
    setTransferStatus(sessionId, "cancelled");
  };

  const peerList = Object.values(peers);
  const transferList = Object.values(transfers);

  return (
    <div className="min-h-screen bg-[#090a0f] text-slate-100 flex flex-col">
      <Navbar onOpenQr={() => setIsQrOpen(true)} onOpenSettings={() => {}} />

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
                onClick={() => setPendingOffer(null)}
                className="flex items-center gap-1 rounded-lg bg-emerald-600 px-3 py-1 text-xs font-medium text-white hover:bg-emerald-500"
              >
                <Check className="h-3.5 w-3.5" /> Accept
              </button>
              <button
                onClick={() => setPendingOffer(null)}
                className="flex items-center gap-1 rounded-lg bg-white/10 px-3 py-1 text-xs font-medium text-slate-300 hover:bg-white/15"
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
            <h1 className="text-2xl font-bold tracking-tight text-white">Nearby Devices</h1>
            <p className="text-xs text-slate-400 mt-1">
              Devices on your Wi-Fi running LanBeam appear automatically below.
            </p>
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => {}}
              className="flex items-center gap-2 rounded-xl border border-white/10 bg-white/[0.03] px-3.5 py-2 text-xs font-medium text-slate-300 hover:bg-white/5 hover:text-white"
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
                Make sure both devices are on the same Wi-Fi or hotspot.
              </p>
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

      <GuestQrModal isOpen={isQrOpen} onClose={() => setIsQrOpen(false)} />
    </div>
  );
}
