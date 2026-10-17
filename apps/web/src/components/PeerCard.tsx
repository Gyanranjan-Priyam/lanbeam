"use client";

import React from "react";
import { Laptop, Smartphone, Globe, Send, Shield, Wifi } from "lucide-react";
import type { DiscoveredPeer } from "@/store/usePeerStore";

interface PeerCardProps {
  peer: DiscoveredPeer;
  onSelectPeer: (peer: DiscoveredPeer) => void;
}

export const PeerCard: React.FC<PeerCardProps> = ({ peer, onSelectPeer }) => {
  const getDeviceIcon = () => {
    switch (peer.deviceType) {
      case "mobile":
        return <Smartphone className="h-6 w-6 text-emerald-400" />;
      case "web":
        return <Globe className="h-6 w-6 text-sky-400" />;
      default:
        return <Laptop className="h-6 w-6 text-indigo-400" />;
    }
  };

  const shortFp = peer.fingerprint.slice(0, 8);
  const primaryAddr = peer.addrs[0] || "LAN address pending";

  return (
    <div
      data-testid={`peer-card-${peer.fingerprint}`}
      className="group relative flex flex-col justify-between rounded-2xl border border-white/5 bg-[#12141e]/70 p-5 backdrop-blur-md transition-all duration-300 hover:border-indigo-500/40 hover:bg-[#161928] hover:shadow-xl hover:shadow-indigo-500/5"
    >
      <div>
        <div className="flex items-start justify-between">
          <div className="flex items-center gap-3">
            <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-white/[0.03] border border-white/5 transition-transform group-hover:scale-105">
              {getDeviceIcon()}
            </div>
            <div>
              <div className="flex items-center gap-1.5">
                <h3 className="font-semibold text-slate-100 group-hover:text-white">{peer.alias}</h3>
                {peer.trusted && (
                  <span title="Trusted Peer">
                    <Shield className="h-3.5 w-3.5 text-emerald-400" />
                  </span>
                )}
              </div>
              <p className="text-xs text-slate-400 font-mono">{primaryAddr}</p>
            </div>
          </div>

          <div className="flex items-center gap-1.5">
            <span className="flex h-2 w-2 rounded-full bg-emerald-400 shadow-[0_0_8px_rgba(52,211,153,0.6)]" />
            <span className="text-[11px] font-medium text-slate-400 capitalize">{peer.via}</span>
          </div>
        </div>

        <div className="mt-4 flex items-center gap-2 text-[11px] text-slate-400">
          <Wifi className="h-3 w-3 text-slate-500" />
          <span>Fingerprint:</span>
          <code className="rounded bg-white/[0.05] px-1.5 py-0.5 font-mono text-[10px] text-indigo-300">
            {shortFp}...
          </code>
        </div>
      </div>

      <div className="mt-5 pt-3 border-t border-white/5 flex items-center justify-end">
        <button
          onClick={() => onSelectPeer(peer)}
          className="flex items-center gap-2 rounded-xl bg-indigo-600/20 px-3.5 py-2 text-xs font-medium text-indigo-300 transition-all hover:bg-indigo-600 hover:text-white"
        >
          <Send className="h-3.5 w-3.5" />
          <span>Send Files</span>
        </button>
      </div>
    </div>
  );
};
