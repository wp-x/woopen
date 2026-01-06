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
	ShareCode   string `json:"share_code"`
	TargetType  string `json:"target_type" binding:"required,oneof=file folder"`
	TargetID    string `json:"target_id" binding:"required"`
	TargetPath  string `json:"target_path"`
	TargetName  string `json:"target_name" binding:"required"`
	Password    string `json:"password"`
	ExpireAt    string `json:"expire_at"` // ISO8601 格式
	Description string `json:"description"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Password string `json:"password" binding:"required"`
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
