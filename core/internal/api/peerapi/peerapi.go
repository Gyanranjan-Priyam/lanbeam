// Package peerapi implements the LAN-facing API handlers.
// These endpoints are served over TLS on the LAN port (default 42100)
// and handle peer-to-peer discovery info and file transfer.
package peerapi

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	Info     InfoResponse
	Config   *config.Config
	Identity *crypto.Identity
	Registry *peer.Registry
	Bus      *events.Bus
	Manager  *transfer.Manager
}

// NewMux builds the HTTP mux for the Peer API.
func NewMux(s *Service) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/info", s.handleInfo)
	mux.HandleFunc("POST /api/v1/prepare-send", s.handlePrepareSend)
	mux.HandleFunc("POST /api/v1/cancel/{sessionId}", s.handleCancel)
	return mux
}

func (s *Service) handleInfo(w http.ResponseWriter, _ *http.Request) {
	info := s.Info
	if s.Config != nil {
		info.Alias = s.Config.Alias
	}
	if s.Identity != nil {
		info.Fingerprint = s.Identity.Fingerprint
	}
	info.ProtocolVersion = 1
	info.AppVersion = "0.1.0"
	info.DeviceType = "desktop"
	info.AcceptingTransfers = true

	writeJSON(w, http.StatusOK, info)
}

type prepareSendBody struct {
	Sender transfer.SenderInfo `json:"sender"`
	Files  []transfer.FileInfo `json:"files"`
	Text   *string             `json:"text,omitempty"`
}

func (s *Service) handlePrepareSend(w http.ResponseWriter, r *http.Request) {
	var req prepareSendBody
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Files) == 0 {
		writeError(w, http.StatusBadRequest, "INVALID_REQUEST", "files and sender are required")
		return
	}

	sessionId := fmt.Sprintf("in-%d", time.Now().UnixNano()/1e6)
	sess := &transfer.Session{
		ID:        sessionId,
		Direction: "receive",
		Sender:    req.Sender,
		Status:    transfer.StatusOffered,
		CreatedAt: time.Now(),
	}

	filesMap := make(map[string]map[string]any)
	for _, f := range req.Files {
		sess.Files = append(sess.Files, &transfer.FileTransfer{
			Info:   f,
			Status: transfer.StatusOffered,
		})
		filesMap[f.ID] = map[string]any{
			"token":  fmt.Sprintf("tok-%s-%s", sessionId, f.ID),
			"offset": int64(0),
		}
	}

	if s.Manager != nil {
		s.Manager.Register(sess)
	}

	if s.Bus != nil {
		s.Bus.Publish(events.Event{
			Type: events.EventOfferIncoming,
			Data: map[string]any{
				"sessionId":   sessionId,
				"peerAlias":   req.Sender.Alias,
				"peerFp":      req.Sender.Fingerprint,
				"fileName":    req.Files[0].Name,
				"totalBytes":  req.Files[0].Size,
				"fileCount":   len(req.Files),
			},
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"sessionId": sessionId,
		"files":     filesMap,
	})
}

func (s *Service) handleCancel(w http.ResponseWriter, r *http.Request) {
	sessionId := r.PathValue("sessionId")
	if s.Manager != nil {
		s.Manager.Cancel(sessionId)
	}
	if s.Bus != nil {
		s.Bus.Publish(events.Event{
			Type: events.EventTransferDone,
			Data: map[string]any{
				"sessionId": sessionId,
				"status":    "cancelled",
			},
		})
	}
	w.WriteHeader(http.StatusNoContent)
}
