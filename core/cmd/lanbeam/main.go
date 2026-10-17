// Package main is the entry point for the lanbeam binary.
// It wires up all services and starts the listeners.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/lanbeam/lanbeam/core/internal/config"
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
	// TODO: Phase 1 — wire crypto, store, event bus, transfer engine, peer API, control API
	// TODO: Phase 2 — wire discovery services
	// TODO: Phase 3 — embed UI assets

	slog.Info("lanbeam ready — waiting for shutdown signal")
	<-ctx.Done()
	slog.Info("shutting down")
	return nil
}
