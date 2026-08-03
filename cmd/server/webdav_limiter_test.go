package main

import "testing"

func TestWebDAVAuthLimiter(t *testing.T) {
	limiter := newWebDAVAuthLimiter()
	const ip = "192.0.2.2"
	for i := 0; i < maxWebdavFailures; i++ {
		if !limiter.allow(ip) {
			t.Fatalf("第 %d 次尝试不应被阻止", i+1)
		}
		limiter.recordFailure(ip)
	}
	if limiter.allow(ip) {
		t.Fatal("达到失败上限后应被限制")
	}
	limiter.reset(ip)
	if !limiter.allow(ip) {
		t.Fatal("成功认证后应重置限制")
	}
}
