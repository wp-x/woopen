package model

import (
	"time"
)

// Token 令牌存储
type Token struct {
	ID           int64     `json:"id"`
	RefreshToken string    `json:"refresh_token"`
	AccessToken  string    `json:"access_token"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Settings 系统设置
type Settings struct {
	ID           int64  `json:"id"`
	RefreshToken string `json:"refresh_token"`
	AccessToken  string `json:"access_token"`   // Access Token（与 Refresh Token 配合使用）
	RootFolderID string `json:"root_folder_id"` // 根文件夹ID，空为根目录
	SiteTitle    string `json:"site_title"`
	SiteLogo     string `json:"site_logo"`
	// 登录页自定义
	LoginTitle      string `json:"login_title"`       // 登录窗口标题栏文字
	LoginAvatar     string `json:"login_avatar"`      // 登录页头像（emoji或图片URL）
	LoginRoleTag    string `json:"login_role_tag"`    // 角色标签
	LoginLevelTag   string `json:"login_level_tag"`   // 等级标签
	LoginSystemName string `json:"login_system_name"` // 系统名称
	// 分享页自定义
	ShareFooter string `json:"share_footer"` // 分享页底部署名
	// 监控配置
	MonitorEnabled  bool `json:"monitor_enabled"`  // 是否启用监控
	MonitorInterval int  `json:"monitor_interval"` // 监控间隔（秒）
	// 多渠道通知配置
	NotifyEnabled        bool   `json:"notify_enabled"`         // 是否启用通知
	DefaultNotifyChannel string `json:"default_notify_channel"` // 默认通知渠道
	BarkURL              string `json:"bark_url"`               // Bark 推送地址
	ServerchanKey        string `json:"serverchan_key"`         // Server 酱 SendKey
	TelegramBotToken     string `json:"telegram_bot_token"`     // Telegram Bot Token
	TelegramChatID       string `json:"telegram_chat_id"`       // Telegram Chat ID
	PushplusToken        string `json:"pushplus_token"`         // PushPlus Token
	DingtalkWebhook      string `json:"dingtalk_webhook"`       // 钉钉 Webhook
	WecomWebhook         string `json:"wecom_webhook"`          // 企业微信群机器人 Webhook
	WecomAppConfig       string `json:"wecom_app_config"`       // 企业微信应用消息 corpid,secret,touser,agentid
	FeishuWebhook        string `json:"feishu_webhook"`         // 飞书机器人 Webhook
	WebhookURL           string `json:"webhook_url"`            // 通用 Webhook（POST JSON）
	PushdeerKey          string `json:"pushdeer_key"`           // PushDeer PushKey
	// 直连（图床/外链）配置
	DirectRefererWhitelist  string `json:"direct_referer_whitelist"`   // 逗号分隔域名，空=不限制
	DirectAllowEmptyReferer bool   `json:"direct_allow_empty_referer"` // 允许空 Referer（默认开）
	DirectRateLimit         int    `json:"direct_rate_limit"`          // 每 IP 每分钟上限，0=不限
	DirectRejectPlaceholder bool   `json:"direct_reject_placeholder"`  // 拒绝时返回占位图
}

// MonitorStatus 监控状态
type MonitorStatus struct {
	ID                  int64     `json:"id"`
	LastCheckAt         time.Time `json:"last_check_at"`
	TokenValid          bool      `json:"token_valid"`
	LastError           string    `json:"last_error"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
}

// NotificationLog 通知记录
type NotificationLog struct {
	ID        int64     `json:"id"`
	EventType string    `json:"event_type"` // token_invalid, token_refresh, monitor_error
	Message   string    `json:"message"`
	Status    string    `json:"status"` // success, failed
	ErrorMsg  string    `json:"error_msg"`
	CreatedAt time.Time `json:"created_at"`
}

// Share 分享链接
type Share struct {
	ID            int64      `json:"id"`
	ShareCode     string     `json:"share_code"`  // 短链接标识
	TargetType    string     `json:"target_type"` // file 或 folder
	TargetID      string     `json:"target_id"`   // 云盘文件/文件夹ID
	TargetPath    string     `json:"target_path"` // 路径（用于显示）
	TargetName    string     `json:"target_name"` // 名称
	Password      string     `json:"password"`    // 访问密码
	ExpireAt      *time.Time `json:"expire_at"`   // 过期时间
	Description   string     `json:"description"` // 备注
	CreatedAt     time.Time  `json:"created_at"`
	IsActive      bool       `json:"is_active"`
	ViewCount     int64      `json:"view_count"`     // 访问次数
	DownloadCount int64      `json:"download_count"` // 下载次数
	MaxDownloads  int64      `json:"max_downloads"`  // 最大下载次数，0表示不限制
	IsDirect      bool       `json:"is_direct"`      // 直连模式（图床/外链），不支持密码
}

// AccessLog 访问日志
type AccessLog struct {
	ID         int64     `json:"id"`
	ShareID    int64     `json:"share_id"`
	Action     string    `json:"action"` // view 或 download
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Referer    string    `json:"referer"`
	AccessedAt time.Time `json:"accessed_at"`
}

// FileInfo 文件信息（来自云盘API）
type FileInfo struct {
	ID       string    `json:"id"`
	FID      string    `json:"fid"`
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	IsDir    bool      `json:"is_dir"`
	ModTime  time.Time `json:"mod_time"`
	ParentID string    `json:"parent_id"`
	Path     string    `json:"path"`
	MimeType string    `json:"mime_type"`
}

// CreateShareRequest 创建分享请求
type CreateShareRequest struct {
	ShareCode    string `json:"share_code"`
	TargetType   string `json:"target_type" binding:"required,oneof=file folder"`
	TargetID     string `json:"target_id" binding:"required"`
	TargetPath   string `json:"target_path"`
	TargetName   string `json:"target_name" binding:"required"`
	Password     string `json:"password"`
	ExpireAt     string `json:"expire_at"` // ISO8601 格式
	Description  string `json:"description"`
	MaxDownloads int64  `json:"max_downloads"` // 最大下载次数，0表示不限制
	IsDirect     bool   `json:"is_direct"`     // 直连模式（图床/外链）
}

// LoginRequest 登录请求
type LoginRequest struct {
	Password string `json:"password" binding:"required"`
	Expire   string `json:"expire"` // 7d | 30d | forever，默认30d
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token string `json:"token"`
}

// APIResponse 通用API响应
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
