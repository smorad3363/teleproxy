package telemt

import (
	"context"
	"net/http"
	"strings"
)

func (c *Client) CreateUserWithSecret(ctx context.Context, username string, enabled bool, secret string) (Credential, error) {
	if err := validateUsername(username); err != nil {
		return Credential{}, err
	}
	if !validHex(secret, 16) {
		return Credential{}, &APIError{Code: FailureRejected}
	}
	body := struct {
		Username string `json:"username"`
		Secret   string `json:"secret"`
		Enabled  bool   `json:"enabled"`
	}{Username: username, Secret: secret, Enabled: enabled}
	var credential Credential
	if err := c.doJSON(ctx, http.MethodPost, "/v1/users", body, []int{http.StatusCreated, http.StatusAccepted}, &credential); err != nil {
		return Credential{}, err
	}
	if err := validateCredential(credential, username); err != nil {
		return Credential{}, err
	}
	if !strings.EqualFold(credential.Secret, secret) {
		return Credential{}, &APIError{Code: FailureInvalidOutput}
	}
	return credential, nil
}
