package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/smorad3363/teleproxy/internal/botcontent"
)

const maxBotContentBodyBytes int64 = 20 << 10

type botContentPayload struct {
	Text string `json:"text"`
}

func (s *Server) registerBotContentRoutes() {
	s.mux.HandleFunc("GET /api/bot-content", s.handleBotContentList)
	s.mux.HandleFunc("PUT /api/bot-content/{slot}", s.handleBotContentPut)
	s.mux.HandleFunc("DELETE /api/bot-content/{slot}", s.handleBotContentDelete)
}

func (s *Server) handleBotContentList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	entries, err := botcontent.List(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"content": entries})
}

func (s *Server) handleBotContentPut(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	payload, ok := decodeBotContentPayload(w, r)
	if !ok {
		return
	}
	entry, err := botcontent.Set(r.Context(), s.db, botcontent.Slot(r.PathValue("slot")), payload.Text, time.Now().UTC())
	if errors.Is(err, botcontent.ErrInvalid) {
		writeBotContentProblem(w, r, http.StatusBadRequest, "BOT_CONTENT_INVALID", "The Bot Content request is invalid.")
		return
	}
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"content": entry})
}

func (s *Server) handleBotContentDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	err := botcontent.Clear(r.Context(), s.db, botcontent.Slot(r.PathValue("slot")))
	switch {
	case errors.Is(err, botcontent.ErrInvalid):
		writeBotContentProblem(w, r, http.StatusBadRequest, "BOT_CONTENT_INVALID", "The Bot Content slot is invalid.")
		return
	case errors.Is(err, botcontent.ErrNotFound):
		writeBotContentProblem(w, r, http.StatusNotFound, "BOT_CONTENT_NOT_FOUND", "The Bot Content override was not found.")
		return
	case err != nil:
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

func decodeBotContentPayload(w http.ResponseWriter, r *http.Request) (botContentPayload, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBotContentBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload botContentPayload
	if err := decoder.Decode(&payload); err != nil {
		if isBotContentBodyTooLarge(err) {
			writeBotContentProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return botContentPayload{}, false
		}
		writeBotContentProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return botContentPayload{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if isBotContentBodyTooLarge(err) {
			writeBotContentProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return botContentPayload{}, false
		}
		writeBotContentProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return botContentPayload{}, false
	}
	return payload, true
}

func isBotContentBodyTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge)
}

func writeBotContentProblem(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Code: code, Message: message, Instance: r.URL.Path})
}
