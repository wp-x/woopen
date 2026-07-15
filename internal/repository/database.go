package repository

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Database 数据库封装
type Database struct {
	db *sql.DB
}

// NewDatabase 创建数据库连接
func NewDatabase(dbPath string) (*Database, error) {
	// 确保数据目录存在
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}

	database := &Database{db: db}
	if err := database.migrate(); err != nil {
		return nil, fmt.Errorf("数据库迁移失败: %w", err)
	}

	return database, nil
}

// migrate 执行数据库迁移
func (d *Database) migrate() error {
	migrations := []string{
		// 设置表
		`CREATE TABLE IF NOT EXISTS settings (
			id INTEGER PRIMARY KEY,
			refresh_token TEXT NOT NULL DEFAULT '',
			access_token TEXT DEFAULT '',
			root_folder_id TEXT DEFAULT '',
			family_id TEXT DEFAULT '',
			site_title TEXT DEFAULT 'WoOpen',
			site_logo TEXT DEFAULT '',
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		// 确保settings表有一条默认记录
		`INSERT OR IGNORE INTO settings (id, refresh_token) VALUES (1, '')`,

		// 站点自定义字段迁移
		`ALTER TABLE settings ADD COLUMN login_title TEXT DEFAULT 'WoOpen_Auth_v10.exe'`,
		`ALTER TABLE settings ADD COLUMN login_avatar TEXT DEFAULT '💀'`,
		`ALTER TABLE settings ADD COLUMN login_role_tag TEXT DEFAULT 'ROLE: ADMIN'`,
		`ALTER TABLE settings ADD COLUMN login_level_tag TEXT DEFAULT 'LEVEL: 99'`,
		`ALTER TABLE settings ADD COLUMN login_system_name TEXT DEFAULT 'WOOPEN CLOUD SYSTEM'`,
		`ALTER TABLE settings ADD COLUMN share_footer TEXT DEFAULT 'Powered by WOOPEN_OS // BRUTAL_EDITION'`,

		// 分享表
		`CREATE TABLE IF NOT EXISTS shares (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			share_code TEXT UNIQUE NOT NULL,
			target_type TEXT NOT NULL CHECK(target_type IN ('file', 'folder')),
			target_id TEXT NOT NULL,
			target_path TEXT NOT NULL,
			target_name TEXT NOT NULL,
			password TEXT DEFAULT '',
			expire_at DATETIME,
			description TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			is_active BOOLEAN DEFAULT 1,
			view_count INTEGER DEFAULT 0,
			download_count INTEGER DEFAULT 0
		)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_code ON shares(share_code)`,
		`CREATE INDEX IF NOT EXISTS idx_shares_active ON shares(is_active)`,

		// 访问日志表
		`CREATE TABLE IF NOT EXISTS access_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			share_id INTEGER NOT NULL,
			action TEXT NOT NULL CHECK(action IN ('view', 'download')),
			ip_address TEXT DEFAULT '',
			user_agent TEXT DEFAULT '',
			referer TEXT DEFAULT '',
			accessed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (share_id) REFERENCES shares(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_share ON access_logs(share_id)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_time ON access_logs(accessed_at)`,

		// 监控配置字段迁移
		`ALTER TABLE settings ADD COLUMN monitor_enabled BOOLEAN DEFAULT 0`,
		`ALTER TABLE settings ADD COLUMN monitor_interval INTEGER DEFAULT 300`,
		// 多渠道通知配置
		`ALTER TABLE settings ADD COLUMN notify_enabled BOOLEAN DEFAULT 0`,
		`ALTER TABLE settings ADD COLUMN default_notify_channel TEXT DEFAULT 'bark'`,
		`ALTER TABLE settings ADD COLUMN bark_url TEXT DEFAULT ''`,
		`ALTER TABLE settings ADD COLUMN serverchan_key TEXT DEFAULT ''`,
		`ALTER TABLE settings ADD COLUMN telegram_bot_token TEXT DEFAULT ''`,
		`ALTER TABLE settings ADD COLUMN telegram_chat_id TEXT DEFAULT ''`,
		`ALTER TABLE settings ADD COLUMN pushplus_token TEXT DEFAULT ''`,
		`ALTER TABLE settings ADD COLUMN dingtalk_webhook TEXT DEFAULT ''`,
		`ALTER TABLE settings ADD COLUMN wecom_webhook TEXT DEFAULT ''`,

		// 通知记录表
		`CREATE TABLE IF NOT EXISTS notification_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type TEXT NOT NULL,
			message TEXT NOT NULL,
			status TEXT NOT NULL CHECK(status IN ('success', 'failed')),
			error_msg TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_notification_time ON notification_logs(created_at)`,

		// 监控状态表
		`CREATE TABLE IF NOT EXISTS monitor_status (
			id INTEGER PRIMARY KEY,
			last_check_at DATETIME,
			token_valid BOOLEAN DEFAULT 1,
			last_error TEXT DEFAULT '',
			consecutive_failures INTEGER DEFAULT 0
		)`,
		`INSERT OR IGNORE INTO monitor_status (id) VALUES (1)`,

		// 下载次数限制字段
		`ALTER TABLE shares ADD COLUMN max_downloads INTEGER DEFAULT 0`,

		// 直连（图床/外链）字段
		`ALTER TABLE shares ADD COLUMN is_direct BOOLEAN DEFAULT 0`,
		`ALTER TABLE settings ADD COLUMN direct_referer_whitelist TEXT DEFAULT ''`,
		`ALTER TABLE settings ADD COLUMN direct_allow_empty_referer BOOLEAN DEFAULT 1`,
		`ALTER TABLE settings ADD COLUMN direct_rate_limit INTEGER DEFAULT 60`,
		`ALTER TABLE settings ADD COLUMN direct_reject_placeholder BOOLEAN DEFAULT 1`,
	}

	for _, migration := range migrations {
		if _, err := d.db.Exec(migration); err != nil {
			// 忽略 ALTER TABLE 的 duplicate column 错误（幂等处理）
			if strings.Contains(migration, "ALTER TABLE") && strings.Contains(err.Error(), "duplicate column") {
				continue
			}
			return fmt.Errorf("执行迁移失败: %s, 错误: %w", migration[:50], err)
		}
	}

	return nil
}

// Close 关闭数据库连接
func (d *Database) Close() error {
	return d.db.Close()
}

// DB 获取底层数据库连接
func (d *Database) DB() *sql.DB {
	return d.db
}
