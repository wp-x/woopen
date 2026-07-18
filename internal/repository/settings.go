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
	var loginUsername sql.NullString
	var monitorEnabled, notifyEnabled sql.NullBool
	var monitorInterval sql.NullInt64
	var barkURL, serverchanKey, telegramBotToken, telegramChatID, pushplusToken, dingtalkWebhook, wecomWebhook sql.NullString
	var wecomAppConfig, feishuWebhook, webhookURL, pushdeerKey sql.NullString
	var defaultNotifyChannel sql.NullString
	var directRefererWhitelist sql.NullString
	var directAllowEmptyReferer, directRejectPlaceholder sql.NullBool
	var directRateLimit sql.NullInt64
	var webdavEnabled, webdavReadonly sql.NullBool
	var webdavUsername, webdavPassword sql.NullString
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
			   COALESCE(wecom_webhook, '') as wecom_webhook,
			   COALESCE(wecom_app_config, '') as wecom_app_config,
			   COALESCE(feishu_webhook, '') as feishu_webhook,
			   COALESCE(webhook_url, '') as webhook_url,
			   COALESCE(pushdeer_key, '') as pushdeer_key,
			   COALESCE(direct_referer_whitelist, '') as direct_referer_whitelist,
			   COALESCE(direct_allow_empty_referer, 1) as direct_allow_empty_referer,
			   COALESCE(direct_rate_limit, 60) as direct_rate_limit,
			   COALESCE(direct_reject_placeholder, 1) as direct_reject_placeholder,
			   COALESCE(webdav_enabled, 1) as webdav_enabled,
			   COALESCE(webdav_username, 'admin') as webdav_username,
			   COALESCE(webdav_password, '') as webdav_password,
			   COALESCE(webdav_readonly, 0) as webdav_readonly,
			   COALESCE(login_username, 'Administrator') as login_username
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
		&wecomAppConfig,
		&feishuWebhook,
		&webhookURL,
		&pushdeerKey,
		&directRefererWhitelist,
		&directAllowEmptyReferer,
		&directRateLimit,
		&directRejectPlaceholder,
		&webdavEnabled,
		&webdavUsername,
		&webdavPassword,
		&webdavReadonly,
		&loginUsername,
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
	if wecomAppConfig.Valid {
		settings.WecomAppConfig = wecomAppConfig.String
	}
	if feishuWebhook.Valid {
		settings.FeishuWebhook = feishuWebhook.String
	}
	if webhookURL.Valid {
		settings.WebhookURL = webhookURL.String
	}
	if pushdeerKey.Valid {
		settings.PushdeerKey = pushdeerKey.String
	}
	if directRefererWhitelist.Valid {
		settings.DirectRefererWhitelist = directRefererWhitelist.String
	}
	if directAllowEmptyReferer.Valid {
		settings.DirectAllowEmptyReferer = directAllowEmptyReferer.Bool
	} else {
		settings.DirectAllowEmptyReferer = true
	}
	if directRateLimit.Valid {
		settings.DirectRateLimit = int(directRateLimit.Int64)
	} else {
		settings.DirectRateLimit = 60
	}
	if directRejectPlaceholder.Valid {
		settings.DirectRejectPlaceholder = directRejectPlaceholder.Bool
	} else {
		settings.DirectRejectPlaceholder = true
	}
	if webdavEnabled.Valid {
		settings.WebdavEnabled = webdavEnabled.Bool
	} else {
		settings.WebdavEnabled = true
	}
	if webdavUsername.Valid && webdavUsername.String != "" {
		settings.WebdavUsername = webdavUsername.String
	} else {
		settings.WebdavUsername = "admin"
	}
	if webdavPassword.Valid {
		settings.WebdavPassword = webdavPassword.String
	}
	if webdavReadonly.Valid {
		settings.WebdavReadonly = webdavReadonly.Bool
	}
	if loginUsername.Valid && loginUsername.String != "" {
		settings.LoginUsername = loginUsername.String
	} else {
		settings.LoginUsername = "Administrator"
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
			login_username = ?,
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
			wecom_app_config = ?,
			feishu_webhook = ?,
			webhook_url = ?,
			pushdeer_key = ?,
			direct_referer_whitelist = ?,
			direct_allow_empty_referer = ?,
			direct_rate_limit = ?,
			direct_reject_placeholder = ?,
			webdav_enabled = ?,
			webdav_username = ?,
			webdav_password = ?,
			webdav_readonly = ?,
			updated_at = ?
		WHERE id = 1
	`,
		settings.RefreshToken,
		settings.AccessToken,
		settings.RootFolderID,
		settings.SiteTitle,
		settings.SiteLogo,
		settings.LoginUsername,
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
		settings.WecomAppConfig,
		settings.FeishuWebhook,
		settings.WebhookURL,
		settings.PushdeerKey,
		settings.DirectRefererWhitelist,
		settings.DirectAllowEmptyReferer,
		settings.DirectRateLimit,
		settings.DirectRejectPlaceholder,
		settings.WebdavEnabled,
		settings.WebdavUsername,
		settings.WebdavPassword,
		settings.WebdavReadonly,
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
