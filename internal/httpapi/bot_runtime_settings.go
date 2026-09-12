package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/smorad3363/teleproxy/internal/settings"
	"github.com/smorad3363/teleproxy/internal/telegrambot"
)

const (
	maxBotRuntimeSettingsBodyBytes int64 = 8 << 10
	defaultBotTokenFile                  = "/run/teleproxy-bot-secrets/token"
	defaultBotWebhookSecretFile          = "/run/teleproxy-bot-secrets/webhook-secret"
	botTestMessage                       = "Teleproxy Bot settings test."
)

type botRuntimeSettingsPayload struct {
	Username    string `json:"username"`
	AdminChatID int64  `json:"admin_chat_id"`
	Enabled     bool   `json:"enabled"`
	Token       string `json:"token"`
}

type botRuntimeSettingsView struct {
	Username        string `json:"username"`
	AdminChatID     int64  `json:"admin_chat_id"`
	Enabled         bool   `json:"enabled"`
	TokenConfigured bool   `json:"token_configured"`
	Configured      bool   `json:"configured"`
	RestartRequired bool   `json:"restart_required"`
}

func (s *Server) handleBotRuntimeSettingsGet(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	view, err := readBotRuntimeSettingsView(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"settings": view})
}

func (s *Server) handleBotRuntimeSettingsPut(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	payload, ok := decodeBotRuntimeSettings(w, r)
	if !ok {
		return
	}
	value := settings.BotRuntimeSettings{
		Username:    settings.NormalizeBotUsername(payload.Username),
		AdminChatID: payload.AdminChatID,
		Enabled:     payload.Enabled,
	}
	if err := settings.ValidateBotRuntime(value); err != nil {
		writeBotRuntimeSettingsProblem(w, r, http.StatusBadRequest, "BOT_SETTINGS_INVALID", "The Telegram Bot settings are invalid.")
		return
	}

	tokenPath, webhookSecretPath := botSecretPaths()
	token := strings.TrimSpace(payload.Token)
	if token != "" {
		if err := telegrambot.StoreTokenFile(tokenPath, token); err != nil {
			writeBotRuntimeSettingsProblem(w, r, http.StatusBadRequest, "BOT_TOKEN_INVALID", "The Telegram Bot token is invalid.")
			return
		}
	}
	tokenConfigured, err := telegrambot.TokenFileConfigured(tokenPath)
	if err != nil {
		tokenConfigured = false
	}
	if value.Enabled && !tokenConfigured {
		writeBotRuntimeSettingsProblem(w, r, http.StatusBadRequest, "BOT_TOKEN_REQUIRED", "A Telegram Bot token is required before enabling the Bot.")
		return
	}
	if value.Enabled {
		if err := telegrambot.EnsureWebhookSecretFile(webhookSecretPath); err != nil {
			s.writeInternalError(w, r)
			return
		}
	}
	stored, err := settings.SetBotRuntime(r.Context(), s.db, value, time.Now().UTC())
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"settings": botRuntimeSettingsView{
		Username:        stored.Username,
		AdminChatID:     stored.AdminChatID,
		Enabled:         stored.Enabled,
		TokenConfigured: tokenConfigured,
		Configured:      true,
		RestartRequired: true,
	}})
}

func (s *Server) handleBotRuntimeSettingsTest(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	value, configured, err := settings.BotRuntime(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	if !configured {
		writeBotRuntimeSettingsProblem(w, r, http.StatusConflict, "BOT_NOT_CONFIGURED", "Save Telegram Bot settings before testing the Bot.")
		return
	}
	tokenPath, _ := botSecretPaths()
	client, err := telegrambot.NewFromTokenFile(tokenPath, 3*time.Second)
	if err != nil {
		writeBotRuntimeSettingsProblem(w, r, http.StatusConflict, "BOT_TOKEN_REQUIRED", "A valid Telegram Bot token is required before testing the Bot.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if _, err := client.SendMessage(ctx, value.AdminChatID, botTestMessage); err != nil {
		writeBotRuntimeSettingsProblem(w, r, http.StatusBadGateway, "BOT_TEST_FAILED", "Telegram Bot test failed.")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func readBotRuntimeSettingsView(ctx context.Context, db *sql.DB) (botRuntimeSettingsView, error) {
	value, configured, err := settings.BotRuntime(ctx, db)
	if err != nil {
		return botRuntimeSettingsView{}, err
	}
	tokenPath, _ := botSecretPaths()
	tokenConfigured, err := telegrambot.TokenFileConfigured(tokenPath)
	if err != nil {
		tokenConfigured = false
	}
	return botRuntimeSettingsView{
		Username:        value.Username,
		AdminChatID:     value.AdminChatID,
		Enabled:         value.Enabled,
		TokenConfigured: tokenConfigured,
		Configured:      configured,
	}, nil
}

func botSecretPaths() (string, string) {
	tokenPath := strings.TrimSpace(os.Getenv("TPROXY_BOT_TOKEN_FILE"))
	if tokenPath == "" {
		tokenPath = defaultBotTokenFile
	}
	secretPath := strings.TrimSpace(os.Getenv("TPROXY_BOT_WEBHOOK_SECRET_FILE"))
	if secretPath == "" {
		secretPath = defaultBotWebhookSecretFile
	}
	return tokenPath, secretPath
}

func decodeBotRuntimeSettings(w http.ResponseWriter, r *http.Request) (botRuntimeSettingsPayload, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBotRuntimeSettingsBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload botRuntimeSettingsPayload
	if err := decoder.Decode(&payload); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeBotRuntimeSettingsProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return botRuntimeSettingsPayload{}, false
		}
		writeBotRuntimeSettingsProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return botRuntimeSettingsPayload{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeBotRuntimeSettingsProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return botRuntimeSettingsPayload{}, false
	}
	return payload, true
}

func writeBotRuntimeSettingsProblem(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Code: code, Message: message, Instance: r.URL.Path})
}
