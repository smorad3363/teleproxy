package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/smorad3363/teleproxy/internal/settings"
)

const maxReferralRewardSettingsBodyBytes int64 = 4 << 10

type referralRewardSettingsPayload struct {
	Bytes      int64 `json:"bytes"`
	ExpiryDays int64 `json:"expiry_days"`
}

func (s *Server) registerReferralSettingsRoutes() {
	s.mux.HandleFunc("GET /api/referral/reward-settings", s.handleReferralRewardSettingsGet)
	s.mux.HandleFunc("PUT /api/referral/reward-settings", s.handleReferralRewardSettingsPut)
}

func (s *Server) handleReferralRewardSettingsGet(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, false) {
		return
	}
	configured, err := settings.ReferralReward(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"settings": configured})
}

func (s *Server) handleReferralRewardSettingsPut(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdminAPI(w, r, true) {
		return
	}
	payload, ok := decodeReferralRewardSettings(w, r)
	if !ok {
		return
	}
	if payload.Bytes <= 0 || payload.ExpiryDays <= 0 {
		writeReferralSettingsProblem(w, r, http.StatusBadRequest, "REFERRAL_REWARD_INVALID", "The referral reward settings are invalid.")
		return
	}
	if err := settings.SetReferralReward(r.Context(), s.db, payload.Bytes, payload.ExpiryDays); err != nil {
		s.writeInternalError(w, r)
		return
	}
	configured, err := settings.ReferralReward(r.Context(), s.db)
	if err != nil {
		s.writeInternalError(w, r)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"settings": configured})
}

func decodeReferralRewardSettings(w http.ResponseWriter, r *http.Request) (referralRewardSettingsPayload, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxReferralRewardSettingsBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	var payload referralRewardSettingsPayload
	if err := decoder.Decode(&payload); err != nil {
		if isReferralSettingsBodyTooLarge(err) {
			writeReferralSettingsProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return referralRewardSettingsPayload{}, false
		}
		writeReferralSettingsProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return referralRewardSettingsPayload{}, false
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		if isReferralSettingsBodyTooLarge(err) {
			writeReferralSettingsProblem(w, r, http.StatusRequestEntityTooLarge, "REQUEST_TOO_LARGE", "The request body is too large.")
			return referralRewardSettingsPayload{}, false
		}
		writeReferralSettingsProblem(w, r, http.StatusBadRequest, "BAD_REQUEST", "The JSON request is invalid.")
		return referralRewardSettingsPayload{}, false
	}
	return payload, true
}

func isReferralSettingsBodyTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge)
}

func writeReferralSettingsProblem(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeProblem(w, Problem{Type: "about:blank", Title: http.StatusText(status), Status: status, Code: code, Message: message, Instance: r.URL.Path})
}
