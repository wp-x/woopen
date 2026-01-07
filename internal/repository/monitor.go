package repository

import (
	"database/sql"
	"time"
	"woopen/internal/model"
)

// MonitorRepository 监控状态仓库
type MonitorRepository struct {
	db *sql.DB
}

// NewMonitorRepository 创建监控状态仓库
func NewMonitorRepository(db *sql.DB) *MonitorRepository {
	return &MonitorRepository{db: db}
}

// GetStatus 获取监控状态
func (r *MonitorRepository) GetStatus() (*model.MonitorStatus, error) {
	var status model.MonitorStatus
	var lastCheckAt sql.NullTime
	var lastError sql.NullString
	err := r.db.QueryRow(`
		SELECT id, last_check_at, token_valid, last_error, consecutive_failures
		FROM monitor_status WHERE id = 1
	`).Scan(
		&status.ID,
		&lastCheckAt,
		&status.TokenValid,
		&lastError,
		&status.ConsecutiveFailures,
	)
	if err != nil {
		return nil, err
	}
	if lastCheckAt.Valid {
		status.LastCheckAt = lastCheckAt.Time
	}
	if lastError.Valid {
		status.LastError = lastError.String
	}
	return &status, nil
}

// UpdateStatus 更新监控状态
func (r *MonitorRepository) UpdateStatus(status *model.MonitorStatus) error {
	_, err := r.db.Exec(`
		UPDATE monitor_status SET
			last_check_at = ?,
			token_valid = ?,
			last_error = ?,
			consecutive_failures = ?
		WHERE id = 1
	`,
		status.LastCheckAt,
		status.TokenValid,
		status.LastError,
		status.ConsecutiveFailures,
	)
	return err
}

// NotificationLogRepository 通知记录仓库
type NotificationLogRepository struct {
	db *sql.DB
}

// NewNotificationLogRepository 创建通知记录仓库
func NewNotificationLogRepository(db *sql.DB) *NotificationLogRepository {
	return &NotificationLogRepository{db: db}
}

// Create 创建通知记录
func (r *NotificationLogRepository) Create(log *model.NotificationLog) error {
	result, err := r.db.Exec(`
		INSERT INTO notification_logs (event_type, message, status, error_msg, created_at)
		VALUES (?, ?, ?, ?, ?)
	`, log.EventType, log.Message, log.Status, log.ErrorMsg, time.Now())
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	log.ID = id
	return nil
}

// List 获取通知记录列表
func (r *NotificationLogRepository) List(page, pageSize int) ([]*model.NotificationLog, int64, error) {
	var total int64
	err := r.db.QueryRow(`SELECT COUNT(*) FROM notification_logs`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(`
		SELECT id, event_type, message, status, error_msg, created_at
		FROM notification_logs
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	logs := make([]*model.NotificationLog, 0)
	for rows.Next() {
		var log model.NotificationLog
		if err := rows.Scan(
			&log.ID,
			&log.EventType,
			&log.Message,
			&log.Status,
			&log.ErrorMsg,
			&log.CreatedAt,
		); err != nil {
			continue
		}
		logs = append(logs, &log)
	}

	return logs, total, nil
}

// CleanOldLogs 清理旧日志（保留最近N条）
func (r *NotificationLogRepository) CleanOldLogs(keepCount int) error {
	_, err := r.db.Exec(`
		DELETE FROM notification_logs
		WHERE id NOT IN (
			SELECT id FROM notification_logs ORDER BY created_at DESC LIMIT ?
		)
	`, keepCount)
	return err
}
