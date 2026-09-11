package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	controlHealthcheckTimeout   = 3 * time.Second
	controlHealthcheckBodyLimit = 4096
)

func runControlCommand(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	if len(args) != 2 || args[0] != "healthcheck" {
		return true, errors.New("invalid control command")
	}
	return true, runHealthcheck(args[1])
}

func runHealthcheck(rawURL string) error {
	return runHealthcheckWithTimeout(rawURL, controlHealthcheckTimeout)
}

func runHealthcheckWithTimeout(rawURL string, timeout time.Duration) error {
	if timeout <= 0 {
		return errors.New("control healthcheck timeout is invalid")
	}
	parsed, err := url.ParseRequestURI(rawURL)
	if err != nil || parsed.Scheme != "http" || parsed.Host == "" || parsed.User != nil {
		return errors.New("control healthcheck URL is invalid")
	}

	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	response, err := client.Get(rawURL)
	if err != nil {
		return errors.New("control healthcheck request failed")
	}
	defer response.Body.Close()

	if _, err := io.Copy(io.Discard, io.LimitReader(response.Body, controlHealthcheckBodyLimit)); err != nil {
		return errors.New("control healthcheck response failed")
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("control healthcheck returned HTTP %d", response.StatusCode)
	}
	return nil
}
