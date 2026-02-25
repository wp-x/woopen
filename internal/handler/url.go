package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func firstHeaderValue(v string) string {
	if v == "" {
		return ""
	}
	// Common reverse-proxy headers can contain multiple values: "https,http".
	if i := strings.IndexByte(v, ','); i >= 0 {
		v = v[:i]
	}
	return strings.TrimSpace(v)
}

// requestBaseURL builds a best-effort external base URL (scheme://host) for display/copy purposes.
// It prefers X-Forwarded-* headers (when behind reverse proxies), then falls back to request info.
func requestBaseURL(c *gin.Context) string {
	scheme := firstHeaderValue(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := firstHeaderValue(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		host = "localhost"
	}

	return scheme + "://" + host
}

