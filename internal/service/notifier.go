package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	WecomAppConfig   string
	FeishuWebhook    string
	WebhookURL       string
	PushdeerKey      string
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

	// 企业微信群机器人
	if config.WecomWebhook != "" {
		go n.sendWecom(config.WecomWebhook, title, message)
	}

	// 企业微信应用消息
	if config.WecomAppConfig != "" {
		go n.sendWecomApp(config.WecomAppConfig, title, message)
	}

	// 飞书机器人
	if config.FeishuWebhook != "" {
		go n.sendFeishu(config.FeishuWebhook, title, message)
	}

	// 通用 Webhook
	if config.WebhookURL != "" {
		go n.sendWebhook(config.WebhookURL, title, message)
	}

	// PushDeer
	if config.PushdeerKey != "" {
		go n.sendPushdeer(config.PushdeerKey, title, message)
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

	// 企业微信群机器人
	if config.WecomWebhook != "" {
		if err := n.sendWecom(config.WecomWebhook, title, message); err != nil {
			results["wecom"] = "失败: " + err.Error()
		} else {
			results["wecom"] = "成功"
		}
	}

	// 企业微信应用消息
	if config.WecomAppConfig != "" {
		if err := n.sendWecomApp(config.WecomAppConfig, title, message); err != nil {
			results["wecom_app"] = "失败: " + err.Error()
		} else {
			results["wecom_app"] = "成功"
		}
	}

	// 飞书机器人
	if config.FeishuWebhook != "" {
		if err := n.sendFeishu(config.FeishuWebhook, title, message); err != nil {
			results["feishu"] = "失败: " + err.Error()
		} else {
			results["feishu"] = "成功"
		}
	}

	// 通用 Webhook
	if config.WebhookURL != "" {
		if err := n.sendWebhook(config.WebhookURL, title, message); err != nil {
			results["webhook"] = "失败: " + err.Error()
		} else {
			results["webhook"] = "成功"
		}
	}

	// PushDeer
	if config.PushdeerKey != "" {
		if err := n.sendPushdeer(config.PushdeerKey, title, message); err != nil {
			results["pushdeer"] = "失败: " + err.Error()
		} else {
			results["pushdeer"] = "成功"
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
			return errors.New(errMsg)
		}
	}

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		n.logNotification("bark", title, "failed", errMsg)
		return errors.New(errMsg)
	}

	n.logNotification("bark", title, "success", "")
	return nil
}

// sendServerchan Server 酱推送 (微信)
func (n *Notifier) sendServerchan(sendKey, title, message string) error {
	apiURL := fmt.Sprintf("https://sctapi.ftqq.com/%s.send", sendKey)
	// Server 酱³ 的 SendKey 以 sctp 开头，走独立域名 https://{num}.push.ft07.com
	if strings.HasPrefix(sendKey, "sctp") {
		num := sendKey[4:]
		if i := strings.IndexFunc(num, func(r rune) bool { return r < '0' || r > '9' }); i > 0 {
			num = num[:i]
		}
		apiURL = fmt.Sprintf("https://%s.push.ft07.com/send/%s.send", num, sendKey)
	}
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
		return errors.New(errMsg)
	}

	// 失败时同样返回 HTTP 200，错误在 body 的 code 字段（0 为成功）
	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.Code != 0 {
		errMsg := fmt.Sprintf("code=%d: %s", result.Code, result.Message)
		n.logNotification("serverchan", title, "failed", errMsg)
		return errors.New(errMsg)
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
		return errors.New(errMsg)
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
		return errors.New(errMsg)
	}

	// 失败时同样返回 HTTP 200，错误在 body 的 code 字段（200 为成功）
	var result struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.Code != 200 {
		errMsg := fmt.Sprintf("code=%d: %s", result.Code, result.Msg)
		n.logNotification("pushplus", title, "failed", errMsg)
		return errors.New(errMsg)
	}

	n.logNotification("pushplus", title, "success", "")
	return nil
}

// postJSONWithErrcode POST JSON 并检查响应中的 errcode/code 字段（企业微信、钉钉、飞书返回 HTTP 200 但错误在 body 中）
func (n *Notifier) postJSONWithErrcode(channel, apiURL string, payload interface{}, title string) error {
	jsonBody, _ := json.Marshal(payload)

	resp, err := n.httpClient.Post(apiURL, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		n.logNotification(channel, title, "failed", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		n.logNotification(channel, title, "failed", errMsg)
		return errors.New(errMsg)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
		if code, ok := result["errcode"].(float64); ok && code != 0 {
			errMsg := fmt.Sprintf("errcode=%v: %v", code, result["errmsg"])
			n.logNotification(channel, title, "failed", errMsg)
			return errors.New(errMsg)
		}
		if code, ok := result["code"].(float64); ok && code != 0 {
			errMsg := fmt.Sprintf("code=%v: %v", code, result["msg"])
			n.logNotification(channel, title, "failed", errMsg)
			return errors.New(errMsg)
		}
	}

	n.logNotification(channel, title, "success", "")
	return nil
}

// sendDingtalk 钉钉机器人推送
// webhook 格式: https://oapi.dingtalk.com/robot/send?access_token=xxx 或 "webhook,加签密钥SEC..."
func (n *Notifier) sendDingtalk(webhook, title, message string) error {
	if parts := strings.SplitN(webhook, ",", 2); len(parts) == 2 {
		secret := strings.TrimSpace(parts[1])
		ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(ts + "\n" + secret))
		sign := url.QueryEscape(base64.StdEncoding.EncodeToString(mac.Sum(nil)))
		webhook = strings.TrimSpace(parts[0]) + "&timestamp=" + ts + "&sign=" + sign
	}
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  fmt.Sprintf("### %s\n\n%s", title, message),
		},
	}
	return n.postJSONWithErrcode("dingtalk", webhook, payload, title)
}

