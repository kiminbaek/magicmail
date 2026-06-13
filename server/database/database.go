// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"unicode"

	"magicmail/models"

	// 纯 Go SQLite 驱动（基于 modernc.org/sqlite，无需 CGO）
	"github.com/glebarez/sqlite"

	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Init 初始化 SQLite 数据库连接并执行自动迁移
func Init(dsn string) *gorm.DB {
	// 确保 data 目录存在
	dbDir := filepath.Dir(dsn)
	if dbDir != "." && dbDir != "" {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			log.Fatalf("❌ 无法创建数据目录 %s: %v", dbDir, err)
		}
	}

	// 按环境控制 SQL 日志：生产环境静默，开发环境输出 SQL
	logLevel := gormlogger.Silent
	if os.Getenv("MAGICMAIL_ENV") != "production" {
		logLevel = gormlogger.Info
	}

	// 连接 SQLite
	db, err := gorm.Open(sqlite.Open(dsn+"?_journal_mode=WAL&_busy_timeout=5000"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(logLevel),
	})
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	// 获取底层数据库连接并配置
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("❌ 获取数据库实例失败: %v", err)
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)

	// 自动迁移表结构
	if err := db.AutoMigrate(
		&models.MailAccount{},
		&models.Mail{},
		&models.Attachment{},
		&models.Webhook{},
		&models.WebhookLog{},
		&models.User{},
		&models.AppConfig{},
		&models.Draft{},
		&models.PushSubscription{},
	); err != nil {
		log.Fatalf("❌ 数据库迁移失败: %v", err)
	}

	// 清理历史遗留的多余列（Host 字段曾短暂映射到 host 导致 AutoMigrate 多建）
	cleanOrphanColumns(db)

	// 补充可能缺失的 VAPID 密钥列（AutoMigrate 对已有表新增列有时不生效）
	ensureVAPIDColumns(db)

	fmt.Println("✅ 数据库初始化成功:", dsn)
	return db
}

// cleanOrphanColumns 清理 AutoMigrate 可能遗留的多余列
func cleanOrphanColumns(db *gorm.DB) {
	// mail_accounts 表：Host 字段正确映射为 imap_host，删除多余的 host 列
	var colType string
	if err := db.Raw("SELECT type FROM pragma_table_info('mail_accounts') WHERE name = 'host'").Scan(&colType).Error; err == nil && colType != "" {
		if err := db.Exec("ALTER TABLE mail_accounts DROP COLUMN host").Error; err != nil {
			log.Printf("⚠️ 清理多余列 host 失败（可忽略）: %v", err)
		} else {
			log.Println("🧹 已清理 mail_accounts 多余列: host")
		}
	}
}

// ensureVAPIDColumns 确保 app_configs 表包含 VAPID 密钥列（AutoMigrate 对已有表新增字段可能不生效）
func ensureVAPIDColumns(db *gorm.DB) {
	columns := []struct{ name, typ string }{
		{"vapid_public_key", "TEXT DEFAULT ''"},
		{"vapid_private_key", "TEXT DEFAULT ''"},
	}
	for _, col := range columns {
		var exists string
		if err := db.Raw("SELECT name FROM pragma_table_info('app_configs') WHERE name = ?", col.name).Scan(&exists).Error; err != nil || exists == "" {
			// 安全验证：确保列名为合法的 SQL 标识符
			if !isValidSQLIdentifier(col.name) {
				log.Printf("⚠️ 无效的列名: %s，跳过添加", col.name)
				continue
			}
			if err := db.Exec("ALTER TABLE app_configs ADD COLUMN \"" + col.name + "\" " + col.typ).Error; err != nil {
				log.Printf("⚠️ 添加列 %s 失败: %v", col.name, err)
			} else {
				log.Printf("✅ 已补充 app_configs 列: %s", col.name)
			}
		}
	}
}

// isValidSQLIdentifier 验证字符串是否为合法的 SQL 标识符（仅允许字母、数字、下划线）
func isValidSQLIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}
	for i, r := range s {
		if !(unicode.IsLetter(r) || (i > 0 && unicode.IsDigit(r)) || r == '_') {
			return false
		}
	}
	return true
}

// EnsureSecuritySecrets 确保安全密钥存在：
//   - 首次启动时优先使用环境变量传入的密钥（MAGICMAIL_JWT_SECRET / MAGICMAIL_ENCRYPT_KEY），
//     未设置则自动生成随机密钥，最终持久化到数据库；
//   - 后续启动从数据库读取，但若环境变量有新值则以环境变量为准（覆盖数据库值）。
func EnsureSecuritySecrets(db *gorm.DB, jwtSecret, encryptionKey *string) {
	// 优先从环境变量读取用户指定的密钥
	envJWT := os.Getenv("MAGICMAIL_JWT_SECRET")
	envEncKey := os.Getenv("MAGICMAIL_ENCRYPT_KEY")

	var cfg models.AppConfig
	result := db.First(&cfg)

	if result.Error != nil {
		// 首次启动：环境变量 > 自动生成
		jwtSec := envJWT
		encKey := envEncKey
		var err error

		if jwtSec == "" {
			jwtSec, err = models.GenerateRandomKey(32)
			if err != nil {
				log.Fatalf("❌ 生成 JWT 密钥失败: %v", err)
			}
			log.Println("🔑 JWT 密钥：已自动生成随机密钥")
		} else {
			log.Println("🔑 JWT 密钥：从环境变量 MAGICMAIL_JWT_SECRET 读取")
		}

		if encKey == "" {
			encKey, err = models.GenerateRandomKey(32)
			if err != nil {
				log.Fatalf("❌ 生成加密密钥失败: %v", err)
			}
			log.Println("🔐 加密密钥：已自动生成随机密钥")
		} else {
			log.Println("🔐 加密密钥：从环境变量 MAGICMAIL_ENCRYPT_KEY 读取")
		}

		cfg = models.AppConfig{
			JWTSecret:     jwtSec,
			EncryptionKey: encKey,
		}
		if err := db.Create(&cfg).Error; err != nil {
			log.Fatalf("❌ 保存安全配置失败: %v", err)
		}

		*jwtSecret = jwtSec
		*encryptionKey = encKey
	} else {
		// 已有记录：环境变量 > 数据库存储
		useJWT := cfg.JWTSecret
		useEncKey := cfg.EncryptionKey
		sourceJWT := "数据库"
		sourceEncKey := "数据库"

		if envJWT != "" && envJWT != cfg.JWTSecret {
			useJWT = envJWT
			sourceJWT = "环境变量"
			// 同步更新数据库，保证下次启动一致
			cfg.JWTSecret = envJWT
			db.Save(&cfg)
		}

		if envEncKey != "" && envEncKey != cfg.EncryptionKey {
			useEncKey = envEncKey
			sourceEncKey = "环境变量"
			cfg.EncryptionKey = envEncKey
			db.Save(&cfg)
		}

		*jwtSecret = useJWT
		*encryptionKey = useEncKey
		log.Printf("🔐 安全密钥已加载（JWT 来源：%s，加密密钥 来源：%s）", sourceJWT, sourceEncKey)
	}
}
