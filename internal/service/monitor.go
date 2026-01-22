package service

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
	"woopen/internal/model"
	"woopen/internal/repository"
	"woopen/internal/wopan"
)

// MonitorService 账号监控服务
type MonitorService struct {
	settingsRepo *repository.SettingsRepository
	monitorRepo  *repository.MonitorRepository
	wopanClient  *wopan.Client
	notifier     *Notifier

	mu           sync.RWMutex
	running      bool
	stopCh       chan struct{}
	interval     time.Duration
	lastNotified time.Time // 上次通知时间，避免重复通知
}

// NewMonitorService 创建监控服务
func NewMonitorService(
	settingsRepo *repository.SettingsRepository,
	monitorRepo *repository.MonitorRepository,
	wopanClient *wopan.Client,
	notifier *Notifier,
) *MonitorService {
	return &MonitorService{
		settingsRepo: settingsRepo,
		monitorRepo:  monitorRepo,
		wopanClient:  wopanClient,
		notifier:     notifier,
		interval:     5 * time.Minute, // 默认5分钟
	}
}

// Start 启动监控服务
func (m *MonitorService) Start() {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return
	}
	m.running = true
	m.stopCh = make(chan struct{})
	m.mu.Unlock()

	go m.run()
	log.Println("[Monitor] 账号监控服务已启动")
}

// Stop 停止监控服务
func (m *MonitorService) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	close(m.stopCh)
	m.running = false
	log.Println("[Monitor] 账号监控服务已停止")
}

// IsRunning 检查是否正在运行
func (m *MonitorService) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// SetInterval 设置检查间隔
func (m *MonitorService) SetInterval(seconds int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if seconds < 60 {
		seconds = 60 // 最小1分钟
	}
	m.interval = time.Duration(seconds) * time.Second
}

// SetWopanClient 更新 WoPan 客户端
func (m *MonitorService) SetWopanClient(client *wopan.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wopanClient = client
}

// run 监控主循环
func (m *MonitorService) run() {
	// 首次启动时立即检查一次
	m.check()

	for {
		m.mu.RLock()
		interval := m.interval
		m.mu.RUnlock()

		select {
		case <-m.stopCh:
			return
		case <-time.After(interval):
			m.check()
		}
	}
}

// check 执行一次检查
func (m *MonitorService) check() {
	m.mu.RLock()
	client := m.wopanClient
	m.mu.RUnlock()

	if client == nil || !client.IsInitialized() {
		log.Println("[Monitor] 云盘客户端未初始化，跳过检查")
		return
	}

	settings, err := m.settingsRepo.Get()
	if err != nil {
		log.Printf("[Monitor] 获取设置失败: %v\n", err)
		return
	}

	if !settings.MonitorEnabled {
		return
	}

	status, err := m.monitorRepo.GetStatus()
	if err != nil {
		log.Printf("[Monitor] 获取监控状态失败: %v\n", err)
		status = &model.MonitorStatus{}
	}

	// 使用轻量级检测方式，带重试机制
	checkErr := m.validateToken(client)

	now := time.Now()
	status.LastCheckAt = now

	// 构建通知配置
	notifyConfig := NotifyConfig{
		NotifyEnabled:    settings.NotifyEnabled,
		BarkURL:          settings.BarkURL,
		ServerchanKey:    settings.ServerchanKey,
		TelegramBotToken: settings.TelegramBotToken,
		TelegramChatID:   settings.TelegramChatID,
		PushplusToken:    settings.PushplusToken,
		DingtalkWebhook:  settings.DingtalkWebhook,
		WecomWebhook:     settings.WecomWebhook,
	}

	if checkErr != nil {
		// Token 可能失效
		status.TokenValid = false
		status.LastError = checkErr.Error()
		status.ConsecutiveFailures++

		log.Printf("[Monitor] Token 检查失败 (连续 %d 次): %v\n", status.ConsecutiveFailures, checkErr)

		// 只在连续失败3次以上才判定为真正失效，避免临时性错误误报
		// 并且间隔超过1小时才发送通知，避免重复通知
		if settings.NotifyEnabled && status.ConsecutiveFailures >= 3 && time.Since(m.lastNotified) > time.Hour {
			m.notifier.NotifyTokenInvalid(notifyConfig, checkErr.Error())
			m.lastNotified = now
		}
	} else {
		// Token 有效
		wasInvalid := !status.TokenValid || status.ConsecutiveFailures > 0

		status.TokenValid = true
		status.LastError = ""
		status.ConsecutiveFailures = 0

		// 如果之前是失效状态，现在恢复了，发送恢复通知
		if wasInvalid && settings.NotifyEnabled {
			m.notifier.NotifyTokenRefreshed(notifyConfig)
			m.lastNotified = now
		}

		log.Println("[Monitor] Token 检查通过")
	}

	// 更新状态
	if err := m.monitorRepo.UpdateStatus(status); err != nil {
		log.Printf("[Monitor] 更新监控状态失败: %v\n", err)
	}
}

