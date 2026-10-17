// Package peerapi implements the LAN-facing API handlers.
// These endpoints are served over TLS on the LAN port (default 42100)
// and handle peer-to-peer discovery info and file transfer.
package peerapi

import (
	"encoding/json"
	"net/http"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes the standard error envelope.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

// InfoResponse is the response body for GET /api/v1/info.
type InfoResponse struct {
	ProtocolVersion    int    `json:"protocolVersion"`
	AppVersion         string `json:"appVersion"`
	Alias              string `json:"alias"`
	DeviceType         string `json:"deviceType"`
	Fingerprint        string `json:"fingerprint"`
	AcceptingTransfers bool   `json:"acceptingTransfers"`
}

// Service holds injected dependencies for the Peer API.
type Service struct {
	Info InfoResponse
	// TODO Phase 1: inject transfer.Manager, peer.Registry, events.Bus
}

// NewMux builds the HTTP mux for the Peer API.
func NewMux(s *Service) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/info", s.handleInfo)
	// TODO Phase 1: prepare-send, send/{session}/{file}, HEAD, cancel
	return mux
}

func (s *Service) handleInfo(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.Info)
}
