package handler

import (
	"testing"
	"time"
)

func TestAllowRequest(t *testing.T) {
	wl := hotlinkPolicy{whitelist: []string{"example.com"}, allowEmptyReferer: true}
	wlStrict := hotlinkPolicy{whitelist: []string{"example.com"}, allowEmptyReferer: false}
	noWL := hotlinkPolicy{whitelist: nil, allowEmptyReferer: true}

	cases := []struct {
		name    string
		dest    string
		referer string
		policy  hotlinkPolicy
		want    bool
	}{
		{"主动打开永远放行", "document", "https://evil.com", wlStrict, true},
		{"空Referer-允许", "image", "", wl, true},
		{"空Referer-拒绝", "image", "", wlStrict, false},
		{"白名单命中", "image", "https://example.com/p", wl, true},
		{"白名单子域命中", "image", "https://blog.example.com/p", wl, true},
		{"白名单未命中", "image", "https://evil.com/p", wl, false},
		{"白名单为空-不限制", "image", "https://evil.com/p", noWL, true},
		{"Referer解析失败-拒绝", "image", "://bad", wl, false},
		{"子域伪造-后缀不含点不匹配", "image", "https://notexample.com/p", wl, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := allowRequest(tc.dest, tc.referer, tc.policy); got != tc.want {
				t.Fatalf("allowRequest(%q,%q)=%v, want %v", tc.dest, tc.referer, got, tc.want)
			}
		})
	}
}

func TestDirectURLCache(t *testing.T) {
	c := NewDirectURLCache(50 * time.Millisecond)
	if _, ok := c.Get("f1"); ok {
		t.Fatal("空缓存不应命中")
	}
	c.Set("f1", "http://cdn/x")
	if url, ok := c.Get("f1"); !ok || url != "http://cdn/x" {
		t.Fatalf("命中失败: %q %v", url, ok)
	}
	time.Sleep(60 * time.Millisecond)
	if _, ok := c.Get("f1"); ok {
		t.Fatal("过期后不应命中")
	}

	zero := NewDirectURLCache(0)
	zero.Set("f1", "http://cdn/x")
	if _, ok := zero.Get("f1"); ok {
		t.Fatal("TTL=0 应恒 miss")
	}
}

func TestDirectRateLimiter(t *testing.T) {
	l := newDirectRateLimiter()
	if !l.allow("1.1.1.1", 0) {
		t.Fatal("limit=0 应不限速")
	}
	for i := 0; i < 3; i++ {
		if !l.allow("2.2.2.2", 3) {
			t.Fatalf("第 %d 次应放行", i+1)
		}
	}
	if l.allow("2.2.2.2", 3) {
		t.Fatal("第 4 次应被限")
	}
	if !l.allow("3.3.3.3", 3) {
		t.Fatal("其他 IP 不应受影响")
	}
}
