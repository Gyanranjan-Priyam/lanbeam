package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadAndDefaults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "lanbeam-config-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "config.json")

	// First load should create file with defaults
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.LANPort != 42100 {
		t.Errorf("expected LANPort 42100, got %d", cfg.LANPort)
	}
	if cfg.ControlPort != 42101 {
		t.Errorf("expected ControlPort 42101, got %d", cfg.ControlPort)
	}
	if cfg.MaxConcurrentFiles != 3 {
		t.Errorf("expected MaxConcurrentFiles 3, got %d", cfg.MaxConcurrentFiles)
	}

	// Verify file was written
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file was not created: %v", err)
	}

	// Modifying and saving
	cfg.Alias = "custom-test-node"
	if err := save(cfg, configPath); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Reloading should reflect change
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("reload failed: %v", err)
	}
	if loaded.Alias != "custom-test-node" {
		t.Errorf("expected Alias custom-test-node, got %s", loaded.Alias)
	}
}
