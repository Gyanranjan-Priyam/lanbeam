"use client";

import React, { useState } from "react";
import { X, Copy, Check, QrCode, Smartphone } from "lucide-react";
import { usePeerStore } from "@/store/usePeerStore";

interface GuestQrModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export const GuestQrModal: React.FC<GuestQrModalProps> = ({ isOpen, onClose }) => {
  const [copied, setCopied] = useState(false);
  const { lanPort } = usePeerStore();

  if (!isOpen) return null;

  // LAN guest URL template
  const guestUrl = `http://192.168.1.100:${lanPort}/g/`;

  const handleCopy = () => {
    navigator.clipboard.writeText(guestUrl);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4 backdrop-blur-sm">
      <div className="relative w-full max-w-md rounded-3xl border border-white/10 bg-[#12141e] p-6 shadow-2xl">
        <button
          onClick={onClose}
          className="absolute right-4 top-4 rounded-xl p-2 text-slate-400 hover:bg-white/5 hover:text-white"
        >
          <X className="h-5 w-5" />
        </button>

        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-indigo-600/15 text-indigo-400">
            <Smartphone className="h-5 w-5" />
          </div>
          <div>
            <h2 className="text-base font-semibold text-white">Connect Phone / Web Guest</h2>
            <p className="text-xs text-slate-400">No app install required on the phone</p>
          </div>
        </div>

        {/* QR Code Placeholder Graphic */}
        <div className="my-6 flex flex-col items-center justify-center rounded-2xl border border-white/5 bg-white/[0.02] p-8">
          <div className="flex h-44 w-44 items-center justify-center rounded-xl border border-white/10 bg-white p-3 text-slate-900 shadow-lg">
            <QrCode className="h-36 w-36 text-slate-950" />
          </div>
          <p className="mt-4 text-center text-xs text-slate-400 max-w-xs">
            Scan with any mobile phone camera connected to the same Wi-Fi network.
          </p>
        </div>

        {/* URL Box */}
        <div className="flex items-center justify-between rounded-xl border border-white/10 bg-black/40 px-3 py-2 text-xs font-mono text-slate-300">
          <span className="truncate pr-2">{guestUrl}</span>
          <button
            onClick={handleCopy}
            className="flex items-center gap-1 rounded-lg bg-indigo-600/20 px-2.5 py-1 text-xs font-sans font-medium text-indigo-300 hover:bg-indigo-600 hover:text-white"
          >
            {copied ? <Check className="h-3.5 w-3.5 text-emerald-400" /> : <Copy className="h-3.5 w-3.5" />}
            <span>{copied ? "Copied" : "Copy"}</span>
          </button>
        </div>

        <div className="mt-5 flex justify-end">
          <button
            onClick={onClose}
            className="rounded-xl bg-white/5 px-4 py-2 text-xs font-medium text-slate-300 hover:bg-white/10 hover:text-white"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};
