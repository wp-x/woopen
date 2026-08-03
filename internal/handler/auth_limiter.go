package handler

import (
	"sync"
	"time"
)

const (
	loginFailureWindow = 15 * time.Minute
	maxLoginFailures   = 10
)

type loginFailureLimiter struct {
	mu       sync.Mutex
	windowAt time.Time
	failures map[string]int
}

func newLoginFailureLimiter() *loginFailureLimiter {
	return &loginFailureLimiter{windowAt: time.Now(), failures: make(map[string]int)}
}

func (l *loginFailureLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotateIfNeeded()
	return l.failures[ip] < maxLoginFailures
}

func (l *loginFailureLimiter) recordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotateIfNeeded()
	l.failures[ip]++
}

func (l *loginFailureLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, ip)
}

func (l *loginFailureLimiter) rotateIfNeeded() {
	if time.Since(l.windowAt) < loginFailureWindow {
		return
	}
	l.windowAt = time.Now()
	l.failures = make(map[string]int)
}
