package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/smorad3363/teleproxy/internal/sponsor"
)

const maxSponsorProfileBodyBytes int64 = 8 << 10

type sponsorProfilePayload struct {
	Name       string     `json:"name"`
	ChannelRef string     `json:"channel_ref"`
	AdTag      string     `json:"ad_tag"`
	Enabled    *bool      `json:"enabled"`
	Weight     *int64     `json:"weight"`
	StartsAt   *time.Time `json:"starts_at"`
	EndsAt     *time.Time `json:"ends_at"`
	Notes      string     `json:"notes"`
}

func (s *Server) registerSponsorRoutes() {
	s.mux.HandleFunc("GET /sponsors", s.handleSponsorPage)
	s.mux.HandleFunc("GET /api/sponsors", s.handleSponsorList)
	s.mux.HandleFunc("POST /api/sponsors", s.handleSponsorCreate)
	s.mux.HandleFunc("PUT /api/sponsors/{id}", s.handleSponsorUpdate)
	s.mux.HandleFunc("DELETE /api/sponsors/{id}", s.handleSponsorDelete)
}

func (s *Server) handleSponsorList(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	profiles, err := sponsor.List(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"profiles": profiles})
}

func (s *Server) handleSponsorCreate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	input, ok := decodeSponsorProfileInput(w, r)
	if !ok {
		return
	}
	profile, err := sponsor.Create(r.Context(), s.db, input, time.Now().UTC())
	switch {
	case errors.Is(err, sponsor.ErrConflict):
		writeSponsorProblem(w, r, http.StatusConflict, "SPONSOR_CONFLICT", "A Sponsor Profile with that AdTag already exists.")
		return
	case err != nil:
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusCreated, map[string]any{"profile": profile})
}

func (s *Server) handleSponsorUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	id, ok := sponsorProfileID(w, r)
	if !ok {
		return
	}
	input, ok := decodeSponsorProfileInput(w, r)
	if !ok {
		return
	}
	profile, err := sponsor.Update(r.Context(), s.db, id, input, time.Now().UTC())
	switch {
	case errors.Is(err, sponsor.ErrNotFound):
		writeSponsorProblem(w, r, http.StatusNotFound, "SPONSOR_NOT_FOUND", "The Sponsor Profile was not found.")
		return
	case errors.Is(err, sponsor.ErrConflict):
		writeSponsorProblem(w, r, http.StatusConflict, "SPONSOR_CONFLICT", "A Sponsor Profile with that AdTag already exists.")
		return
	case err != nil:
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"profile": profile})
}

func (s *Server) handleSponsorDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	id, ok := sponsorProfileID(w, r)
	if !ok {
		return
	}
	if err := sponsor.Delete(r.Context(), s.db, id); errors.Is(err, sponsor.ErrNotFound) {
		writeSponsorProblem(w, r, http.StatusNotFound, "SPONSOR_NOT_FOUND", "The Sponsor Profile was not found.")
		return
	} else if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func decodeSponsorProfileInput(w http.ResponseWriter, r *http.Request) (sponsor.CreateProfile, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxSponsorProfileBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload sponsorProfilePayload
	if err := decoder.Decode(&payload); err != nil {
		if isSponsorBodyTooLarge(err) {
			writeSponsorProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return sponsor.CreateProfile{}, false
		}
		writeSponsorProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return sponsor.CreateProfile{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if isSponsorBodyTooLarge(err) {
			writeSponsorProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return sponsor.CreateProfile{}, false
		}
		writeSponsorProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return sponsor.CreateProfile{}, false
	}
	if payload.Enabled == nil || payload.Weight == nil {
		writeSponsorProblem(w, r, http.StatusBadRequest, "SPONSOR_INVALID", "The Sponsor Profile request is invalid.")
		return sponsor.CreateProfile{}, false
	}
	input := sponsor.CreateProfile{
		Name:       payload.Name,
		ChannelRef: payload.ChannelRef,
		AdTag:      payload.AdTag,
		Enabled:    *payload.Enabled,
		Weight:     *payload.Weight,
		StartsAt:   payload.StartsAt,
		EndsAt:     payload.EndsAt,
		Notes:      payload.Notes,
	}
	if err := sponsor.ValidateProfileInput(input); err != nil {
		writeSponsorProblem(w, r, http.StatusBadRequest, "SPONSOR_INVALID", "The Sponsor Profile request is invalid.")
		return sponsor.CreateProfile{}, false
	}
	return input, true
}

func sponsorProfileID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeSponsorProblem(w, r, http.StatusBadRequest, "SPONSOR_INVALID_ID", "The Sponsor Profile identifier is invalid.")
		return 0, false
	}
	return id, true
}

func isSponsorBodyTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge)
}

func writeSponsorProblem(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Code: code, Message: message, Instance: r.URL.Path})
}
