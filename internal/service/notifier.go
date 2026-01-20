package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
	"woopen/internal/model"
	"woopen/internal/repository"
)

// Notifier 多渠道通知服务
type Notifier struct {
	notificationRepo *repository.NotificationLogRepository
	httpClient       *http.Client
}

// NewNotifier 创建通知服务
func NewNotifier(notificationRepo *repository.NotificationLogRepository) *Notifier {
	return &Notifier{
		notificationRepo: notificationRepo,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NotifyConfig 通知配置
type NotifyConfig struct {
	NotifyEnabled    bool
	BarkURL          string
	ServerchanKey    string
	TelegramBotToken string
	TelegramChatID   string
	PushplusToken    string
	DingtalkWebhook  string
	WecomWebhook     string
}

// SendAll 向所有已配置的渠道发送通知
func (n *Notifier) SendAll(config NotifyConfig, title, message string) {
	if !config.NotifyEnabled {
		return
	}

	// Bark
	if config.BarkURL != "" {
		go n.sendBark(config.BarkURL, title, message)
	}

	// Server 酱
	if config.ServerchanKey != "" {
		go n.sendServerchan(config.ServerchanKey, title, message)
	}

	// Telegram
	if config.TelegramBotToken != "" && config.TelegramChatID != "" {
		go n.sendTelegram(config.TelegramBotToken, config.TelegramChatID, title, message)
	}

	// PushPlus
	if config.PushplusToken != "" {
		go n.sendPushplus(config.PushplusToken, title, message)
	}

	// 钉钉
	if config.DingtalkWebhook != "" {
		go n.sendDingtalk(config.DingtalkWebhook, title, message)
	}

	// 企业微信
	if config.WecomWebhook != "" {
		go n.sendWecom(config.WecomWebhook, title, message)
	}
}

// TestAll 测试所有已配置的渠道
func (n *Notifier) TestAll(config NotifyConfig) (results map[string]string) {
	results = make(map[string]string)
	title := "WoOpen 测试通知"
	message := "这是一条测试消息，如果您收到说明配置正确。\n时间: " + time.Now().Format("2006-01-02 15:04:05")

	// Bark
	if config.BarkURL != "" {
		if err := n.sendBark(config.BarkURL, title, message); err != nil {
			results["bark"] = "失败: " + err.Error()
		} else {
			results["bark"] = "成功"
		}
	}

	// Server 酱
	if config.ServerchanKey != "" {
		if err := n.sendServerchan(config.ServerchanKey, title, message); err != nil {
			results["serverchan"] = "失败: " + err.Error()
		} else {
			results["serverchan"] = "成功"
		}
	}

	// Telegram
	if config.TelegramBotToken != "" && config.TelegramChatID != "" {
		if err := n.sendTelegram(config.TelegramBotToken, config.TelegramChatID, title, message); err != nil {
			results["telegram"] = "失败: " + err.Error()
		} else {
			results["telegram"] = "成功"
		}
	}

	// PushPlus
	if config.PushplusToken != "" {
		if err := n.sendPushplus(config.PushplusToken, title, message); err != nil {
			results["pushplus"] = "失败: " + err.Error()
		} else {
			results["pushplus"] = "成功"
		}
	}

	// 钉钉
	if config.DingtalkWebhook != "" {
		if err := n.sendDingtalk(config.DingtalkWebhook, title, message); err != nil {
			results["dingtalk"] = "失败: " + err.Error()
		} else {
			results["dingtalk"] = "成功"
		}
	}

	// 企业微信
	if config.WecomWebhook != "" {
		if err := n.sendWecom(config.WecomWebhook, title, message); err != nil {
			results["wecom"] = "失败: " + err.Error()
		} else {
			results["wecom"] = "成功"
		}
	}

	return results
}

// sendBark Bark 推送 (iOS)
func (n *Notifier) sendBark(barkURL, title, message string) error {
	// Bark URL 格式: https://api.day.app/xxxxx
	// 如果只提供了 token，自动补全为完整 URL
	if !strings.HasPrefix(barkURL, "http://") && !strings.HasPrefix(barkURL, "https://") {
		barkURL = "https://api.day.app/" + barkURL
	}
	// 去掉末尾的斜杠
	barkURL = strings.TrimSuffix(barkURL, "/")

	// 使用 POST 方式发送，更可靠
	payload := map[string]interface{}{
		"title": title,
		"body":  message,
		"sound": "minuet",
		"group": "WoOpen",
	}
	jsonBody, _ := json.Marshal(payload)

	resp, err := n.httpClient.Post(barkURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		n.logNotification("bark", title, "failed", err.Error())
		return err
	}
	defer resp.Body.Close()

	// 解析响应检查是否成功
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		if code, ok := result["code"].(float64); ok && code != 200 {
			errMsg := fmt.Sprintf("Bark 返回错误: %v", result["message"])
			n.logNotification("bark", title, "failed", errMsg)
			return fmt.Errorf(errMsg)
		}
	}

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		n.logNotification("bark", title, "failed", errMsg)
		return fmt.Errorf(errMsg)
	}

	n.logNotification("bark", title, "success", "")
	return nil
}

