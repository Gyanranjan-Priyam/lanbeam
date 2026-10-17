// Package controlapi implements the localhost-only Control API and event hub.
// It is served on 127.0.0.1:42101 and optionally requires a startup bearer token.
package controlapi

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"strings"
	"time"

	"github.com/lanbeam/lanbeam/core/internal/config"
	"github.com/lanbeam/lanbeam/core/internal/crypto"
	"github.com/lanbeam/lanbeam/core/internal/events"
	"github.com/lanbeam/lanbeam/core/internal/peer"
	"github.com/lanbeam/lanbeam/core/internal/transfer"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a standardized error envelope.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// Service holds injected dependencies for the Control API.
type Service struct {
	Token    string
	Config   *config.Config
	Identity *crypto.Identity
	Registry *peer.Registry
	Bus      *events.Bus
	Manager  *transfer.Manager
}

// NewMux builds the HTTP mux for the Control API with CORS support.
func NewMux(s *Service) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /control/v1/self", s.authMiddleware(s.handleSelf))
	mux.HandleFunc("GET /control/v1/peers", s.authMiddleware(s.handlePeers))
	mux.HandleFunc("POST /control/v1/peers/manual", s.authMiddleware(s.handleManualPeer))
	mux.HandleFunc("POST /control/v1/send", s.authMiddleware(s.handleSend))
	mux.HandleFunc("GET /control/v1/transfers", s.authMiddleware(s.handleTransfers))
	mux.HandleFunc("POST /control/v1/transfers/{id}/accept", s.authMiddleware(s.handleTransferAccept))
	mux.HandleFunc("POST /control/v1/transfers/{id}/reject", s.authMiddleware(s.handleTransferReject))
	mux.HandleFunc("POST /control/v1/transfers/{id}/cancel", s.authMiddleware(s.handleTransferCancel))
	mux.HandleFunc("GET /control/v1/events", s.authMiddleware(s.handleEvents))
	mux.HandleFunc("GET /control/v1/diagnostics", s.authMiddleware(s.handleDiagnostics))

	return s.corsMiddleware(mux)
}

// corsMiddleware provides CORS support for local web apps (e.g. Next.js on localhost:3000).
func (s *Service) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authMiddleware validates the Authorization: Bearer <token> header when a token is configured.
func (s *Service) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Token != "" {
			got := r.Header.Get("Authorization")
			if got != "Bearer "+s.Token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func (s *Service) handleSelf(w http.ResponseWriter, _ *http.Request) {
	alias := "lanbeam-device"
	lanPort := 42100
	controlPort := 42101
	downloadDir := ""
	fingerprint := ""

	if s.Config != nil {
		alias = s.Config.Alias
		lanPort = s.Config.LANPort
		controlPort = s.Config.ControlPort
		downloadDir = s.Config.DownloadDir
	}
	if s.Identity != nil {
		fingerprint = s.Identity.Fingerprint
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"alias":       alias,
		"deviceType":  "desktop",
		"fingerprint": fingerprint,
		"lanPort":     lanPort,
		"controlPort": controlPort,
		"downloadDir": downloadDir,
	})
}

