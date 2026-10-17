import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "LanBeam — Fast Offline LAN File Transfer",
  description: "Secure, direct peer-to-peer file transfer over local Wi-Fi and Ethernet. No cloud, no internet required.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" className="dark">
      <body className="min-h-screen bg-[#090a0f] text-slate-100 antialiased selection:bg-indigo-500/30 selection:text-indigo-200">
        {children}
      </body>
    </html>
  );
}
