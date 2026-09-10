package telemt

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type TLSDomainLink struct {
	Domain string `json:"domain"`
	Link   string `json:"link"`
}

type UserLinks struct {
	Classic    []string        `json:"classic"`
	Secure     []string        `json:"secure"`
	TLS        []string        `json:"tls"`
	TLSDomains []TLSDomainLink `json:"tls_domains"`
}

func (c *Client) GetUserLinks(ctx context.Context, username string) (UserLinks, error) {
	if err := validateUsername(username); err != nil {
		return UserLinks{}, err
	}
	var view struct {
		Username string    `json:"username"`
		Links    UserLinks `json:"links"`
	}
	path := "/v1/users/" + url.PathEscape(username)
	if err := c.doJSON(ctx, http.MethodGet, path, nil, []int{http.StatusOK}, &view); err != nil {
		return UserLinks{}, err
	}
	if view.Username != username || validateLinks(view.Links) != nil {
		return UserLinks{}, &APIError{Code: FailureInvalidOutput}
	}
	return view.Links, nil
}

func (links UserLinks) All() []string {
	result := make([]string, 0, len(links.Classic)+len(links.Secure)+len(links.TLS)+len(links.TLSDomains))
	result = append(result, links.Classic...)
	result = append(result, links.Secure...)
	result = append(result, links.TLS...)
	for _, entry := range links.TLSDomains {
		result = append(result, entry.Link)
	}
	return result
}

func validateLinks(links UserLinks) error {
	if len(links.Classic) > 64 || len(links.Secure) > 64 || len(links.TLS) > 64 || len(links.TLSDomains) > 64 {
		return fmt.Errorf("too many Telemt user links")
	}
	for _, link := range links.Classic {
		if err := validateProxyLink(link); err != nil {
			return err
		}
	}
	for _, link := range links.Secure {
		if err := validateProxyLink(link); err != nil {
			return err
		}
	}
	for _, link := range links.TLS {
		if err := validateProxyLink(link); err != nil {
			return err
		}
	}
	for _, entry := range links.TLSDomains {
		if entry.Domain == "" || len(entry.Domain) > 253 {
			return fmt.Errorf("invalid Telemt TLS domain link")
		}
		if err := validateProxyLink(entry.Link); err != nil {
			return err
		}
	}
	return nil
}

func validateProxyLink(raw string) error {
	if len(raw) < 1 || len(raw) > 4096 {
		return fmt.Errorf("invalid Telemt proxy link")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "tg" || parsed.Host != "proxy" || parsed.Fragment != "" || parsed.User != nil {
		return fmt.Errorf("invalid Telemt proxy link")
	}
	query := parsed.Query()
	if strings.TrimSpace(query.Get("server")) == "" {
		return fmt.Errorf("invalid Telemt proxy link")
	}
	port, err := strconv.ParseUint(query.Get("port"), 10, 16)
	if err != nil || port == 0 {
		return fmt.Errorf("invalid Telemt proxy link")
	}
	secret := query.Get("secret")
	if len(secret) < 32 || len(secret) > 512 {
		return fmt.Errorf("invalid Telemt proxy link")
	}
	for _, ch := range secret {
		if (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F') || (ch >= '0' && ch <= '9') {
			continue
		}
		return fmt.Errorf("invalid Telemt proxy link")
	}
	return nil
}