// sendServerchan Server 酱推送 (微信)
func (n *Notifier) sendServerchan(sendKey, title, message string) error {
	apiURL := fmt.Sprintf("https://sctapi.ftqq.com/%s.send", sendKey)
	data := url.Values{}
	data.Set("title", title)
	data.Set("desp", message)

	resp, err := n.httpClient.PostForm(apiURL, data)
	if err != nil {
		n.logNotification("serverchan", title, "failed", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		n.logNotification("serverchan", title, "failed", errMsg)
		return fmt.Errorf(errMsg)
	}

	n.logNotification("serverchan", title, "success", "")
	return nil
}

// sendTelegram Telegram 推送
func (n *Notifier) sendTelegram(botToken, chatID, title, message string) error {
	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	text := fmt.Sprintf("*%s*\n\n%s", title, message)

	payload := map[string]interface{}{
		"chat_id":    chatID,
		"text":       text,
		"parse_mode": "Markdown",
	}
	jsonBody, _ := json.Marshal(payload)

	resp, err := n.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		n.logNotification("telegram", title, "failed", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		n.logNotification("telegram", title, "failed", errMsg)
		return fmt.Errorf(errMsg)
	}

	n.logNotification("telegram", title, "success", "")
	return nil
}

// sendPushplus PushPlus 推送 (微信公众号)
func (n *Notifier) sendPushplus(token, title, message string) error {
	apiURL := "https://www.pushplus.plus/send"
	payload := map[string]string{
		"token":   token,
		"title":   title,
		"content": message,
	}
	jsonBody, _ := json.Marshal(payload)

	resp, err := n.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		n.logNotification("pushplus", title, "failed", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		n.logNotification("pushplus", title, "failed", errMsg)
		return fmt.Errorf(errMsg)
	}

	n.logNotification("pushplus", title, "success", "")
	return nil
}

// sendDingtalk 钉钉机器人推送
func (n *Notifier) sendDingtalk(webhook, title, message string) error {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  fmt.Sprintf("### %s\n\n%s", title, message),
		},
	}
	jsonBody, _ := json.Marshal(payload)

	resp, err := n.httpClient.Post(webhook, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		n.logNotification("dingtalk", title, "failed", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		n.logNotification("dingtalk", title, "failed", errMsg)
		return fmt.Errorf(errMsg)
	}

	n.logNotification("dingtalk", title, "success", "")
	return nil
}

// sendWecom 企业微信机器人推送
func (n *Notifier) sendWecom(webhook, title, message string) error {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": fmt.Sprintf("### %s\n\n%s", title, message),
		},
	}
	jsonBody, _ := json.Marshal(payload)

	resp, err := n.httpClient.Post(webhook, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		n.logNotification("wecom", title, "failed", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		n.logNotification("wecom", title, "failed", errMsg)
		return fmt.Errorf(errMsg)
	}

	n.logNotification("wecom", title, "success", "")
	return nil
}

// logNotification 记录通知日志
func (n *Notifier) logNotification(channel, message, status, errMsg string) {
	if n.notificationRepo == nil {
		return
	}
	log := &model.NotificationLog{
		EventType: channel,
		Message:   message,
		Status:    status,
		ErrorMsg:  errMsg,
	}
	n.notificationRepo.Create(log)
}

// NotifyTokenInvalid 通知 Token 失效
func (n *Notifier) NotifyTokenInvalid(config NotifyConfig, errorMsg string) {
	title := "【WoOpen 告警】账号 Token 已失效"
	message := fmt.Sprintf(`时间：%s
原因：%s
状态：自动刷新失败

请访问管理后台更新 Token`, time.Now().Format("2006-01-02 15:04:05"), errorMsg)
	n.SendAll(config, title, message)
}

// NotifyTokenRefreshed 通知 Token 刷新成功
func (n *Notifier) NotifyTokenRefreshed(config NotifyConfig) {
	title := "【WoOpen 通知】Token 已自动刷新"
	message := fmt.Sprintf(`时间：%s
状态：Token 已自动刷新成功`, time.Now().Format("2006-01-02 15:04:05"))
	n.SendAll(config, title, message)
}
