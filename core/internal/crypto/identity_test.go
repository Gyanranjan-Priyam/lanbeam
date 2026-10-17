package crypto

import (
	"os"
	"testing"
)

func TestNewIdentity(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	if id.Fingerprint == "" {
		t.Fatal("expected non-empty fingerprint")
	}
	if len(id.Fingerprint) != 64 {
		t.Fatalf("expected 64-char hex fingerprint, got %d", len(id.Fingerprint))
	}
	if len(id.Cert.Certificate) == 0 {
		t.Fatal("expected certificate in tls.Certificate")
	}
}

func TestLoadOrCreate(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lanbeam-crypto-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// First load: creates new files
	id1, err := LoadOrCreate(tempDir)
	if err != nil {
		t.Fatalf("initial LoadOrCreate failed: %v", err)
	}

	// Second load: loads existing files
	id2, err := LoadOrCreate(tempDir)
	if err != nil {
		t.Fatalf("second LoadOrCreate failed: %v", err)
	}

	if id1.Fingerprint != id2.Fingerprint {
		t.Fatalf("expected stable fingerprint across loads: %s != %s", id1.Fingerprint, id2.Fingerprint)
	}
}

func TestPinnedTLSConfig(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatal(err)
	}

	cfg := PinnedTLSConfig(id.Fingerprint)
	if !cfg.InsecureSkipVerify {
		t.Fatal("expected InsecureSkipVerify=true")
	}
	if cfg.VerifyPeerCertificate == nil {
		t.Fatal("expected VerifyPeerCertificate callback to be set")
	}

	// Should pass with matching certificate
	err = cfg.VerifyPeerCertificate(id.Cert.Certificate, nil)
	if err != nil {
		t.Fatalf("expected peer cert verification to succeed, got %v", err)
	}

	// Should fail with mismatched fingerprint
	mismatchedCfg := PinnedTLSConfig("0000000000000000000000000000000000000000000000000000000000000000")
	err = mismatchedCfg.VerifyPeerCertificate(id.Cert.Certificate, nil)
	if err == nil {
		t.Fatal("expected verification failure for mismatched fingerprint")
	}
}
