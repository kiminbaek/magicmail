// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package models

import (
	"log"
	"time"

	"magicmail/crypto"

	"gorm.io/gorm"
)

// MailAccount 邮箱账号模型 - 存储用户配置的 IMAP/POP3/SMTP 邮箱连接信息
//
// Password 字段在数据库中以 AES-256-GCM 密文存储（ENC: 前缀标识），
// 通过 GORM 钩子实现写入时自动加密、读取时自动解密，业务层无需关心。
type MailAccount struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	Name      string         `json:"name" gorm:"type:varchar(100);not null;comment:显示名称"`
	Email     string         `json:"email" gorm:"type:varchar(255);not null;uniqueIndex;comment:邮箱地址"`
	Protocol  string         `json:"protocol" gorm:"type:varchar(10);not null;default:'imap';comment:邮件协议(imap/pop3)"`
	ImapHost  string         `json:"host" gorm:"type:varchar(255);not null;comment:收信服务器地址"`
	Port      int            `json:"port" gorm:"default:993;comment:收信端口"`
	SmtpHost  string         `json:"smtp_host" gorm:"type:varchar(255);comment:SMTP发信服务器地址(为空则使用收信地址)"`
	SmtpPort  int            `json:"smtp_port" gorm:"comment:SMTP端口(默认587)"`
	Username  string         `json:"username" gorm:"type:varchar(255);not null;comment:登录用户名"`
	Password  string         `json:"-" gorm:"type:text;not null;comment:密码(AES-256-GCM加密存储)"`
	ProxyEnabled bool       `json:"proxy_enabled" gorm:"default:false;comment:是否启用HTTP代理"`
	ProxyURL     string     `json:"proxy_url,omitempty" gorm:"type:varchar(512);comment:HTTP代理地址(http://user:pass@host:port)"`
	SyncMode     string     `json:"sync_mode" gorm:"type:varchar(20);default:'unread';comment:同步模式(unread/all/recent)"`
	SyncDays     int        `json:"sync_days" gorm:"default:30;comment:同步最近天数(仅sync_mode=recent时有效)"`
	DeleteOnServer bool      `json:"delete_on_server" gorm:"default:false;comment:删除时是否同步删除源服务器上的邮件"`
	LastSyncAt   *time.Time `json:"last_sync_at" gorm:"comment:最后同步时间"`
	Status       string     `json:"status" gorm:"type:varchar(20);default:'active';index;comment:状态(active/error/disabled)"`
	ErrorMsg     string     `json:"error_msg,omitempty" gorm:"type:text;comment:错误信息"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 关联
	Mails []Mail `json:"mails,omitempty" gorm:"foreignKey:AccountID"`
}

// TableName 指定表名
func (MailAccount) TableName() string {
	return "mail_accounts"
}

// BeforeCreate 创建前钩子：设置默认值 + 加密密码
func (a *MailAccount) BeforeCreate(tx *gorm.DB) error {
	a.setDefaultValues()
	return a.encryptPassword()
}

// BeforeUpdate 更新前钩子：加密密码（仅当密码字段被更新时）
func (a *MailAccount) BeforeUpdate(tx *gorm.DB) error {
	// GORM 的 Changed 方法仅在 tx.Statement 中可用，此处通过判断值是否已加密来避免重复加密
	if a.Password != "" && !crypto.IsEncrypted(a.Password) {
		encrypted, err := crypto.Encrypt(a.Password)
		if err != nil {
			return err
		}
		a.Password = encrypted
	}
	return nil
}

// AfterFind 查询后钩子：自动解密密码供运行时使用（IMAP/POP3/SMTP 认证等）
func (a *MailAccount) AfterFind(tx *gorm.DB) error {
	if a.Password != "" {
		decrypted, err := crypto.Decrypt(a.Password)
		if err != nil {
			log.Printf("[WARN] MailAccount#%d 密码解密失败: %v", a.ID, err)
			a.Password = "" // 标记为空，但不阻断查询
			return nil
		}
		a.Password = decrypted
	}
	return nil
}

// setDefaultValues 设置默认值
func (a *MailAccount) setDefaultValues() {
	if a.Status == "" {
		a.Status = "active"
	}
	if a.Protocol == "" {
		a.Protocol = "imap"
	}
	if a.Port == 0 {
		a.Port = DefaultPort(a.Protocol)
	}
	if a.SyncMode == "" {
		a.SyncMode = "unread" // 默认只同步未读邮件
	}
	if a.SyncDays == 0 {
		a.SyncDays = 30 // 默认最近30天
	}
}

// encryptPassword 加密明文密码（已是密文则跳过）
func (a *MailAccount) encryptPassword() error {
	if a.Password == "" {
		return nil
	}
	if crypto.IsEncrypted(a.Password) {
		return nil // 已经是密文（如导入场景）
	}
	encrypted, err := crypto.Encrypt(a.Password)
	if err != nil {
		return err
	}
	a.Password = encrypted
	return nil
}

// DefaultPort 根据协议返回默认端口
func DefaultPort(protocol string) int {
	switch protocol {
	case "pop3":
		return 995 // POP3S (SSL/TLS)
	case "pop3-no-ssl":
		return 110
	default:
		return 993 // IMAPS (SSL/TLS)
	}
}

// AccountRequest 新增/编辑邮箱的请求体（不暴露密码字段）
type AccountRequest struct {
	Name         string `json:"name" validate:"required,min=1,max=100"`
	Email        string `json:"email" validate:"required,email"`
	Protocol     string `json:"protocol" validate:"omitempty,oneof=imap pop3 pop3-no-ssl"`
	Host         string `json:"host" validate:"required"`
	Port         int    `json:"port" validate:"min=1,max=65535"`
	SmtpHost     string `json:"smtp_host"`
	SmtpPort     int    `json:"smtp_port" validate:"omitempty,min=1,max=65535"`
	Username     string `json:"username" validate:"required"`
	Password     string `json:"password"` // 为空时不更新密码
	ProxyEnabled bool   `json:"proxy_enabled"` // 是否启用HTTP代理
	ProxyURL     string `json:"proxy_url,omitempty"` // HTTP代理地址
	SyncMode     string `json:"sync_mode" validate:"omitempty,oneof=unread all recent"` // 同步模式
	SyncDays     int    `json:"sync_days" validate:"omitempty,min=1,max=365"`           // 最近天数
	DeleteOnServer bool  `json:"delete_on_server"`                                      // 删除时同步到源服务器
}

// AccountResponse 邮箱 API 响应（脱敏）
type AccountResponse struct {
	ID           uint        `json:"id"`
	Name         string      `json:"name"`
	Email        string      `json:"email"`
	Protocol     string      `json:"protocol"`
	ImapHost     string      `json:"host"`
	Port         int         `json:"port"`
	SmtpHost     string      `json:"smtp_host,omitempty"`
	SmtpPort     int         `json:"smtp_port,omitempty"`
	Username     string      `json:"username"`
	HasPassword  bool        `json:"has_password"` // 是否已设置密码
	ProxyEnabled bool        `json:"proxy_enabled"` // 是否启用HTTP代理
	ProxyURL     string      `json:"proxy_url,omitempty"` // HTTP代理地址
	SyncMode     string      `json:"sync_mode"`            // 同步模式
	SyncDays     int         `json:"sync_days"`            // 最近天数
	DeleteOnServer bool       `json:"delete_on_server"`     // 删除时同步到源服务器
	LastSyncAt   *time.Time  `json:"last_sync_at"`
	Status       string      `json:"status"`
	ErrorMsg     string      `json:"error_msg,omitempty"`
	MailCount    int64       `json:"mail_count"`
	UnreadCount  int64       `json:"unread_count"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

