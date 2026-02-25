package middleware

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// WebDAVAuthMiddleware authenticates WebDAV requests.
//
// Rationale:
// - Most WebDAV clients use Basic auth, not Bearer JWT.
// - Keep Bearer JWT support for power users / scripted access.
func WebDAVAuthMiddleware(adminPassword string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			challengeBasic(c)
			return
		}

		// Bearer JWT (same as API)
		if strings.HasPrefix(authHeader, "Bearer ") {
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !ValidateToken(parts[1]) {
				challengeBasic(c)
				return
			}
			c.Next()
			return
		}

		// Basic auth (common for WebDAV mounts)
		if strings.HasPrefix(authHeader, "Basic ") {
			b64 := strings.TrimSpace(strings.TrimPrefix(authHeader, "Basic "))
			raw, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				challengeBasic(c)
				return
			}
			// raw is "username:password"
			user, pass, ok := strings.Cut(string(raw), ":")
			if !ok || user != "admin" || pass != adminPassword {
				challengeBasic(c)
				return
			}
			c.Next()
			return
		}

		challengeBasic(c)
	}
}

func challengeBasic(c *gin.Context) {
	// WebDAV clients expect a Basic challenge to prompt for credentials.
	c.Header("WWW-Authenticate", `Basic realm="WoOpen WebDAV"`)
	c.JSON(http.StatusUnauthorized, gin.H{
		"code":    401,
		"message": "未授权访问",
	})
	c.Abort()
}

