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
	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/quotareconcile"
	"github.com/smorad3363/teleproxy/internal/telegrambot"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

const telegramWebhookPath = "/telegram/webhook"

func main() {
	if handled, err := runControlCommand(os.Args[1:]); handled {
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
		return
	}

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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	var proxyClient *telemt.Client
	var quotaRunner *quotareconcile.Runner
	if cfg.TelemtAPIURL != "" {
		client, err := telemt.NewFromTokenFile(cfg.TelemtAPIURL, cfg.TelemtAPITokenFile, 2*time.Second)
		if err != nil {
			return fmt.Errorf("configure Telemt client: %w", err)
		}
		proxyClient = client

		reconciler, err := quotareconcile.NewTelemtReconciler(db, proxyClient, quotareconcile.Options{})
		if err != nil {
			return fmt.Errorf("configure quota reconciler: %w", err)
		}
		quotaRunner, err = quotareconcile.NewRunner(ctx, db, reconciler, quotareconcile.RunnerOptions{Concurrency: cfg.ReconcileConcurrency})
		if err != nil {
			return fmt.Errorf("configure quota reconciliation runner: %w", err)
		}
		defer func() {
			quotaRunner.Stop()
			waitCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := quotaRunner.Wait(waitCtx); err != nil {
				logger.Warn("quota reconciliation runner did not stop cleanly")
			}
		}()

		queueCtx, cancelQueue := context.WithTimeout(context.Background(), 5*time.Second)
		err = quotaRunner.TriggerAll(queueCtx)
		cancelQueue()
		if err != nil {
			return fmt.Errorf("queue startup quota reconciliation: %w", err)
		}
	}

	api := httpapi.NewWithProxyServicesAndForcedJoin(db, httpapi.Options{CookieSecure: cfg.CookieSecure}, proxyClient, quotaRunner)
	var handler http.Handler = api.Handler()
	if cfg.BotTokenFile != "" {
		if proxyClient == nil || quotaRunner == nil {
			return fmt.Errorf("configure Telegram Bot proxy provisioning: Telemt quota reconciliation is required")
		}
		botClient, err := telegrambot.NewFromTokenFile(cfg.BotTokenFile, 3*time.Second)
		if err != nil {
			return fmt.Errorf("configure Telegram Bot client: %w", err)
		}
		webhookSecret, err := telegrambot.LoadWebhookSecretFile(cfg.BotWebhookSecretFile)
		if err != nil {
			return fmt.Errorf("configure Telegram webhook authentication: %w", err)
		}
		provisioner, err := proxyprovision.NewService(db, proxyClient, quotaRunner, nil)
		if err != nil {
			return fmt.Errorf("configure Telegram proxy provisioning: %w", err)
		}
		startApplication, err := telegrambot.NewStartApplicationWithForcedJoin(db, cfg.BotUsername, nil, provisioner, botClient)
		if err != nil {
			return fmt.Errorf("configure Telegram start application: %w", err)
		}
		webhook, err := telegrambot.NewWebhookHandler(webhookSecret, startApplication, botClient)
		if err != nil {
			return fmt.Errorf("configure Telegram webhook: %w", err)
		}
		root := http.NewServeMux()
		root.Handle(telegramWebhookPath, webhook)
		root.Handle("/", handler)
		handler = root
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

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
		if quotaRunner != nil {
			quotaRunner.Stop()
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}

	return <-errCh
}
