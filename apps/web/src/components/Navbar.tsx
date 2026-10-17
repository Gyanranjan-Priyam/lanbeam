"use client";

import React from "react";
import { Radio, QrCode, Settings, ShieldCheck, Laptop } from "lucide-react";
import { usePeerStore } from "@/store/usePeerStore";

interface NavbarProps {
  onOpenQr: () => void;
  onOpenSettings: () => void;
}

export const Navbar: React.FC<NavbarProps> = ({ onOpenQr, onOpenSettings }) => {
  const { myAlias, lanPort } = usePeerStore();

  return (
    <header className="sticky top-0 z-40 w-full border-b border-white/5 bg-[#090a0f]/80 backdrop-blur-xl">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-4 sm:px-6 lg:px-8">
        <div className="flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-tr from-indigo-600 to-indigo-400 text-white shadow-lg shadow-indigo-500/20">
            <Radio className="h-5 w-5 animate-pulse" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <span className="font-bold tracking-tight text-white text-lg">LanBeam</span>
              <span className="rounded-full bg-emerald-500/10 px-2 py-0.5 text-[11px] font-medium text-emerald-400 border border-emerald-500/20">
                LAN Ready
              </span>
            </div>
            <p className="text-xs text-slate-400">Offline P2P Transfer</p>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {/* Device Badge */}
          <div className="hidden sm:flex items-center gap-2 rounded-lg border border-white/5 bg-white/[0.02] px-3 py-1.5 text-xs text-slate-300">
            <Laptop className="h-4 w-4 text-indigo-400" />
            <span className="font-medium text-white">{myAlias}</span>
            <span className="text-slate-500">:{lanPort}</span>
            <ShieldCheck className="h-3.5 w-3.5 text-emerald-400 ml-1" />
          </div>

          {/* Web Guest Button */}
          <button
            onClick={onOpenQr}
            className="flex items-center gap-2 rounded-lg border border-indigo-500/30 bg-indigo-500/10 px-3 py-1.5 text-xs font-medium text-indigo-300 transition-all hover:bg-indigo-500/20 hover:text-white"
            title="Connect Phone or Guest Browser"
          >
            <QrCode className="h-4 w-4 text-indigo-400" />
            <span className="hidden md:inline">Web Guest</span>
          </button>

          {/* Settings Button */}
          <button
            onClick={onOpenSettings}
            className="rounded-lg border border-white/5 p-2 text-slate-400 transition-colors hover:bg-white/5 hover:text-white"
            title="Settings"
          >
            <Settings className="h-4 w-4" />
          </button>
        </div>
      </div>
    </header>
  );
};
