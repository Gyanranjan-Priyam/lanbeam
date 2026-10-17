// API Client connecting Next.js frontend to LanBeam Go Control API (http://127.0.0.1:42101)

export const CONTROL_API_BASE =
  process.env.NEXT_PUBLIC_CONTROL_URL || "http://127.0.0.1:42101";

export interface SelfInfo {
  alias: string;
  deviceType: string;
  fingerprint: string;
  lanPort: number;
  controlPort: number;
  downloadDir: string;
}

export interface PeerInfo {
  fingerprint: string;
  alias: string;
  deviceType: "desktop" | "mobile" | "web" | "other";
  addrs: string[];
  online: boolean;
  trusted: boolean;
  via: string;
  lastSeen: string;
}

export interface TransferInfo {
  sessionId: string;
  direction: "inbound" | "outbound" | "send" | "receive";
  peerAlias: string;
  peerFingerprint: string;
  fileName: string;
  bytesDone: number;
  totalBytes: number;
  speedBytesPerSec: number;
  status: "pending" | "transferring" | "verifying" | "completed" | "failed" | "cancelled" | "rejected" | "offered";
  createdAt?: string;
}

/**
 * Fetch local device identity and ports
 */
export async function fetchSelf(): Promise<SelfInfo> {
  const res = await fetch(`${CONTROL_API_BASE}/control/v1/self`);
  if (!res.ok) {
    throw new Error(`Failed to fetch self: ${res.statusText}`);
  }
  return res.json();
}

/**
 * Fetch known discovered peers
 */
export async function fetchPeers(): Promise<PeerInfo[]> {
  const res = await fetch(`${CONTROL_API_BASE}/control/v1/peers`);
  if (!res.ok) {
    throw new Error(`Failed to fetch peers: ${res.statusText}`);
  }
  return res.json();
}

/**
 * Add a peer manually by IP/hostname (e.g. "192.168.1.15:42100")
 */
export async function addManualPeer(address: string): Promise<PeerInfo> {
  const res = await fetch(`${CONTROL_API_BASE}/control/v1/peers/manual`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ address }),
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.error?.message || `Failed to add peer (${res.status})`);
  }
  return res.json();
}

/**
 * Fetch active and recent transfer sessions
 */
export async function fetchTransfers(): Promise<TransferInfo[]> {
  const res = await fetch(`${CONTROL_API_BASE}/control/v1/transfers`);
  if (!res.ok) {
    throw new Error(`Failed to fetch transfers: ${res.statusText}`);
  }
  return res.json();
}

/**
 * Start a send session to a remote peer
 */
export async function startSend(
  peerFingerprint: string,
  files: { id: string; name: string; size: number; mime?: string }[]
): Promise<{ sessionId: string }> {
  const res = await fetch(`${CONTROL_API_BASE}/control/v1/send`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      peerFingerprint,
      files,
    }),
  });
  if (!res.ok) {
    const data = await res.json().catch(() => ({}));
    throw new Error(data.error?.message || `Failed to initiate send (${res.status})`);
  }
  return res.json();
}

/**
 * Accept an incoming transfer offer
 */
export async function acceptTransfer(sessionId: string): Promise<void> {
  await fetch(`${CONTROL_API_BASE}/control/v1/transfers/${encodeURIComponent(sessionId)}/accept`, {
    method: "POST",
  });
}

/**
 * Reject an incoming transfer offer
 */
export async function rejectTransfer(sessionId: string): Promise<void> {
  await fetch(`${CONTROL_API_BASE}/control/v1/transfers/${encodeURIComponent(sessionId)}/reject`, {
    method: "POST",
  });
}

/**
 * Cancel an active transfer session
 */
export async function cancelTransfer(sessionId: string): Promise<void> {
  await fetch(`${CONTROL_API_BASE}/control/v1/transfers/${encodeURIComponent(sessionId)}/cancel`, {
    method: "POST",
  });
}

/**
 * Subscribe to real-time server events via Server-Sent Events (SSE)
 */
export function subscribeToEvents(
  onEvent: (event: { type: string; data: Record<string, unknown> }) => void,
  onError?: (err: unknown) => void
): () => void {
  let eventSource: EventSource | null = null;
  try {
    eventSource = new EventSource(`${CONTROL_API_BASE}/control/v1/events`);

    eventSource.onmessage = (e) => {
      try {
        const payload = JSON.parse(e.data);
        if (payload && payload.type) {
          onEvent(payload);
        }
      } catch {
        // Heartbeat or comment
      }
    };

    eventSource.onerror = (err) => {
      if (onError) onError(err);
    };
  } catch (e) {
    if (onError) onError(e);
  }

  return () => {
    if (eventSource) {
      eventSource.close();
    }
  };
}
