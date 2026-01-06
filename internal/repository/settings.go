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
	err := r.db.QueryRow(`
		SELECT id, refresh_token, COALESCE(access_token, '') as access_token, root_folder_id, site_title, site_logo,
			   COALESCE(login_title, 'WoOpen_Auth_v10.exe') as login_title,
			   COALESCE(login_avatar, '💀') as login_avatar,
			   COALESCE(login_role_tag, 'ROLE: ADMIN') as login_role_tag,
			   COALESCE(login_level_tag, 'LEVEL: 99') as login_level_tag,
			   COALESCE(login_system_name, 'WOOPEN CLOUD SYSTEM') as login_system_name,
			   COALESCE(share_footer, 'Powered by WOOPEN_OS // BRUTAL_EDITION') as share_footer
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
