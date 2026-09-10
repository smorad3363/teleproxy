package httpapi

import (
	"sync"
	"time"
)

type loginAttempt struct {
	windowStart time.Time
	failures    int
}

type loginLimiter struct {
	mu      sync.Mutex
	max     int
	window  time.Duration
	entries map[string]loginAttempt
}

func newLoginLimiter(max int, window time.Duration) *loginLimiter {
	if max <= 0 {
		max = 5
	}
	if window <= 0 {
		window = 10 * time.Minute
	}
	return &loginLimiter{max: max, window: window, entries: make(map[string]loginAttempt)}
}

func (l *loginLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.windowStart) >= l.window {
		return true
	}
	return entry.failures < l.max
}

func (l *loginLimiter) Failure(key string, now time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, ok := l.entries[key]
	if !ok || now.Sub(entry.windowStart) >= l.window {
		l.entries[key] = loginAttempt{windowStart: now, failures: 1}
		return
	}
	entry.failures++
	l.entries[key] = entry
}

func (l *loginLimiter) Reset(key string) {
	l.mu.Lock()
	delete(l.entries, key)
	l.mu.Unlock()
}
