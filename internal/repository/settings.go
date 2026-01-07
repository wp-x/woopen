package repository

import (
	"database/sql"
	"time"
	"woopen/internal/model"
)

// SettingsRepository 设置仓库
type SettingsRepository struct {
	db *sql.DB
}

// NewSettingsRepository 创建设置仓库
func NewSettingsRepository(db *sql.DB) *SettingsRepository {
	return &SettingsRepository{db: db}
}

// Get 获取设置
func (r *SettingsRepository) Get() (*model.Settings, error) {
	var settings model.Settings
	var accessToken, loginTitle, loginAvatar, loginRoleTag, loginLevelTag, loginSystemName, shareFooter sql.NullString
	var monitorEnabled, notifyEnabled sql.NullBool
	var monitorInterval sql.NullInt64
	var barkURL, serverchanKey, telegramBotToken, telegramChatID, pushplusToken, dingtalkWebhook, wecomWebhook sql.NullString
	var defaultNotifyChannel sql.NullString
	err := r.db.QueryRow(`
		SELECT id, refresh_token, COALESCE(access_token, '') as access_token, root_folder_id, site_title, site_logo,
			   COALESCE(login_title, 'WoOpen_Auth_v10.exe') as login_title,
			   COALESCE(login_avatar, '💀') as login_avatar,
			   COALESCE(login_role_tag, 'ROLE: ADMIN') as login_role_tag,
			   COALESCE(login_level_tag, 'LEVEL: 99') as login_level_tag,
			   COALESCE(login_system_name, 'WOOPEN CLOUD SYSTEM') as login_system_name,
			   COALESCE(share_footer, 'Powered by WOOPEN_OS // BRUTAL_EDITION') as share_footer,
			   COALESCE(monitor_enabled, 0) as monitor_enabled,
			   COALESCE(monitor_interval, 300) as monitor_interval,
			   COALESCE(notify_enabled, 0) as notify_enabled,
			   COALESCE(default_notify_channel, 'bark') as default_notify_channel,
			   COALESCE(bark_url, '') as bark_url,
			   COALESCE(serverchan_key, '') as serverchan_key,
			   COALESCE(telegram_bot_token, '') as telegram_bot_token,
			   COALESCE(telegram_chat_id, '') as telegram_chat_id,
			   COALESCE(pushplus_token, '') as pushplus_token,
			   COALESCE(dingtalk_webhook, '') as dingtalk_webhook,
			   COALESCE(wecom_webhook, '') as wecom_webhook
		FROM settings WHERE id = 1
	`).Scan(
		&settings.ID,
		&settings.RefreshToken,
		&accessToken,
		&settings.RootFolderID,
		&settings.SiteTitle,
		&settings.SiteLogo,
		&loginTitle,
		&loginAvatar,
		&loginRoleTag,
		&loginLevelTag,
		&loginSystemName,
		&shareFooter,
		&monitorEnabled,
		&monitorInterval,
		&notifyEnabled,
		&defaultNotifyChannel,
		&barkURL,
		&serverchanKey,
		&telegramBotToken,
		&telegramChatID,
		&pushplusToken,
		&dingtalkWebhook,
		&wecomWebhook,
	)
	if err != nil {
		return nil, err
	}
	if accessToken.Valid {
		settings.AccessToken = accessToken.String
	}
	if loginTitle.Valid {
		settings.LoginTitle = loginTitle.String
	}
	if loginAvatar.Valid {
		settings.LoginAvatar = loginAvatar.String
	}
	if loginRoleTag.Valid {
		settings.LoginRoleTag = loginRoleTag.String
	}
	if loginLevelTag.Valid {
		settings.LoginLevelTag = loginLevelTag.String
	}
	if loginSystemName.Valid {
		settings.LoginSystemName = loginSystemName.String
	}
	if shareFooter.Valid {
		settings.ShareFooter = shareFooter.String
	}
	if monitorEnabled.Valid {
		settings.MonitorEnabled = monitorEnabled.Bool
	}
	if monitorInterval.Valid {
		settings.MonitorInterval = int(monitorInterval.Int64)
	} else {
		settings.MonitorInterval = 300 // 默认5分钟
	}
	if notifyEnabled.Valid {
		settings.NotifyEnabled = notifyEnabled.Bool
	}
	if defaultNotifyChannel.Valid && defaultNotifyChannel.String != "" {
		settings.DefaultNotifyChannel = defaultNotifyChannel.String
	} else {
		settings.DefaultNotifyChannel = "bark" // 默认 bark
	}
	if barkURL.Valid {
		settings.BarkURL = barkURL.String
	}
	if serverchanKey.Valid {
		settings.ServerchanKey = serverchanKey.String
	}
	if telegramBotToken.Valid {
		settings.TelegramBotToken = telegramBotToken.String
	}
	if telegramChatID.Valid {
		settings.TelegramChatID = telegramChatID.String
	}
	if pushplusToken.Valid {
		settings.PushplusToken = pushplusToken.String
	}
	if dingtalkWebhook.Valid {
		settings.DingtalkWebhook = dingtalkWebhook.String
	}
	if wecomWebhook.Valid {
		settings.WecomWebhook = wecomWebhook.String
	}
	return &settings, nil
}

// Update 更新设置
func (r *SettingsRepository) Update(settings *model.Settings) error {
	_, err := r.db.Exec(`
		UPDATE settings SET
			refresh_token = ?,
			access_token = ?,
			root_folder_id = ?,
			site_title = ?,
			site_logo = ?,
			login_title = ?,
			login_avatar = ?,
			login_role_tag = ?,
			login_level_tag = ?,
			login_system_name = ?,
			share_footer = ?,
			monitor_enabled = ?,
			monitor_interval = ?,
			notify_enabled = ?,
			default_notify_channel = ?,
			bark_url = ?,
			serverchan_key = ?,
			telegram_bot_token = ?,
			telegram_chat_id = ?,
			pushplus_token = ?,
			dingtalk_webhook = ?,
			wecom_webhook = ?,
			updated_at = ?
		WHERE id = 1
	`,
		settings.RefreshToken,
		settings.AccessToken,
		settings.RootFolderID,
		settings.SiteTitle,
		settings.SiteLogo,
		settings.LoginTitle,
		settings.LoginAvatar,
		settings.LoginRoleTag,
		settings.LoginLevelTag,
		settings.LoginSystemName,
		settings.ShareFooter,
		settings.MonitorEnabled,
		settings.MonitorInterval,
		settings.NotifyEnabled,
		settings.DefaultNotifyChannel,
		settings.BarkURL,
		settings.ServerchanKey,
		settings.TelegramBotToken,
		settings.TelegramChatID,
		settings.PushplusToken,
		settings.DingtalkWebhook,
		settings.WecomWebhook,
		time.Now(),
	)
	return err
}

// UpdateToken 更新Token
func (r *SettingsRepository) UpdateToken(refreshToken, accessToken string) error {
	_, err := r.db.Exec(`
		UPDATE settings SET 
			refresh_token = ?,
			access_token = ?,
			updated_at = ?
		WHERE id = 1
	`, refreshToken, accessToken, time.Now())
	return err
}

// GetAccessToken 获取AccessToken
func (r *SettingsRepository) GetAccessToken() (string, error) {
	var accessToken sql.NullString
	err := r.db.QueryRow(`SELECT access_token FROM settings WHERE id = 1`).Scan(&accessToken)
	if accessToken.Valid {
		return accessToken.String, err
	}
	return "", err
}
