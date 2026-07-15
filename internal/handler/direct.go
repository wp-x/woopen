package handler

import (
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"woopen/internal/model"

	"github.com/gin-gonic/gin"
)

// hotlinkPolicy 防盗链策略（从 settings 派生）。
type hotlinkPolicy struct {
	whitelist         []string // 已 split + trim + 小写；空=不限制
	allowEmptyReferer bool
}

// allowRequest 判定一次直连请求是否放行。纯函数，便于单测。
// 规则顺序：主动打开(document) > 空Referer开关 > 白名单匹配。
func allowRequest(secFetchDest, referer string, p hotlinkPolicy) bool {
	// 用户主动打开/新标签访问，永远放行（防的是「嵌入盗用」，不是「点开」）
	if strings.EqualFold(strings.TrimSpace(secFetchDest), "document") {
		return true
	}
	referer = strings.TrimSpace(referer)
	if referer == "" {
		return p.allowEmptyReferer
	}
	if len(p.whitelist) == 0 {
		return true
	}
	u, err := url.Parse(referer)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "" {
		return false
	}
	for _, entry := range p.whitelist {
		if entry == "" {
			continue
		}
		if host == entry || strings.HasSuffix(host, "."+entry) {
			return true
		}
	}
	return false
}

func buildHotlinkPolicy(s *model.Settings) hotlinkPolicy {
	var list []string
	for _, part := range strings.Split(s.DirectRefererWhitelist, ",") {
		if v := strings.ToLower(strings.TrimSpace(part)); v != "" {
			list = append(list, v)
		}
	}
	return hotlinkPolicy{whitelist: list, allowEmptyReferer: s.DirectAllowEmptyReferer}
}

// DirectLink 处理 GET /f/:code —— 图床/视频床/文件床直连入口。
func (h *Handler) DirectLink(c *gin.Context) {
	code := c.Param("code")

	share, err := h.shareRepo.GetByCode(code)
	// 非直连分享不能走 /f/（避免绕过密码），一律按不存在处理
	if err != nil || !share.IsActive || !share.IsDirect {
		h.directRejectWith(c, http.StatusNotFound, "链接不存在", true)
		return
	}

	if share.ExpireAt != nil && share.ExpireAt.Before(time.Now()) {
		h.directRejectWith(c, http.StatusGone, "链接已失效", true)
		return
	}
	if share.MaxDownloads > 0 && share.DownloadCount >= share.MaxDownloads {
		h.directRejectWith(c, http.StatusForbidden, "链接已失效", true)
		return
	}

	settings, err := h.settingsRepo.Get()
	if err != nil {
		h.directRejectWith(c, http.StatusInternalServerError, "服务异常", false)
		return
	}

	// 限速（每 IP 每分钟）
	if !h.directLimiter.allow(c.ClientIP(), settings.DirectRateLimit) {
		h.directRejectWith(c, http.StatusTooManyRequests, "请求过于频繁", settings.DirectRejectPlaceholder)
		return
	}

	// 防盗链
	if !allowRequest(c.GetHeader("Sec-Fetch-Dest"), c.GetHeader("Referer"), buildHotlinkPolicy(settings)) {
		h.directRejectWith(c, http.StatusForbidden, "仅限授权站点引用", settings.DirectRejectPlaceholder)
		return
	}

	// 取直链：缓存优先，miss 才打上游并计一次引用
	downloadURL, hit := h.directCache.Get(share.TargetID)
	if !hit {
		downloadURL, err = h.wopanClient.GetDownloadURL(share.TargetID)
		if err != nil {
			h.directRejectWith(c, http.StatusBadGateway, "获取直链失败", true)
			return
		}
		h.directCache.Set(share.TargetID, downloadURL)

		h.shareRepo.IncrementDownloadCount(share.ID)
		h.accessLogRepo.Create(&model.AccessLog{
			ShareID:   share.ID,
			Action:    "download",
			IPAddress: c.ClientIP(),
			UserAgent: c.GetHeader("User-Agent"),
			Referer:   c.GetHeader("Referer"),
		})
	}

	c.Redirect(http.StatusFound, downloadURL)
}

// directRejectWith 拒绝一次直连请求：usePlaceholder 时返回占位图，否则返回 JSON。
func (h *Handler) directRejectWith(c *gin.Context, status int, msg string, usePlaceholder bool) {
	if usePlaceholder {
		c.Header("Cache-Control", "no-store")
		c.Data(status, "image/svg+xml", placeholderSVG(msg))
		return
	}
	c.JSON(status, model.APIResponse{Code: status, Message: msg})
}

// placeholderSVG 返回一张内联占位图，避免盗链/失效时页面裂图看不出原因。
func placeholderSVG(text string) []byte {
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="320" height="180" viewBox="0 0 320 180">` +
		`<rect width="320" height="180" fill="#1a1a1a"/>` +
		`<text x="160" y="96" fill="#888" font-family="monospace" font-size="16" text-anchor="middle">` +
		svgEscape(text) + `</text></svg>`
	return []byte(svg)
}

func svgEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// directRateLimiter 每 IP 固定窗口计数器（每分钟重置）。
// ponytail: 固定窗口 + 全局锁，自用规模足够；需要平滑再换令牌桶。
type directRateLimiter struct {
	mu     sync.Mutex
	window int64 // 当前窗口（Unix 分钟）
	counts map[string]int
}

func newDirectRateLimiter() *directRateLimiter {
	return &directRateLimiter{counts: make(map[string]int)}
}

// allow 返回是否放行；limit<=0 表示不限速。
func (l *directRateLimiter) allow(ip string, limit int) bool {
	if limit <= 0 {
		return true
	}
	minute := time.Now().Unix() / 60

	l.mu.Lock()
	defer l.mu.Unlock()
	if minute != l.window {
		l.window = minute
		l.counts = make(map[string]int)
	}
	if l.counts[ip] >= limit {
		return false
	}
	l.counts[ip]++
	return true
}
