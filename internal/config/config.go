package config

import (
	"os"
	"path/filepath"
)

// Config 系统配置
type Config struct {
	Port          string // 服务端口
	DataDir       string // 数据目录
	AdminPassword string // 管理员密码
	JWTSecret     string // JWT密钥
	SiteURL       string // 站点URL
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Port:          getEnv("WOOPEN_PORT", "8080"),
		DataDir:       getEnv("WOOPEN_DATA_DIR", "./data"),
		AdminPassword: getEnv("WOOPEN_ADMIN_PASSWORD", "admin123"),
		JWTSecret:     getEnv("WOOPEN_JWT_SECRET", "woopen-secret-key-change-in-production"),
		SiteURL:       getEnv("WOOPEN_SITE_URL", "http://localhost:8080"),
	}
}

// DatabasePath 返回数据库文件路径
func (c *Config) DatabasePath() string {
	return filepath.Join(c.DataDir, "woopen.db")
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
