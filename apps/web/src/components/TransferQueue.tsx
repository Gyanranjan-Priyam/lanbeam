"use client";

import React from "react";
import { ArrowDownLeft, ArrowUpRight, CheckCircle2, XCircle, Loader2 } from "lucide-react";
import type { TransferSession } from "@/store/usePeerStore";

interface TransferQueueProps {
  transfers: TransferSession[];
  onCancel: (sessionId: string) => void;
}

export const TransferQueue: React.FC<TransferQueueProps> = ({ transfers, onCancel }) => {
  if (transfers.length === 0) return null;

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 B";
    const k = 1024;
    const sizes = ["B", "KB", "MB", "GB", "TB"];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return `${(bytes / Math.pow(k, i)).toFixed(1)} ${sizes[i]}`;
  };

  const formatSpeed = (bytesPerSec: number) => {
    return `${formatBytes(bytesPerSec)}/s`;
  };

  return (
    <div className="rounded-2xl border border-white/5 bg-[#12141e]/60 p-5 backdrop-blur-md">
      <h3 className="text-sm font-semibold text-slate-200">Active Transfers</h3>
      <div className="mt-3 space-y-3">
        {transfers.map((t) => {
          const percent = Math.min(100, Math.round((t.bytesDone / t.totalBytes) * 100)) || 0;
          const isInbound = t.direction === "inbound";

          return (
            <div
              key={t.sessionId}
              data-testid={`transfer-row-${t.sessionId}`}
              className="rounded-xl border border-white/5 bg-white/[0.02] p-4 transition-all"
            >
              <div className="flex items-center justify-between text-xs">
                <div className="flex items-center gap-2 font-medium text-slate-200">
                  <div
                    className={`flex h-6 w-6 items-center justify-center rounded-lg ${
                      isInbound ? "bg-emerald-500/10 text-emerald-400" : "bg-indigo-500/10 text-indigo-400"
                    }`}
                  >
                    {isInbound ? (
                      <ArrowDownLeft className="h-3.5 w-3.5" />
                    ) : (
                      <ArrowUpRight className="h-3.5 w-3.5" />
                    )}
                  </div>
                  <span className="truncate max-w-[200px] sm:max-w-xs">{t.fileName}</span>
                  <span className="text-slate-500">•</span>
                  <span className="text-slate-400 font-normal">{t.peerAlias}</span>
                </div>

                <div className="flex items-center gap-2">
                  {t.status === "completed" && (
                    <span className="flex items-center gap-1 text-emerald-400 font-medium">
                      <CheckCircle2 className="h-3.5 w-3.5" /> Done
                    </span>
                  )}
                  {t.status === "transferring" && (
                    <span className="text-slate-400 font-mono text-[11px]">
                      {formatSpeed(t.speedBytesPerSec)}
                    </span>
                  )}
                  {t.status === "verifying" && (
                    <span className="flex items-center gap-1 text-indigo-400 text-[11px]">
                      <Loader2 className="h-3 w-3 animate-spin" /> Verifying hash
                    </span>
                  )}
                  {(t.status === "transferring" || t.status === "pending") && (
                    <button
                      onClick={() => onCancel(t.sessionId)}
                      className="rounded p-1 text-slate-400 hover:text-rose-400"
                      title="Cancel Transfer"
                    >
                      <XCircle className="h-4 w-4" />
                    </button>
                  )}
                </div>
              </div>

              {/* Progress bar */}
              <div className="mt-3">
                <div className="h-1.5 w-full overflow-hidden rounded-full bg-white/5">
                  <div
                    className={`h-full transition-all duration-300 ${
                      t.status === "completed"
                        ? "bg-emerald-500"
                        : t.status === "failed"
                        ? "bg-rose-500"
                        : "bg-indigo-500 shadow-[0_0_10px_rgba(99,102,241,0.5)]"
                    }`}
                    style={{ width: `${percent}%` }}
                  />
                </div>
                <div className="mt-1.5 flex justify-between text-[11px] text-slate-400">
                  <span>{percent}%</span>
                  <span>
                    {formatBytes(t.bytesDone)} / {formatBytes(t.totalBytes)}
                  </span>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
};
