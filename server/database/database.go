// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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
		&models.QQNotification{},
		&models.QQNotificationLog{},
	); err != nil {
		log.Fatalf("❌ 数据库迁移失败: %v", err)
	}

	fmt.Println("✅ 数据库初始化成功:", dsn)
	return db
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
