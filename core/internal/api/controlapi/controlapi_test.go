package controlapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lanbeam/lanbeam/core/internal/config"
	"github.com/lanbeam/lanbeam/core/internal/crypto"
	"github.com/lanbeam/lanbeam/core/internal/events"
	"github.com/lanbeam/lanbeam/core/internal/peer"
	"github.com/lanbeam/lanbeam/core/internal/transfer"
)

func setupTestService(t *testing.T) *Service {
	t.Helper()
	id, err := crypto.New()
	if err != nil {
		t.Fatalf("crypto.New failed: %v", err)
	}

	cfg := &config.Config{
		Alias:       "Test-Device",
		LANPort:     42100,
		ControlPort: 42101,
		DownloadDir: "/tmp/downloads",
	}

	return &Service{
		Config:   cfg,
		Identity: id,
		Registry: peer.NewRegistry(),
		Bus:      events.NewBus(),
		Manager:  transfer.NewManager(),
	}
}

func TestControlGetSelf(t *testing.T) {
	s := setupTestService(t)
	handler := NewMux(s)

	req := httptest.NewRequest("GET", "/control/v1/self", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var res map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if res["alias"] != "Test-Device" {
		t.Errorf("expected alias Test-Device, got %v", res["alias"])
	}
	if res["fingerprint"] != s.Identity.Fingerprint {
		t.Errorf("expected fingerprint %s, got %v", s.Identity.Fingerprint, res["fingerprint"])
	}
}

func TestControlGetPeersEmpty(t *testing.T) {
	s := setupTestService(t)
	handler := NewMux(s)

	req := httptest.NewRequest("GET", "/control/v1/peers", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var peers []any
	if err := json.NewDecoder(rr.Body).Decode(&peers); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(peers) != 0 {
		t.Errorf("expected empty peers list, got %d", len(peers))
	}
}

func TestControlGetTransfersEmpty(t *testing.T) {
	s := setupTestService(t)
	handler := NewMux(s)

	req := httptest.NewRequest("GET", "/control/v1/transfers", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var transfers []any
	if err := json.NewDecoder(rr.Body).Decode(&transfers); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(transfers) != 0 {
		t.Errorf("expected empty transfers list, got %d", len(transfers))
	}
}

func TestControlGetDiagnostics(t *testing.T) {
	s := setupTestService(t)
	handler := NewMux(s)

	req := httptest.NewRequest("GET", "/control/v1/diagnostics", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var diag map[string]any
	if err := json.NewDecoder(rr.Body).Decode(&diag); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if diag["status"] != "ok" {
		t.Errorf("expected status ok, got %v", diag["status"])
	}
}
