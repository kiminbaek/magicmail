// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码) — original MagicMail project
// Copyright (C) 2026  kiminbaek — QQ notification feature

package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"magicmail/models"

	"gorm.io/gorm"
)

const defaultTemplate = `📧 新邮件
来自: {{.From}}
主题: {{.Subject}}
时间: {{.SentAt}}
摘要: {{.Preview}}`

// QQNotificationService QQ 邮件通知服务
type QQNotificationService struct {
	db *gorm.DB
}

// NewQQNotificationService 创建服务实例
func NewQQNotificationService(db *gorm.DB) *QQNotificationService {
	return &QQNotificationService{db: db}
}

// HandleNotification 通知回调入口（注册到 notifier，由 TriggerByEvent 调用）
func (s *QQNotificationService) HandleNotification(event string, data map[string]interface{}) {
	if event != "mail.received" {
		return
	}

	// 单用户模式，user_id = 1
	config, err := s.GetConfig(1)
	if err != nil || config == nil || !config.Enabled {
		return
	}

	// 过滤规则
	if !shouldSend(config, data) {
		return
	}

	// 静默时段
	if isInSilentHour(config.SilentStart, config.SilentEnd) {
		return
	}

	// 渲染模板
	text := renderTemplate(config.Template, data)
	if text == "" {
		text = defaultTemplate
	}

	// 调用 QwenPaw API
	err = s.callQwenPawAPI(config, text)

	// 记录日志
	s.logResult(config, data, err)
}

// GetConfig 获取用户配置（不存在则返回默认值，不写库）
func (s *QQNotificationService) GetConfig(userID uint) (*models.QQNotification, error) {
	var config models.QQNotification
	result := s.db.Where("user_id = ?", userID).First(&config)
	if result.Error == gorm.ErrRecordNotFound {
		return &models.QQNotification{
			UserID:        userID,
			Enabled:       false,
			QwenPawURL:    "http://127.0.0.1:19091",
			AgentID:       "003",
			Template:      defaultTemplate,
			PreviewLength: 200,
		}, nil
	}
	if result.Error != nil {
		return nil, result.Error
	}
	if config.Template == "" {
		config.Template = defaultTemplate
	}
	if config.PreviewLength == 0 {
		config.PreviewLength = 200
	}
	return &config, nil
}

// SaveConfig 保存配置（upsert）
func (s *QQNotificationService) SaveConfig(userID uint, req *models.QQNotificationRequest) (*models.QQNotification, error) {
	config, err := s.GetConfig(userID)
	if err != nil {
		return nil, err
	}

	// 应用请求中的字段
	if req.Enabled != nil {
		config.Enabled = *req.Enabled
	}
	if req.QwenPawURL != nil {
		config.QwenPawURL = *req.QwenPawURL
	}
	if req.AgentID != nil {
		config.AgentID = *req.AgentID
	}
	if req.TargetUser != nil {
		config.TargetUser = *req.TargetUser
	}
	if req.TargetSession != nil {
		config.TargetSession = *req.TargetSession
	}
	if req.FilterFrom != nil {
		config.FilterFrom = *req.FilterFrom
	}
	if req.FilterSubject != nil {
		config.FilterSubject = *req.FilterSubject
	}
	if req.ExcludeFrom != nil {
		config.ExcludeFrom = *req.ExcludeFrom
	}
	if req.ExcludeSubject != nil {
		config.ExcludeSubject = *req.ExcludeSubject
	}
	if req.SilentStart != nil {
		config.SilentStart = *req.SilentStart
	}
	if req.SilentEnd != nil {
		config.SilentEnd = *req.SilentEnd
	}
	if req.Template != nil {
		config.Template = *req.Template
	}
	if req.PreviewLength != nil {
		config.PreviewLength = *req.PreviewLength
	}

	config.UserID = userID
	result := s.db.Save(&config)
	if result.Error != nil {
		return nil, result.Error
	}
	return config, nil
}

// TestNotification 发送测试通知
func (s *QQNotificationService) TestNotification(userID uint) error {
	config, err := s.GetConfig(userID)
	if err != nil {
		return err
	}
	if config.AgentID == "" || config.TargetUser == "" || config.TargetSession == "" {
		return fmt.Errorf("请先填写 Agent ID、Target User 和 Target Session")
	}

	text := "📧 QQ 邮件通知测试\n这是一条来自 MagicMail 的测试消息。\n如果你收到了，说明 QQ 通知配置成功！"
	return s.callQwenPawAPI(config, text)
}

