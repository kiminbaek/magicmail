// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码) — original MagicMail project
// Copyright (C) 2026  kiminbaek — QQ notification feature

package models

import "time"

// QQNotification QQ 邮件通知配置（单用户模式，UserID 默认 1）
type QQNotification struct {
	ID       uint   `json:"id" gorm:"primaryKey"`
	UserID   uint   `json:"user_id" gorm:"default:1;index"`
	Enabled  bool   `json:"enabled" gorm:"default:false"`

	// QwenPaw API 配置
	QwenPawURL    string `json:"qwenpaw_url" gorm:"type:varchar(255);default:'http://127.0.0.1:19091';comment:QwenPaw API 地址"`
	AgentID       string `json:"agent_id" gorm:"type:varchar(50);comment:发送 Agent ID (如 003)"`
	TargetUser    string `json:"target_user" gorm:"type:varchar(255);comment:目标用户 ID"`
	TargetSession string `json:"target_session" gorm:"type:varchar(255);comment:目标会话 ID"`

	// 过滤规则（逗号分隔，空 = 不限制）
	FilterFrom     string `json:"filter_from,omitempty" gorm:"type:text;comment:发件人白名单"`
	FilterSubject  string `json:"filter_subject,omitempty" gorm:"type:text;comment:主题关键词白名单"`
	ExcludeFrom    string `json:"exclude_from,omitempty" gorm:"type:text;comment:发件人黑名单"`
	ExcludeSubject string `json:"exclude_subject,omitempty" gorm:"type:text;comment:排除主题关键词"`
	SilentStart    string `json:"silent_start,omitempty" gorm:"type:varchar(5);comment:静默开始 HH:MM"`
	SilentEnd      string `json:"silent_end,omitempty" gorm:"type:varchar(5);comment:静默结束 HH:MM"`

	// 通知模板
	Template      string `json:"template" gorm:"type:text;comment:通知模板"`
	PreviewLength int    `json:"preview_length" gorm:"default:200;comment:摘要长度"`

	// 统计
	LastSentAt *time.Time `json:"last_sent_at,omitempty"`
	TotalSent  int        `json:"total_sent" gorm:"default:0"`
	LastError  string     `json:"last_error,omitempty" gorm:"type:text"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// QQNotificationLog QQ 通知发送日志
type QQNotificationLog struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	ConfigID    uint      `json:"config_id" gorm:"index;not null"`
	MailFrom    string    `json:"mail_from,omitempty" gorm:"type:varchar(500)"`
	MailSubject string    `json:"mail_subject,omitempty" gorm:"type:varchar(500)"`
	Status     string    `json:"status" gorm:"type:varchar(20);not null;comment:success/failed"`
	ErrorMsg   string    `json:"error_msg,omitempty" gorm:"type:text"`
	CreatedAt  time.Time `json:"created_at"`
}

// QQNotificationRequest API 请求体（所有字段可选，nil = 不修改）
type QQNotificationRequest struct {
	Enabled        *bool   `json:"enabled"`
	QwenPawURL     *string `json:"qwenpaw_url"`
	AgentID        *string `json:"agent_id"`
	TargetUser     *string `json:"target_user"`
	TargetSession  *string `json:"target_session"`
	FilterFrom     *string `json:"filter_from"`
	FilterSubject  *string `json:"filter_subject"`
	ExcludeFrom    *string `json:"exclude_from"`
	ExcludeSubject *string `json:"exclude_subject"`
	SilentStart    *string `json:"silent_start"`
	SilentEnd      *string `json:"silent_end"`
	Template       *string `json:"template"`
	PreviewLength  *int    `json:"preview_length"`
}
