"use client";

import React, { useRef, useState } from "react";
import { UploadCloud, File, FolderUp } from "lucide-react";

interface DropzoneProps {
  onFilesSelected: (files: File[]) => void;
  selectedPeerAlias?: string;
}

export const Dropzone: React.FC<DropzoneProps> = ({ onFilesSelected, selectedPeerAlias }) => {
  const [isDragOver, setIsDragOver] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(true);
  };

  const handleDragLeave = () => {
    setIsDragOver(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragOver(false);
    if (e.dataTransfer.files && e.dataTransfer.files.length > 0) {
      onFilesSelected(Array.from(e.dataTransfer.files));
    }
  };

  const handleFileInput = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files.length > 0) {
      onFilesSelected(Array.from(e.target.files));
    }
  };

  return (
    <div
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
      onClick={() => fileInputRef.current?.click()}
      className={`group relative flex cursor-pointer flex-col items-center justify-center rounded-3xl border-2 border-dashed p-8 text-center transition-all duration-300 ${
        isDragOver
          ? "border-indigo-400 bg-indigo-500/10 shadow-[0_0_30px_rgba(99,102,241,0.2)]"
          : "border-white/10 bg-[#12141e]/40 hover:border-indigo-500/30 hover:bg-[#12141e]/80"
      }`}
    >
      <input
        ref={fileInputRef}
        type="file"
        multiple
        className="hidden"
        onChange={handleFileInput}
      />

      <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-indigo-600/10 text-indigo-400 transition-transform group-hover:scale-110 group-hover:bg-indigo-600/20">
        <UploadCloud className="h-8 w-8" />
      </div>

      <h3 className="mt-4 text-base font-semibold text-slate-100">
        {selectedPeerAlias ? (
          <>
            Send files to <span className="text-indigo-400">{selectedPeerAlias}</span>
          </>
        ) : (
          "Drop files or folders here"
        )}
      </h3>

      <p className="mt-1.5 max-w-sm text-xs text-slate-400">
        Direct streaming over high-speed local network with resumable transfers. No size limits.
      </p>

      <div className="mt-5 flex items-center gap-4 text-xs font-medium text-slate-400">
        <span className="flex items-center gap-1.5 rounded-lg border border-white/5 bg-white/[0.02] px-2.5 py-1">
          <File className="h-3.5 w-3.5 text-indigo-400" />
          Choose Files
        </span>
        <span className="flex items-center gap-1.5 rounded-lg border border-white/5 bg-white/[0.02] px-2.5 py-1">
          <FolderUp className="h-3.5 w-3.5 text-indigo-400" />
          Drag Folders
        </span>
      </div>
    </div>
  );
};
