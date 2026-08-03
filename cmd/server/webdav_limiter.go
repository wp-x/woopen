package main

import (
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	webdavAuthWindow  = 15 * time.Minute
	maxWebdavFailures = 10
)

type webdavAuthLimiter struct {
	mu       sync.Mutex
	windowAt time.Time
	failures map[string]int
}

func newWebDAVAuthLimiter() *webdavAuthLimiter {
	return &webdavAuthLimiter{windowAt: time.Now(), failures: make(map[string]int)}
}

func (l *webdavAuthLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotateIfNeeded()
	return l.failures[ip] < maxWebdavFailures
}

func (l *webdavAuthLimiter) recordFailure(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rotateIfNeeded()
	l.failures[ip]++
}

func (l *webdavAuthLimiter) reset(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.failures, ip)
}

func (l *webdavAuthLimiter) rotateIfNeeded() {
	if time.Since(l.windowAt) < webdavAuthWindow {
		return
	}
	l.windowAt = time.Now()
	l.failures = make(map[string]int)
}

func remoteClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return r.RemoteAddr
}