// sendWecom 企业微信群机器人推送
func (n *Notifier) sendWecom(webhook, title, message string) error {
	payload := map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": fmt.Sprintf("### %s\n\n%s", title, message),
		},
	}
	return n.postJSONWithErrcode("wecom", webhook, payload, title)
}

// sendWecomApp 企业微信应用消息推送（推送到微信"企业微信插件"）
// config 格式: corpid,corpsecret,touser,agentid（touser 可为 @all）
func (n *Notifier) sendWecomApp(config, title, message string) error {
	parts := strings.Split(config, ",")
	if len(parts) != 4 {
		errMsg := "配置格式错误，应为: corpid,corpsecret,touser,agentid"
		n.logNotification("wecom_app", title, "failed", errMsg)
		return errors.New(errMsg)
	}
	corpid := strings.TrimSpace(parts[0])
	secret := strings.TrimSpace(parts[1])
	touser := strings.TrimSpace(parts[2])
	agentid, err := strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil {
		errMsg := "agentid 必须为数字: " + parts[3]
		n.logNotification("wecom_app", title, "failed", errMsg)
		return errors.New(errMsg)
	}
	if touser == "" {
		touser = "@all"
	}

	// 获取 access_token
	tokenURL := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s",
		url.QueryEscape(corpid), url.QueryEscape(secret))
	resp, err := n.httpClient.Get(tokenURL)
	if err != nil {
		n.logNotification("wecom_app", title, "failed", err.Error())
		return err
	}
	defer resp.Body.Close()

	var tokenResult struct {
		Errcode     int    `json:"errcode"`
		Errmsg      string `json:"errmsg"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResult); err != nil {
		n.logNotification("wecom_app", title, "failed", err.Error())
		return err
	}
	if tokenResult.AccessToken == "" {
		errMsg := fmt.Sprintf("获取 access_token 失败 errcode=%d: %s", tokenResult.Errcode, tokenResult.Errmsg)
		n.logNotification("wecom_app", title, "failed", errMsg)
		return errors.New(errMsg)
	}

	// 发送应用消息
	sendURL := "https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=" + url.QueryEscape(tokenResult.AccessToken)
	payload := map[string]interface{}{
		"touser":  touser,
		"msgtype": "text",
		"agentid": agentid,
		"text": map[string]string{
			"content": fmt.Sprintf("%s\n\n%s", title, message),
		},
	}
	return n.postJSONWithErrcode("wecom_app", sendURL, payload, title)
}

// sendFeishu 飞书机器人推送
func (n *Notifier) sendFeishu(webhook, title, message string) error {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content": map[string]string{
			"text": fmt.Sprintf("%s\n\n%s", title, message),
		},
	}
	return n.postJSONWithErrcode("feishu", webhook, payload, title)
}

// sendWebhook 通用 Webhook 推送（POST JSON: {title, message}）
func (n *Notifier) sendWebhook(webhook, title, message string) error {
	payload := map[string]string{
		"title":   title,
		"message": message,
	}
	return n.postJSONWithErrcode("webhook", webhook, payload, title)
}

// sendPushdeer PushDeer 推送
// key 格式: PDU 开头的 pushkey，自建服务可填 "https://自建地址/message/push,pushkey"
func (n *Notifier) sendPushdeer(key, title, message string) error {
	apiURL := "https://api2.pushdeer.com/message/push"
	if parts := strings.SplitN(key, ",", 2); len(parts) == 2 {
		apiURL = strings.TrimSpace(parts[0])
		key = strings.TrimSpace(parts[1])
	}
	data := url.Values{}
	data.Set("pushkey", key)
	data.Set("text", title)
	data.Set("desp", message)
	data.Set("type", "markdown")

	resp, err := n.httpClient.PostForm(apiURL, data)
	if err != nil {
		n.logNotification("pushdeer", title, "failed", err.Error())
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
		n.logNotification("pushdeer", title, "failed", errMsg)
		return errors.New(errMsg)
	}

	// 失败时同样返回 HTTP 200，错误在 body 的 code 字段（0 为成功）
	var result struct {
		Code  int    `json:"code"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err == nil && result.Code != 0 {
		errMsg := fmt.Sprintf("code=%d: %s", result.Code, result.Error)
		n.logNotification("pushdeer", title, "failed", errMsg)
		return errors.New(errMsg)
	}

	n.logNotification("pushdeer", title, "success", "")
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