// GetLogs 获取发送日志
func (s *QQNotificationService) GetLogs(userID uint, limit int) ([]models.QQNotificationLog, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	config, err := s.GetConfig(userID)
	if err != nil {
		return nil, err
	}

	var logs []models.QQNotificationLog
	err = s.db.Where("config_id = ?", config.ID).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// DeleteLog 删除单条日志
func (s *QQNotificationService) DeleteLog(logID uint) error {
	return s.db.Delete(&models.QQNotificationLog{}, logID).Error
}

// callQwenPawAPI 调用 QwenPaw messages/send API
func (s *QQNotificationService) callQwenPawAPI(config *models.QQNotification, text string) error {
	url := strings.TrimSuffix(config.QwenPawURL, "/") + "/api/messages/send"

	payload := map[string]string{
		"channel":        "qq",
		"target_user":    config.TargetUser,
		"target_session": config.TargetSession,
		"text":           text,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-Id", config.AgentID)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// logResult 记录发送日志 + 更新统计
func (s *QQNotificationService) logResult(config *models.QQNotification, data map[string]interface{}, err error) {
	getStr := func(key string) string {
		if v, ok := data[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}

	logEntry := models.QQNotificationLog{
		ConfigID:    config.ID,
		MailFrom:    getStr("from"),
		MailSubject: getStr("subject"),
		Status:      "success",
	}
	if err != nil {
		logEntry.Status = "failed"
		logEntry.ErrorMsg = err.Error()
	}

	if e := s.db.Create(&logEntry).Error; e != nil {
		log.Printf("[QQ-Notify] 写日志失败: %v", e)
	}

	updates := map[string]interface{}{
		"last_sent_at": time.Now(),
		"total_sent":   gorm.Expr("total_sent + 1"),
	}
	if err != nil {
		updates["last_error"] = err.Error()
	} else {
		updates["last_error"] = ""
	}
	s.db.Model(&models.QQNotification{}).Where("id = ?", config.ID).Updates(updates)
}

// shouldSend 检查过滤规则
func shouldSend(config *models.QQNotification, data map[string]interface{}) bool {
	getStr := func(key string) string {
		if v, ok := data[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	from := getStr("from")
	subject := getStr("subject")

	// 发件人白名单
	if config.FilterFrom != "" {
		if !matchAny(from, config.FilterFrom) {
			return false
		}
	}
	// 主题关键词白名单
	if config.FilterSubject != "" {
		if !matchAny(subject, config.FilterSubject) {
			return false
		}
	}
	// 发件人黑名单
	if config.ExcludeFrom != "" {
		if matchAny(from, config.ExcludeFrom) {
			return false
		}
	}
	// 排除主题关键词
	if config.ExcludeSubject != "" {
		if matchAny(subject, config.ExcludeSubject) {
			return false
		}
	}
	return true
}

// matchAny 检查 value 是否包含 keywords 中的任意一个（逗号分隔）
func matchAny(value, keywords string) bool {
	for _, k := range strings.Split(keywords, ",") {
		k = strings.TrimSpace(k)
		if k != "" && strings.Contains(strings.ToLower(value), strings.ToLower(k)) {
			return true
		}
	}
	return false
}

// isInSilentHour 检查当前是否在静默时段
func isInSilentHour(start, end string) bool {
	if start == "" || end == "" {
		return false
	}
	now := time.Now()
	current := now.Hour()*60 + now.Minute()
	s, _ := parseHHMM(start)
	e, _ := parseHHMM(end)
	if s < e {
		return current >= s && current <= e
	}
	// 跨天（如 23:00-08:00）
	return current >= s || current <= e
}

// parseHHMM 解析 "HH:MM" 为分钟数
func parseHHMM(s string) (int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid time format")
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, err
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, err
	}
	return h*60 + m, nil
}

// renderTemplate 渲染通知模板
func renderTemplate(template string, data map[string]interface{}) string {
	if template == "" {
		template = defaultTemplate
	}
	getStr := func(key string) string {
		if v, ok := data[key]; ok {
			return fmt.Sprintf("%v", v)
		}
		return ""
	}
	result := template
	result = strings.ReplaceAll(result, "{{.From}}", getStr("from"))
	result = strings.ReplaceAll(result, "{{.Subject}}", getStr("subject"))
	result = strings.ReplaceAll(result, "{{.SentAt}}", getStr("sent_at"))
	result = strings.ReplaceAll(result, "{{.Preview}}", getStr("preview"))
	result = strings.ReplaceAll(result, "{{.AccountEmail}}", getStr("account_email"))
	result = strings.ReplaceAll(result, "{{.AccountName}}", getStr("account_name"))
	return result
}
