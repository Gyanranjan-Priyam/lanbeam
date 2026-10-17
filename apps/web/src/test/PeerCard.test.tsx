import React from "react";
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PeerCard } from "../components/PeerCard";
import type { DiscoveredPeer } from "../store/usePeerStore";

describe("PeerCard", () => {
  const mockPeer: DiscoveredPeer = {
    fingerprint: "abcdef1234567890abcdef1234567890",
    alias: "Office Laptop",
    deviceType: "desktop",
    addrs: ["192.168.1.99:42100"],
    online: true,
    trusted: true,
    via: "mdns",
    lastSeen: new Date().toISOString(),
  };

  it("renders peer alias, address, and fingerprint correctly", () => {
    const onSelect = vi.fn();
    render(<PeerCard peer={mockPeer} onSelectPeer={onSelect} />);

    expect(screen.getByText("Office Laptop")).toBeInTheDocument();
    expect(screen.getByText("192.168.1.99:42100")).toBeInTheDocument();
    expect(screen.getByText("abcdef12...")).toBeInTheDocument();
    expect(screen.getByText("mdns")).toBeInTheDocument();
  });

  it("calls onSelectPeer callback when Send Files button is clicked", () => {
    const onSelect = vi.fn();
    render(<PeerCard peer={mockPeer} onSelectPeer={onSelect} />);

    const sendButton = screen.getByRole("button", { name: /send files/i });
    fireEvent.click(sendButton);

    expect(onSelect).toHaveBeenCalledTimes(1);
    expect(onSelect).toHaveBeenCalledWith(mockPeer);
  });
});