// validateToken 验证 Token 有效性（带重试机制）
func (m *MonitorService) validateToken(client *wopan.Client) error {
	maxRetries := 2
	retryDelay := 2 * time.Second

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			log.Printf("[Monitor] Token 检查重试 %d/%d\n", i, maxRetries)
			time.Sleep(retryDelay)
		}

		// 使用轻量级的文件列表查询来验证 Token
		// 只获取1个文件，减少 API 负担
		_, err := client.ListFiles("0", 1, 1)

		if err == nil {
			// 检查成功
			return nil
		}

		// 判断是否为真正的 Token 失效错误
		if isTokenInvalidError(err) {
			// 确认是 Token 失效，不再重试
			return err
		}

		// 其他错误（如网络问题、临时限流等），继续重试
		log.Printf("[Monitor] 检查遇到临时错误: %v\n", err)
	}

	// 重试后仍然失败，但不是 Token 失效错误，返回最后一次错误
	_, lastErr := client.ListFiles("0", 1, 1)
	return lastErr
}

// isTokenInvalidError 判断是否为 Token 失效错误
func isTokenInvalidError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// 只有明确的 Token 失效错误码才判定为失效
	return strings.Contains(msg, "rsp_code: 8005") ||
		strings.Contains(msg, "rsp_code: 1001") ||
		strings.Contains(msg, "登录失败") ||
		strings.Contains(msg, "无效的令牌") ||
		strings.Contains(msg, "token已过期") ||
		strings.Contains(msg, "token invalid")
}

// CheckNow 立即执行一次检查
func (m *MonitorService) CheckNow() (*model.MonitorStatus, error) {
	m.check()
	return m.monitorRepo.GetStatus()
}

// GetStatus 获取当前监控状态
func (m *MonitorService) GetStatus() (*model.MonitorStatus, error) {
	return m.monitorRepo.GetStatus()
}

// TestNotify 测试通知（所有已配置渠道）
func (m *MonitorService) TestNotify() (map[string]string, error) {
	settings, err := m.settingsRepo.Get()
	if err != nil {
		return nil, err
	}

	config := NotifyConfig{
		NotifyEnabled:    true, // 测试时强制启用
		BarkURL:          settings.BarkURL,
		ServerchanKey:    settings.ServerchanKey,
		TelegramBotToken: settings.TelegramBotToken,
		TelegramChatID:   settings.TelegramChatID,
		PushplusToken:    settings.PushplusToken,
		DingtalkWebhook:  settings.DingtalkWebhook,
		WecomWebhook:     settings.WecomWebhook,
	}

	results := m.notifier.TestAll(config)
	if len(results) == 0 {
		return nil, fmt.Errorf("未配置任何通知渠道")
	}
	return results, nil
}

// Reload 重新加载配置
func (m *MonitorService) Reload() error {
	settings, err := m.settingsRepo.Get()
	if err != nil {
		return err
	}

	m.SetInterval(settings.MonitorInterval)

	if settings.MonitorEnabled && !m.IsRunning() {
		m.Start()
	} else if !settings.MonitorEnabled && m.IsRunning() {
		m.Stop()
	}

	return nil
}
