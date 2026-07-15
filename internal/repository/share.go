package repository

import (
	"database/sql"
	"time"
	"woopen/internal/model"
)

// ShareRepository 分享仓库
type ShareRepository struct {
	db *sql.DB
}

// NewShareRepository 创建分享仓库
func NewShareRepository(db *sql.DB) *ShareRepository {
	return &ShareRepository{db: db}
}

// Create 创建分享
func (r *ShareRepository) Create(share *model.Share) error {
	result, err := r.db.Exec(`
		INSERT INTO shares (share_code, target_type, target_id, target_path, target_name, password, expire_at, description, is_active, max_downloads, is_direct)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		share.ShareCode,
		share.TargetType,
		share.TargetID,
		share.TargetPath,
		share.TargetName,
		share.Password,
		share.ExpireAt,
		share.Description,
		share.IsActive,
		share.MaxDownloads,
		share.IsDirect,
	)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	share.ID = id
	return nil
}

// GetByCode 通过code获取分享
func (r *ShareRepository) GetByCode(code string) (*model.Share, error) {
	var share model.Share
	var expireAt sql.NullTime
	var maxDownloads sql.NullInt64
	err := r.db.QueryRow(`
		SELECT id, share_code, target_type, target_id, target_path, target_name,
			   password, expire_at, description, created_at, is_active, view_count, download_count, max_downloads, is_direct
		FROM shares WHERE share_code = ?
	`, code).Scan(
		&share.ID,
		&share.ShareCode,
		&share.TargetType,
		&share.TargetID,
		&share.TargetPath,
		&share.TargetName,
		&share.Password,
		&expireAt,
		&share.Description,
		&share.CreatedAt,
		&share.IsActive,
		&share.ViewCount,
		&share.DownloadCount,
		&maxDownloads,
		&share.IsDirect,
	)
	if err != nil {
		return nil, err
	}
	if expireAt.Valid {
		share.ExpireAt = &expireAt.Time
	}
	if maxDownloads.Valid {
		share.MaxDownloads = maxDownloads.Int64
	}
	return &share, nil
}

// GetByID 通过ID获取分享
func (r *ShareRepository) GetByID(id int64) (*model.Share, error) {
	var share model.Share
	var expireAt sql.NullTime
	var maxDownloads sql.NullInt64
	err := r.db.QueryRow(`
		SELECT id, share_code, target_type, target_id, target_path, target_name,
			   password, expire_at, description, created_at, is_active, view_count, download_count, max_downloads, is_direct
		FROM shares WHERE id = ?
	`, id).Scan(
		&share.ID,
		&share.ShareCode,
		&share.TargetType,
		&share.TargetID,
		&share.TargetPath,
		&share.TargetName,
		&share.Password,
		&expireAt,
		&share.Description,
		&share.CreatedAt,
		&share.IsActive,
		&share.ViewCount,
		&share.DownloadCount,
		&maxDownloads,
		&share.IsDirect,
	)
	if err != nil {
		return nil, err
	}
	if expireAt.Valid {
		share.ExpireAt = &expireAt.Time
	}
	if maxDownloads.Valid {
		share.MaxDownloads = maxDownloads.Int64
	}
	return &share, nil
}

// List 获取分享列表
func (r *ShareRepository) List(page, pageSize int) ([]*model.Share, int64, error) {
	// 获取总数
	var total int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM shares`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(`
		SELECT id, share_code, target_type, target_id, target_path, target_name,
			   password, expire_at, description, created_at, is_active, view_count, download_count, max_downloads, is_direct
		FROM shares ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shares []*model.Share
	for rows.Next() {
		var share model.Share
		var expireAt sql.NullTime
		var maxDownloads sql.NullInt64
		err := rows.Scan(
			&share.ID,
			&share.ShareCode,
			&share.TargetType,
			&share.TargetID,
			&share.TargetPath,
			&share.TargetName,
			&share.Password,
			&expireAt,
			&share.Description,
			&share.CreatedAt,
			&share.IsActive,
			&share.ViewCount,
			&share.DownloadCount,
			&maxDownloads,
			&share.IsDirect,
		)
		if err != nil {
			return nil, 0, err
		}
		if expireAt.Valid {
			share.ExpireAt = &expireAt.Time
		}
		if maxDownloads.Valid {
			share.MaxDownloads = maxDownloads.Int64
		}
		shares = append(shares, &share)
	}
	return shares, total, nil
}

// Update 更新分享
func (r *ShareRepository) Update(share *model.Share) error {
	_, err := r.db.Exec(`
		UPDATE shares SET
			share_code = ?,
			password = ?,
			expire_at = ?,
			description = ?,
			is_active = ?,
			max_downloads = ?,
			is_direct = ?
		WHERE id = ?
	`,
		share.ShareCode,
		share.Password,
		share.ExpireAt,
		share.Description,
		share.IsActive,
		share.MaxDownloads,
		share.IsDirect,
		share.ID,
	)
	return err
}

// Delete 删除分享
func (r *ShareRepository) Delete(id int64) error {
	_, err := r.db.Exec(`DELETE FROM shares WHERE id = ?`, id)
	return err
}

// IncrementViewCount 增加访问计数
func (r *ShareRepository) IncrementViewCount(id int64) error {
	_, err := r.db.Exec(`UPDATE shares SET view_count = view_count + 1 WHERE id = ?`, id)
	return err
}

// IncrementDownloadCount 增加下载计数
func (r *ShareRepository) IncrementDownloadCount(id int64) error {
	_, err := r.db.Exec(`UPDATE shares SET download_count = download_count + 1 WHERE id = ?`, id)
	return err
}

// CodeExists 检查share_code是否存在
func (r *ShareRepository) CodeExists(code string) (bool, error) {
	var count int
	err := r.db.QueryRow(`SELECT COUNT(*) FROM shares WHERE share_code = ?`, code).Scan(&count)
	return count > 0, err
}

// AccessLogRepository 访问日志仓库
type AccessLogRepository struct {
	db *sql.DB
}

// NewAccessLogRepository 创建访问日志仓库
func NewAccessLogRepository(db *sql.DB) *AccessLogRepository {
	return &AccessLogRepository{db: db}
}

// Create 创建访问日志
func (r *AccessLogRepository) Create(log *model.AccessLog) error {
	_, err := r.db.Exec(`
		INSERT INTO access_logs (share_id, action, ip_address, user_agent, referer)
		VALUES (?, ?, ?, ?, ?)
	`, log.ShareID, log.Action, log.IPAddress, log.UserAgent, log.Referer)
	return err
}

// GetStats 获取统计数据
func (r *AccessLogRepository) GetStats() (totalViews, totalDownloads, todayViews int64, err error) {
	err = r.db.QueryRow(`SELECT COUNT(*) FROM access_logs WHERE action = 'view'`).Scan(&totalViews)
	if err != nil {
		return
	}
	err = r.db.QueryRow(`SELECT COUNT(*) FROM access_logs WHERE action = 'download'`).Scan(&totalDownloads)
	if err != nil {
		return
	}
	today := time.Now().Format("2006-01-02")
	err = r.db.QueryRow(`
		SELECT COUNT(*) FROM access_logs 
		WHERE action = 'view' AND DATE(accessed_at) = ?
	`, today).Scan(&todayViews)
	return
}

// GetTrend 获取最近7天趋势
func (r *AccessLogRepository) GetTrend() ([]map[string]interface{}, error) {
	rows, err := r.db.Query(`
		SELECT DATE(accessed_at) as date, action, COUNT(*) as count
		FROM access_logs 
		WHERE accessed_at >= DATE('now', '-7 days')
		GROUP BY DATE(accessed_at), action
		ORDER BY date
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]map[string]interface{}, 0)
	for rows.Next() {
		var date, action string
		var count int64
		if err := rows.Scan(&date, &action, &count); err != nil {
			return nil, err
		}
		result = append(result, map[string]interface{}{
			"date":   date,
			"action": action,
			"count":  count,
		})
	}
	return result, nil
}
