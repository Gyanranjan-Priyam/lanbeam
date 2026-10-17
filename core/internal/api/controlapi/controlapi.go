// Package controlapi implements the localhost-only Control API and WebSocket event hub.
// It is served on 127.0.0.1:42101 and requires a startup bearer token.
package controlapi

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

// Service holds injected dependencies for the Control API.
type Service struct {
	Token string // startup bearer token
	// TODO Phase 1: inject transfer.Manager, peer.Registry, events.Bus, crypto.Identity
}

// NewMux builds the HTTP mux for the Control API.
func NewMux(s *Service) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /control/v1/self", s.authMiddleware(s.handleSelf))
	mux.HandleFunc("GET /control/v1/peers", s.authMiddleware(s.handlePeers))
	mux.HandleFunc("GET /control/v1/transfers", s.authMiddleware(s.handleTransfers))
	mux.HandleFunc("GET /control/v1/diagnostics", s.authMiddleware(s.handleDiagnostics))
	// TODO Phase 1: POST /peers/manual, /send, /transfers/{id}/accept, /reject, /cancel, PATCH /settings
	// TODO Phase 3: GET /control/v1/events (WebSocket)
	// TODO Phase 4: POST /guest/invite, /guest/{id}/approve
	return mux
}

// authMiddleware validates the Authorization: Bearer <token> header.
func (s *Service) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get("Authorization")
		if got != "Bearer "+s.Token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func (s *Service) handleSelf(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Service) handlePeers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (s *Service) handleTransfers(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, []any{})
}

func (s *Service) handleDiagnostics(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