// AccountListDTO 列表查询专用 DTO（不含 Password 字段，避免触发 AfterFind 解密）
// 用于列表场景：只需展示基本信息，不需要密码明文
type AccountListDTO struct {
	ID             uint       `gorm:"primaryKey" json:"id"`
	Name           string     `json:"name"`
	Email          string     `json:"email"`
	Protocol       string     `json:"protocol"`
	ImapHost       string     `gorm:"column:imap_host" json:"host"`
	Port           int        `json:"port"`
	SmtpHost       string     `gorm:"column:smtp_host" json:"smtp_host,omitempty"`
	SmtpPort       int        `gorm:"column:smtp_port" json:"smtp_port,omitempty"`
	Username       string     `json:"username"`
	PasswordRaw    string     `gorm:"column:password" json:"-"` // 接收原始密码密文，不对外暴露
	ProxyEnabled   bool       `gorm:"column:proxy_enabled" json:"proxy_enabled"`
	ProxyURL       string     `gorm:"column:proxy_url" json:"proxy_url,omitempty"`
	SyncMode       string     `gorm:"column:sync_mode" json:"sync_mode"`
	SyncDays       int        `gorm:"column:sync_days" json:"sync_days"`
	DeleteOnServer bool        `gorm:"column:delete_on_server" json:"delete_on_server"`
	LastSyncAt     *time.Time `gorm:"column:last_sync_at" json:"last_sync_at"`
	Status         string     `gorm:"column:status" json:"status"`
	ErrorMsg       string     `gorm:"column:error_msg" json:"error_msg,omitempty"`
	CreatedAt      time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt      time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// HasPassword 判断是否设置了密码（PasswordRaw 非空即有密码）
func (dto *AccountListDTO) HasPassword() bool {
	return dto.PasswordRaw != ""
}

// TableName 指定 DTO 的表名（与 MailAccount 共享同一张表）
func (AccountListDTO) TableName() string {
	return "mail_accounts"
}

// DefaultPort 根据协议返回默认端口
