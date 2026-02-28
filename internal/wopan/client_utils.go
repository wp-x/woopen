package wopan

import (
	"os"
	"strings"
)

func isLoginFailed(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "rsp_code: 8005") ||
		strings.Contains(msg, "rsp_code: 1001") ||
		strings.Contains(msg, "登录失败") ||
		strings.Contains(msg, "无效的令牌")
}

// TempFileUploader is used by WebDAV to spool data to disk without holding it in memory.
// It is intentionally minimal and internal to this package.
func TempFileUploader(prefix string) (*os.File, error) {
	return os.CreateTemp("", prefix)
}
