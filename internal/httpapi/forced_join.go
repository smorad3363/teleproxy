package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/smorad3363/teleproxy/internal/forcedjoin"
	"github.com/smorad3363/teleproxy/internal/telemt"
)

const maxForcedJoinBodyBytes int64 = 16 << 10

type forcedJoinPayload struct {
	ChatRef     string `json:"chat_ref"`
	DisplayName string `json:"display_name"`
	JoinURL     string `json:"join_url"`
	Enabled     *bool  `json:"enabled"`
	Required    *bool  `json:"required"`
	Position    *int   `json:"position"`
	CustomText  string `json:"custom_text"`
}

func NewWithProxyServicesAndForcedJoin(db *sql.DB, options Options, client *telemt.Client, trigger quotaReconcileTrigger) *Server {
	s := NewWithProxyServices(db, options, client, trigger)
	s.registerForcedJoinRoutes()
	s.registerReferralSettingsRoutes()
	return s
}

func (s *Server) registerForcedJoinRoutes() {
	s.mux.HandleFunc("GET /api/forced-join/channels", s.handleForcedJoinList)
	s.mux.HandleFunc("POST /api/forced-join/channels", s.handleForcedJoinCreate)
	s.mux.HandleFunc("PUT /api/forced-join/channels/{id}", s.handleForcedJoinUpdate)
	s.mux.HandleFunc("DELETE /api/forced-join/channels/{id}", s.handleForcedJoinDelete)
}

func (s *Server) handleForcedJoinList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	channels, err := forcedjoin.List(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"channels": channels})
}

func (s *Server) handleForcedJoinCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	input, ok := decodeForcedJoinInput(w, r)
	if !ok {
		return
	}
	channel, err := forcedjoin.Create(r.Context(), s.db, input, time.Now().UTC())
	if errors.Is(err, forcedjoin.ErrConflict) {
		writeForcedJoinProblem(w, r, http.StatusConflict, "FORCED_JOIN_CONFLICT", "A Forced Join channel with that Telegram chat reference already exists.")
		return
	}
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"channel": channel})
}

func (s *Server) handleForcedJoinUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	id, ok := forcedJoinID(w, r)
	if !ok {
		return
	}
	input, ok := decodeForcedJoinInput(w, r)
	if !ok {
		return
	}
	channel, err := forcedjoin.Update(r.Context(), s.db, id, input, time.Now().UTC())
	switch {
	case errors.Is(err, forcedjoin.ErrNotFound):
		writeForcedJoinProblem(w, r, http.StatusNotFound, "FORCED_JOIN_NOT_FOUND", "The Forced Join channel was not found.")
		return
	case errors.Is(err, forcedjoin.ErrConflict):
		writeForcedJoinProblem(w, r, http.StatusConflict, "FORCED_JOIN_CONFLICT", "A Forced Join channel with that Telegram chat reference already exists.")
		return
	case err != nil:
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"channel": channel})
}

func (s *Server) handleForcedJoinDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	id, ok := forcedJoinID(w, r)
	if !ok {
		return
	}
	if err := forcedjoin.Delete(r.Context(), s.db, id); errors.Is(err, forcedjoin.ErrNotFound) {
		writeForcedJoinProblem(w, r, http.StatusNotFound, "FORCED_JOIN_NOT_FOUND", "The Forced Join channel was not found.")
		return
	} else if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeForcedJoinInput(w http.ResponseWriter, r *http.Request) (forcedjoin.CreateChannel, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxForcedJoinBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload forcedJoinPayload
	if err := decoder.Decode(&payload); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			writeForcedJoinProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return forcedjoin.CreateChannel{}, false
		}
		writeForcedJoinProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return forcedjoin.CreateChannel{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		writeForcedJoinProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return forcedjoin.CreateChannel{}, false
	}
	if payload.Enabled == nil || payload.Required == nil || payload.Position == nil {
		writeForcedJoinProblem(w, r, http.StatusBadRequest, "FORCED_JOIN_INVALID", "The Forced Join channel request is invalid.")
		return forcedjoin.CreateChannel{}, false
	}
	input := forcedjoin.CreateChannel{
		ChatRef: payload.ChatRef, DisplayName: payload.DisplayName, JoinURL: payload.JoinURL,
		Enabled: *payload.Enabled, Required: *payload.Required, Position: *payload.Position, CustomText: payload.CustomText,
	}
	if err := forcedjoin.ValidateChannelInput(input); err != nil {
		writeForcedJoinProblem(w, r, http.StatusBadRequest, "FORCED_JOIN_INVALID", "The Forced Join channel request is invalid.")
		return forcedjoin.CreateChannel{}, false
	}
	return input, true
}

func forcedJoinID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeForcedJoinProblem(w, r, http.StatusBadRequest, "FORCED_JOIN_INVALID_ID", "The Forced Join channel identifier is invalid.")
		return 0, false
	}
	return id, true
}

func writeForcedJoinProblem(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Code: code, Message: message, Instance: r.URL.Path})
}
