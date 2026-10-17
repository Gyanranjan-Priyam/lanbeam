// Package main is the entry point for the lanbeam binary.
// It wires up all services and starts the listeners.
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/lanbeam/lanbeam/core/internal/api/controlapi"
	"github.com/lanbeam/lanbeam/core/internal/api/middleware"
	"github.com/lanbeam/lanbeam/core/internal/api/peerapi"
	"github.com/lanbeam/lanbeam/core/internal/config"
	"github.com/lanbeam/lanbeam/core/internal/crypto"
	"github.com/lanbeam/lanbeam/core/internal/events"
	"github.com/lanbeam/lanbeam/core/internal/peer"
	"github.com/lanbeam/lanbeam/core/internal/transfer"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "", "path to config file (default: OS config dir)")
	logLevel := flag.String("log", "info", "log level: debug, info, warn, error")
	flag.Parse()

	// Set up structured logging
	var level slog.Level
	if err := level.UnmarshalText([]byte(*logLevel)); err != nil {
		level = slog.LevelInfo
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	slog.Info("lanbeam starting",
		"alias", cfg.Alias,
		"lan_port", cfg.LANPort,
		"control_port", cfg.ControlPort,
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

// run starts all services and blocks until ctx is cancelled.
func run(ctx context.Context, cfg *config.Config) error {
	id, err := crypto.LoadOrCreate(cfg.DataDir)
	if err != nil {
		return fmt.Errorf("failed to load/create identity: %w", err)
	}
	slog.Info("identity initialized", "fingerprint", id.Fingerprint)

	reg := peer.NewRegistry()
	bus := events.NewBus()
	mgr := transfer.NewManager()

	// Control API (Localhost only)
	ctrlSvc := &controlapi.Service{
		Config:   cfg,
		Identity: id,
		Registry: reg,
		Bus:      bus,
		Manager:  mgr,
	}
	ctrlHandler := controlapi.NewMux(ctrlSvc)
	ctrlAddr := fmt.Sprintf("127.0.0.1:%d", cfg.ControlPort)
	ctrlServer := &http.Server{
		Addr:              ctrlAddr,
		Handler:           middleware.Chain(ctrlHandler, middleware.Logger, middleware.Recover),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Peer API (LAN facing, TLS pinned)
	peerSvc := &peerapi.Service{
		Config:   cfg,
		Identity: id,
		Registry: reg,
		Bus:      bus,
		Manager:  mgr,
		Info: peerapi.InfoResponse{
			ProtocolVersion:    1,
			AppVersion:         "0.1.0",
			Alias:              cfg.Alias,
			DeviceType:         "desktop",
			Fingerprint:        id.Fingerprint,
			AcceptingTransfers: true,
		},
	}
	peerHandler := peerapi.NewMux(peerSvc)
	lanAddr := fmt.Sprintf("0.0.0.0:%d", cfg.LANPort)
	peerServer := &http.Server{
		Addr:              lanAddr,
		Handler:           middleware.Chain(peerHandler, middleware.Logger, middleware.Recover),
		ReadHeaderTimeout: 5 * time.Second,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{id.Cert},
			MinVersion:   tls.VersionTLS12,
		},
	}

	errCh := make(chan error, 2)

	go func() {
		slog.Info("control server listening", "url", "http://"+ctrlAddr)
		if err := ctrlServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("control server failed: %w", err)
		}
	}()

	go func() {
		slog.Info("lan peer server listening (TLS)", "url", "https://"+lanAddr)
		if err := peerServer.ListenAndServeTLS("", ""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("lan peer server failed: %w", err)
		}
	}()

	slog.Info("lanbeam ready — listening for peers and UI control")

	select {
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	case err := <-errCh:
		slog.Error("server startup error", "err", err)
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = ctrlServer.Shutdown(shutdownCtx)
	_ = peerServer.Shutdown(shutdownCtx)
	slog.Info("servers stopped gracefully")
	return nil
}
