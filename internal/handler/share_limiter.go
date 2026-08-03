package handler

import (
	"sync"
	"time"
)

const (
	sharePasswordWindow      = 15 * time.Minute
	maxSharePasswordFailures = 10
)

type sharePasswordLimiter struct {
	mu       sync.Mutex
	windowAt time.Time
	failures map[string]int
}

func newSharePasswordLimiter() *sharePasswordLimiter {
	return &sharePasswordLimiter{windowAt: time.Now(), failures: make(map[string]int)}
}

func (l *sharePasswordLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotateIfNeeded()
	return l.failures[key] < maxSharePasswordFailures
}

func (l *sharePasswordLimiter) recordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotateIfNeeded()
	l.failures[key]++
}

func (l *sharePasswordLimiter) reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, key)
}

func (l *sharePasswordLimiter) rotateIfNeeded() {
	if time.Since(l.windowAt) < sharePasswordWindow {
		return
	}
	l.windowAt = time.Now()
	l.failures = make(map[string]int)
}