func (s *Service) handlePeers(w http.ResponseWriter, _ *http.Request) {
	if s.Registry == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	peers := s.Registry.List()
	out := make([]map[string]any, 0, len(peers))
	for _, p := range peers {
		addrs := make([]string, 0, len(p.Addrs))
		for _, a := range p.Addrs {
			addrs = append(addrs, a.String())
		}
		lastSeen := p.LastSeen
		if lastSeen.IsZero() {
			lastSeen = time.Now()
		}
		out = append(out, map[string]any{
			"fingerprint": p.Fingerprint,
			"alias":       p.Alias,
			"deviceType":  string(p.DeviceType),
			"addrs":       addrs,
			"online":      p.Online,
			"trusted":     p.Trusted,
			"via":         p.Via,
			"lastSeen":    lastSeen.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

type manualPeerRequest struct {
	Address string `json:"address"`
}

func (s *Service) handleManualPeer(w http.ResponseWriter, r *http.Request) {
	var req manualPeerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Address == "" {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "address is required")
		return
	}

	addrStr := strings.TrimSpace(req.Address)
	if !strings.Contains(addrStr, ":") {
		addrStr = fmt.Sprintf("%s:42100", addrStr)
	}

	parsedAddr, err := netip.ParseAddrPort(addrStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_ADDRESS", "could not parse host:port")
		return
	}

	// Probe peer over TLS
	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec // Self-signed certs in local network
			},
		},
	}

	url := fmt.Sprintf("https://%s/api/v1/info", addrStr)
	resp, err := client.Get(url)
	if err != nil {
		writeError(w, http.StatusNotFound, "PEER_UNREACHABLE", fmt.Sprintf("cannot connect to peer at %s: %v", addrStr, err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadGateway, "PEER_ERROR", fmt.Sprintf("peer returned status %d", resp.StatusCode))
		return
	}

	var info struct {
		Alias       string `json:"alias"`
		DeviceType  string `json:"deviceType"`
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		writeError(w, http.StatusBadGateway, "INVALID_PEER_RESPONSE", "failed to decode peer info")
		return
	}

	p := peer.Peer{
		Fingerprint: info.Fingerprint,
		Alias:       info.Alias,
		DeviceType:  peer.DeviceType(info.DeviceType),
		Addrs:       []netip.AddrPort{parsedAddr},
		Online:      true,
		Trusted:     false,
		Via:         "manual",
		LastSeen:    time.Now(),
	}

	if s.Registry != nil {
		s.Registry.Upsert(p)
	}

	if s.Bus != nil {
		s.Bus.Publish(events.Event{
			Type: events.EventPeerUp,
			Data: p,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"fingerprint": p.Fingerprint,
		"alias":       p.Alias,
		"deviceType":  string(p.DeviceType),
		"addrs":       []string{parsedAddr.String()},
		"online":      p.Online,
		"trusted":     p.Trusted,
		"via":         p.Via,
		"lastSeen":    p.LastSeen.Format(time.RFC3339),
	})
}

type sendFileItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Size int64  `json:"size"`
	Mime string `json:"mime,omitempty"`
}

type sendRequestBody struct {
	PeerFingerprint string         `json:"peerFingerprint"`
	Files           []sendFileItem `json:"files"`
}

func (s *Service) handleSend(w http.ResponseWriter, r *http.Request) {
	var req sendRequestBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.PeerFingerprint == "" || len(req.Files) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "peerFingerprint and files are required")
		return
	}

	sessionId := fmt.Sprintf("tx-%d-%s", time.Now().UnixNano()/1e6, req.Files[0].ID)

	var peerAlias string
	if s.Registry != nil {
		if p, ok := s.Registry.Get(req.PeerFingerprint); ok {
			peerAlias = p.Alias
		}
	}
	if peerAlias == "" {
		peerAlias = "Remote Peer"
	}

	sess := &transfer.Session{
		ID:        sessionId,
		Direction: "send",
		Sender: transfer.SenderInfo{
			Alias:       s.Config.Alias,
			DeviceType:  "desktop",
			Fingerprint: s.Identity.Fingerprint,
		},
		Status:    transfer.StatusTransferring,
		CreatedAt: time.Now(),
	}

	for _, f := range req.Files {
		sess.Files = append(sess.Files, &transfer.FileTransfer{
			Info: transfer.FileInfo{
				ID:   f.ID,
				Name: f.Name,
				Size: f.Size,
				MIME: f.Mime,
			},
			Status: transfer.StatusTransferring,
		})
	}

	if s.Manager != nil {
		s.Manager.Register(sess)
	}

	if s.Bus != nil {
		s.Bus.Publish(events.Event{
			Type: events.EventProgress,
			Data: map[string]any{
				"sessionId":   sessionId,
				"peerAlias":   peerAlias,
				"fileName":    req.Files[0].Name,
				"bytesDone":   0,
				"totalBytes":  req.Files[0].Size,
				"bytesPerSec": 0,
				"status":      "transferring",
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessionId": sessionId,
	})
}

func (s *Service) handleTransfers(w http.ResponseWriter, _ *http.Request) {
	if s.Manager == nil {
		writeJSON(w, http.StatusOK, []any{})
		return
	}
	sessions := s.Manager.List()
	out := make([]map[string]any, 0, len(sessions))
	for _, sess := range sessions {
		var fileName string
		var totalBytes int64
		var bytesDone int64
		if len(sess.Files) > 0 {
			fileName = sess.Files[0].Info.Name
			totalBytes = sess.Files[0].Info.Size
			bytesDone = sess.Files[0].BytesDone
		}
		out = append(out, map[string]any{
			"sessionId":        sess.ID,
			"direction":        sess.Direction,
			"peerAlias":        sess.Sender.Alias,
			"peerFingerprint":  sess.Sender.Fingerprint,
			"fileName":         fileName,
			"bytesDone":        bytesDone,
			"totalBytes":       totalBytes,
			"speedBytesPerSec": 0,
			"status":           string(sess.Status),
			"createdAt":        sess.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Service) handleTransferAccept(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.Manager != nil {
		if sess, ok := s.Manager.Get(id); ok {
			sess.Status = transfer.StatusTransferring
		}
	}
	if s.Bus != nil {
		s.Bus.Publish(events.Event{
			Type: events.EventProgress,
			Data: map[string]any{
				"sessionId": id,
				"status":    "transferring",
			},
		})
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "accepted"})
}

func (s *Service) handleTransferReject(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.Manager != nil {
		if sess, ok := s.Manager.Get(id); ok {
			sess.Status = transfer.StatusRejected
		}
	}
	if s.Bus != nil {
		s.Bus.Publish(events.Event{
			Type: events.EventTransferDone,
			Data: map[string]any{
				"sessionId": id,
				"status":    "rejected",
			},
		})
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "rejected"})
}

func (s *Service) handleTransferCancel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if s.Manager != nil {
		s.Manager.Cancel(id)
	}
	if s.Bus != nil {
		s.Bus.Publish(events.Event{
			Type: events.EventTransferDone,
			Data: map[string]any{
				"sessionId": id,
				"status":    "cancelled",
			},
		})
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Service) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	if s.Bus == nil {
		fmt.Fprintf(w, ": no event bus\n\n")
		flusher.Flush()
		return
	}

	ch, cancel := s.Bus.Subscribe(32)
	defer cancel()

	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, open := <-ch:
			if !open {
				return
			}
			data, err := json.Marshal(ev)
			if err != nil {
				continue
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (s *Service) handleDiagnostics(w http.ResponseWriter, _ *http.Request) {
	addrs, _ := net.InterfaceAddrs()
	ipList := make([]string, 0)
	for _, a := range addrs {
		if ipNet, ok := a.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			ipList = append(ipList, ipNet.IP.String())
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":     "ok",
		"interfaces": ipList,
		"goroutines": runtime.NumGoroutine(),
		"version":    "0.1.0",
		"alias":      s.Config.Alias,
		"lanPort":    s.Config.LANPort,
	})
}
