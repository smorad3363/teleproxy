package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/smorad3363/teleproxy/internal/admin"
	"github.com/smorad3363/teleproxy/internal/config"
	"github.com/smorad3363/teleproxy/internal/database"
	"github.com/smorad3363/teleproxy/internal/httpapi"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(logger); err != nil {
		logger.Error("control plane stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	startupCtx, cancelStartup := context.WithTimeout(context.Background(), 30*time.Second)
	db, err := database.Open(startupCtx, cfg.DatabasePath)
	if err != nil {
		cancelStartup()
		return err
	}
	defer db.Close()

	administrator, created, err := admin.BootstrapOwnerFromFile(
		startupCtx,
		db,
		cfg.BootstrapAdminUser,
		cfg.BootstrapPasswordFile,
	)
	cancelStartup()
	if err != nil {
		return err
	}
	if created {
		logger.Info("initial administrator created", "username", administrator.Username)
	}

	var proxyClient *telemt.Client
	if cfg.TelemtAPIURL != "" {
		client, err := telemt.NewFromTokenFile(cfg.TelemtAPIURL, cfg.TelemtAPITokenFile, 2*time.Second)
		if err != nil {
			return fmt.Errorf("configure Telemt client: %w", err)
		}
		proxyClient = client
	}

	api := httpapi.NewWithProxyClient(db, httpapi.Options{CookieSecure: cfg.CookieSecure}, proxyClient)
	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           api.Handler(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		logger.Info("control plane listening", "addr", cfg.HTTPAddr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return <-errCh
}
