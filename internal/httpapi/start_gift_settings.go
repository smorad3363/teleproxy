package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/smorad3363/teleproxy/internal/settings"
)

const maxStartGiftSettingsBodyBytes int64 = 4 << 10

type startGiftSettingsPayload struct {
	Bytes int64 `json:"bytes"`
}

func (s *Server) registerStartGiftSettingsRoutes() {
	s.mux.HandleFunc("GET /settings", s.handleStartGiftSettingsPage)
	s.mux.HandleFunc("GET /api/settings/start-gift", s.handleStartGiftSettingsGet)
	s.mux.HandleFunc("PUT /api/settings/start-gift", s.handleStartGiftSettingsPut)
}

func (s *Server) handleStartGiftSettingsGet(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	value, err := settings.StartGiftBytes(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"settings": startGiftSettingsPayload{Bytes: value}})
}

func (s *Server) handleStartGiftSettingsPut(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	payload, ok := decodeStartGiftSettings(w, r)
	if !ok {
		return
	}
	if payload.Bytes <= 0 {
		writeStartGiftSettingsProblem(w, r, http.StatusBadRequest, "START_GIFT_INVALID", "The start gift setting is invalid.")
		return
	}
	if err := settings.SetStartGiftBytes(r.Context(), s.db, payload.Bytes); err != nil {
		s.writeInternalError(w, r)
		return
	}
	value, err := settings.StartGiftBytes(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"settings": startGiftSettingsPayload{Bytes: value}})
}

func decodeStartGiftSettings(w http.ResponseWriter, r *http.Request) (startGiftSettingsPayload, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxStartGiftSettingsBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload startGiftSettingsPayload
	if err := decoder.Decode(&payload); err != nil {
		if isStartGiftSettingsBodyTooLarge(err) {
			writeStartGiftSettingsProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return startGiftSettingsPayload{}, false
		}
		writeStartGiftSettingsProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return startGiftSettingsPayload{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if isStartGiftSettingsBodyTooLarge(err) {
			writeStartGiftSettingsProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return startGiftSettingsPayload{}, false
		}
		writeStartGiftSettingsProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return startGiftSettingsPayload{}, false
	}
	return payload, true
}

func isStartGiftSettingsBodyTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge)
}

func writeStartGiftSettingsProblem(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Code: code, Message: message, Instance: r.URL.Path})
}
