// Package config handles loading, saving, and defaulting of application configuration.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Config holds all runtime configuration for lanbeam.
type Config struct {
	// Alias is the human-readable name shown to other devices.
	Alias string `json:"alias"`

	// LANPort is the TLS port for peer and guest traffic (default 42100).
	LANPort int `json:"lan_port"`

	// ControlPort is the localhost-only control API port (default 42101).
	ControlPort int `json:"control_port"`

	// DownloadDir is where received files are saved.
	DownloadDir string `json:"download_dir"`

	// AutoAcceptTrusted, if true, skips the approval prompt for trusted devices.
	AutoAcceptTrusted bool `json:"auto_accept_trusted"`

	// MaxConcurrentFiles is the number of files transferred simultaneously per session.
	MaxConcurrentFiles int `json:"max_concurrent_files"`

	// OfferTimeoutSec is how many seconds an incoming offer waits before expiring.
	OfferTimeoutSec int `json:"offer_timeout_s"`

	// Interface is the network interface to bind to. Empty = auto.
	Interface string `json:"interface"`

	// DataDir is the directory for the SQLite database, identity keys, and other state.
	DataDir string `json:"data_dir"`
}

// Load reads the config from the given path (or the default location if empty).
// If no file exists, it writes a default config and returns it.
func Load(path string) (*Config, error) {
	if path == "" {
		var err error
		path, err = defaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	cfg := defaults()

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		// First run: write defaults.
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return nil, err
		}
		return cfg, save(cfg, path)
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// save writes cfg to path as pretty-printed JSON.
func save(cfg *Config, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

// defaults returns a Config populated with sensible defaults.
func defaults() *Config {
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "lanbeam-device"
	}

	return &Config{
		Alias:              hostname,
		LANPort:            42100,
		ControlPort:        42101,
		DownloadDir:        defaultDownloadDir(),
		AutoAcceptTrusted:  false,
		MaxConcurrentFiles: 3,
		OfferTimeoutSec:    60,
		Interface:          "",
		DataDir:            defaultDataDir(),
	}
}

func defaultConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "lanbeam", "config.json"), nil
}

func defaultDataDir() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "."
	}
	return filepath.Join(dir, "lanbeam")
}

func defaultDownloadDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "Downloads")
	default:
		return filepath.Join(home, "Downloads")
	}
}
