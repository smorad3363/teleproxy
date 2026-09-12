package main

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/proxyprovision"
	"github.com/smorad3363/teleproxy/internal/quotareconcile"
	"github.com/smorad3363/teleproxy/internal/settings"
	"github.com/smorad3363/teleproxy/internal/telegrambot"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

const (
	defaultRuntimeBotTokenFile         = "/run/teleproxy-bot-secrets/token"
	defaultRuntimeBotWebhookSecretFile = "/run/teleproxy-bot-secrets/webhook-secret"
)

func configuredTelegramWebhook(ctx context.Context, db *sql.DB, legacyUsername string, proxyClient *telemt.Client, quotaRunner *quotareconcile.Runner) (http.Handler, bool, error) {
	value, configured, err := settings.BotRuntime(ctx, db)
	if err != nil {
		return nil, false, fmt.Errorf("load Telegram Bot runtime settings: %w", err)
	}
	username := ""
	if configured {
		if !value.Enabled {
			return nil, false, nil
		}
		username = value.Username
	} else {
		username = settings.NormalizeBotUsername(legacyUsername)
		if username == "" {
			return nil, false, nil
		}
	}
	if proxyClient == nil || quotaRunner == nil {
		return nil, false, fmt.Errorf("configure Telegram Bot proxy provisioning: Telemt quota reconciliation is required")
	}

	tokenFile, secretFile := runtimeBotSecretPaths()
	botClient, err := telegrambot.NewFromTokenFile(tokenFile, 3*time.Second)
	if err != nil {
		return nil, false, fmt.Errorf("configure Telegram Bot client: %w", err)
	}
	webhookSecret, err := telegrambot.LoadWebhookSecretFile(secretFile)
	if err != nil {
		return nil, false, fmt.Errorf("configure Telegram webhook authentication: %w", err)
	}
	provisioner, err := proxyprovision.NewService(db, proxyClient, quotaRunner, nil)
	if err != nil {
		return nil, false, fmt.Errorf("configure Telegram proxy provisioning: %w", err)
	}
	startApplication, err := telegrambot.NewStartApplicationWithForcedJoin(db, username, nil, provisioner, botClient)
	if err != nil {
		return nil, false, fmt.Errorf("configure Telegram start application: %w", err)
	}
	webhook, err := telegrambot.NewWebhookHandler(webhookSecret, startApplication, botClient)
	if err != nil {
		return nil, false, fmt.Errorf("configure Telegram webhook: %w", err)
	}
	return webhook, true, nil
}

func runtimeBotSecretPaths() (string, string) {
	tokenFile := strings.TrimSpace(os.Getenv("TPROXY_BOT_TOKEN_FILE"))
	if tokenFile == "" {
		tokenFile = defaultRuntimeBotTokenFile
	}
	secretFile := strings.TrimSpace(os.Getenv("TPROXY_BOT_WEBHOOK_SECRET_FILE"))
	if secretFile == "" {
		secretFile = defaultRuntimeBotWebhookSecretFile
	}
	return tokenFile, secretFile
}
