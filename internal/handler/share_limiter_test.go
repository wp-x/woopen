package handler

import "testing"

func TestSharePasswordLimiter(t *testing.T) {
	limiter := newSharePasswordLimiter()
	key := "198.51.100.10|share-code"
	for i := 0; i < maxSharePasswordFailures; i++ {
		if !limiter.allow(key) {
			t.Fatalf("attempt %d should be allowed", i)
		}
		limiter.recordFailure(key)
	}
	if limiter.allow(key) {
		t.Fatal("attempt after the limit should be rejected")
	}
	limiter.reset(key)
	if !limiter.allow(key) {
		t.Fatal("successful authentication should clear failures")
	}
}
