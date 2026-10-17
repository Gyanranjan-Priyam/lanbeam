// Package transfer implements the file transfer session state machine.
// It owns the concepts of sessions, files-within-sessions, progress, and integrity.
package transfer

import (
	"context"
	"sync"
	"time"
)

// Status represents the lifecycle state of a session or file.
type Status string

const (
	StatusOffered     Status = "offered"
	StatusTransferring Status = "transferring"
	StatusVerifying   Status = "verifying"
	StatusCompleted   Status = "completed"
	StatusRejected    Status = "rejected"
	StatusFailed      Status = "failed"
	StatusCancelled   Status = "cancelled"
)

// FileInfo describes a single file within a transfer offer.
type FileInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	RelPath string `json:"rel_path,omitempty"`
	Size    int64  `json:"size"`
	MIME    string `json:"mime,omitempty"`
	Hash    string `json:"hash,omitempty"` // "blake3:<hex>" or "sha256:<hex>"
}

// SenderInfo identifies the remote sender.
type SenderInfo struct {
	Alias       string `json:"alias"`
	DeviceType  string `json:"device_type"`
	Fingerprint string `json:"fingerprint"`
}

// Session represents a single transfer session (one offer, potentially many files).
type Session struct {
	ID        string
	Direction string // "send" | "receive"
	Sender    SenderInfo
	Files     []*FileTransfer
	Status    Status
	CreatedAt time.Time

	mu     sync.Mutex
	cancel context.CancelFunc
}

// FileTransfer tracks the state of a single file within a session.
type FileTransfer struct {
	Info      FileInfo
	Status    Status
	BytesDone int64
	SavedPath string
	PartPath  string
	Token     string

	mu sync.Mutex
}

// ReportProgress atomically updates bytes done.
func (f *FileTransfer) ReportProgress(n int64) {
	f.mu.Lock()
	f.BytesDone += n
	f.mu.Unlock()
}

// Manager is the top-level service for managing sessions.
// It will be injected with a store for persistence in later phases.
type Manager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
}

// NewManager creates a Manager with no active sessions.
func NewManager() *Manager {
	return &Manager{
		sessions: make(map[string]*Session),
	}
}

// Get returns a session by ID.
func (m *Manager) Get(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	s, ok := m.sessions[id]
	return s, ok
}

// Register adds a new session.
func (m *Manager) Register(s *Session) {
	m.mu.Lock()
	m.sessions[s.ID] = s
	m.mu.Unlock()
}

// Cancel cancels a session by ID. Idempotent.
func (m *Manager) Cancel(id string) {
	m.mu.RLock()
	s, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
	s.Status = StatusCancelled
}
