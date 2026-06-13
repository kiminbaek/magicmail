// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package imap

import (
	"crypto/tls"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"magicmail/config"
	"magicmail/models"
	"magicmail/notifier"
	"magicmail/sse"
	pop3pkg "magicmail/pop3"

	"github.com/emersion/go-imap/v2/imapclient"
	"gorm.io/gorm"
)

// WorkerPool 管理所有邮箱账号的后台同步协程
type WorkerPool struct {
	db           *gorm.DB
	config       *config.Config
	workers      map[uint]*AccountWorker // accountID -> worker
	mu           sync.RWMutex
	shutdown     int32 // 原子标志：1=关闭中
	shutdownCh   chan struct{}
	wg           sync.WaitGroup
	sem          chan struct{} // 并发信号量
}

var globalPool *WorkerPool

// GlobalPool exposes the worker pool for external packages (services/handlers)
// Returns nil if workers haven't been started yet
func GlobalPool() *WorkerPool {
	return globalPool
}

// StartWorkers 启动所有活跃邮箱的后台同步 Worker（程序启动时调用）
func StartWorkers(db *gorm.DB, cfg *config.Config) {
	pool := &WorkerPool{
		db:         db,
		config:     cfg,
		workers:    make(map[uint]*AccountWorker),
		shutdownCh: make(chan struct{}),
		sem:        make(chan struct{}, cfg.IMAP.MaxConcurrent),
	}
	globalPool = pool

	// 查询所有活跃的邮箱账号
	var accounts []models.MailAccount
	if err := db.Where("status = ?", "active").Find(&accounts).Error; err != nil {
		log.Printf("❌ 查询邮箱账号失败: %v", err)
		return
	}

	if len(accounts) == 0 {
		log.Println("📭 没有活跃的邮箱账号")
		return
	}

	log.Printf("🚀 启动 %d 个邮箱同步 Worker...", len(accounts))

	for i := range accounts {
		pool.StartWorker(&accounts[i])
	}
}

// StopWorkers 优雅关闭所有 Worker
func StopWorkers() {
	if globalPool == nil {
		return
	}
	atomic.StoreInt32(&globalPool.shutdown, 1)
	close(globalPool.shutdownCh)

	globalPool.mu.RLock()
	for _, w := range globalPool.workers {
		w.Stop()
	}
	globalPool.mu.RUnlock()

	globalPool.wg.Wait()
	log.Println("🛑 所有 IMAP Worker 已停止")
}

// StartWorker 为单个邮箱账号启动同步协程
func (p *WorkerPool) StartWorker(account *models.MailAccount) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// 如果已有该账号的 Worker，先停掉旧的
	if existing, ok := p.workers[account.ID]; ok {
		existing.Stop()
	}

	w := NewAccountWorker(account, p.db, p.config, p.sem, p.shutdownCh)
	p.workers[account.ID] = w

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		w.Run()
	}()

	log.Printf("▶️  Worker 启动: %s (%s)", account.Email, account.Name)
}

// StopWorker 停止指定账号的 Worker
func (p *WorkerPool) StopWorker(accountID uint) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if w, ok := p.workers[accountID]; ok {
		w.Stop()
		delete(p.workers, accountID)
	}
}

// RestartWorker 重启指定账号的 Worker（配置变更后调用）
func (p *WorkerPool) RestartWorker(account *models.MailAccount) {
	p.StartWorker(account)
}

// AccountWorker 单个邮箱账号的同步 Worker
type AccountWorker struct {
	account    *models.MailAccount
	db         *gorm.DB
	config     *config.Config
	sem        chan struct{}
	shutdownCh chan struct{}
	stopCh     chan struct{} // 该 Worker 的独立停止通道
}

// NewAccountWorker 创建新的账号 Worker
func NewAccountWorker(account *models.MailAccount, db *gorm.DB, cfg *config.Config, sem chan struct{}, shutdownCh chan struct{}) *AccountWorker {
	return &AccountWorker{
		account:    account,
		db:         db,
		config:     cfg,
		sem:        sem,
		shutdownCh: shutdownCh,
		stopCh:     make(chan struct{}),
	}
}

// Run 启动 Worker 主循环：先做一次全量同步，然后进入 IDLE（仅 IMAP）或轮询模式
func (w *AccountWorker) Run() {
	defer log.Printf("⏹️  Worker 退出: %s", w.account.Email)

	ticker := time.NewTicker(time.Duration(w.config.IMAP.PollInterval) * time.Second)
	defer ticker.Stop()

	// 首次全量同步
	w.syncOnce()

	for {
		select {
		case <-w.stopCh:
			return
		case <-w.shutdownCh:
			return
		case <-ticker.C:
			// 定时轮询同步
			w.syncOnce()
		default:
			// 仅 IMAP 协议支持 IDLE 实时监听；POP3 不支持 IDLE，直接等待下次轮询
			if w.isIMAP() && w.config.IMAP.IDLEEnabled {
				if err := w.idleLoop(); err != nil {
					log.Printf("⚠️  IDLE 异常 (%s): %v，降级为轮询", w.account.Email, err)
					select {
					case <-time.After(30 * time.Second):
					case <-w.stopCh:
						return
					case <-w.shutdownCh:
						return
					}
				} else {
					// idleLoop 正常返回说明检测到新邮件或超时，立即同步
					w.syncOnce()
				}
			} else {
				// POP3 或未启用 IDLE：等待下一次定时触发
				select {
				case <-ticker.C:
				case <-w.stopCh:
					return
				case <-w.shutdownCh:
					return
				}
			}
		}
	}
}

// syncOnce 执行单次完整同步
func (w *AccountWorker) syncOnce() {
	// 获取并发令牌
	select {
	case w.sem <- struct{}{}:
	default:
		log.Printf("⏳ 并发已满，跳过本次同步: %s", w.account.Email)
		return
	}
	defer func() { <-w.sem }()

	// 重新从数据库获取最新账号信息（密码可能被更新）
	var fresh models.MailAccount
	if err := w.db.First(&fresh, w.account.ID).Error; err != nil {
		log.Printf("❌ 无法获取账号信息 (ID=%d): %v", w.account.ID, err)
		return
	}
	w.account = &fresh

	// 根据协议创建对应的邮件客户端
	client, err := NewMailClient(w.account, w.config)
	if err != nil {
		w.updateAccountStatus("error", err.Error())
		return
	}
	defer client.Close()

	// 认证
	if err := client.Authenticate(); err != nil {
		w.updateAccountStatus("error", err.Error())
		return
	}

	// 根据协议选择对应的拉取器执行同步
	var count int
	if w.isIMAP() {
		// IMAP 同步（INBOX + Sent）
		imapClient := client.(*IMAPClient)
		fetcher := NewFetcher(w.db, w.config)
		count, err = fetcher.SyncMailbox(imapClient)
		if err == nil {
			// 继续同步已发送文件夹（失败不阻止主流程）
			if sentCount, sentErr := fetcher.SyncSentMailbox(imapClient); sentErr == nil {
				count += sentCount
			}
		}
	} else {
		// POP3 同步
		pop3Client := client.(*pop3pkg.POP3Client)
		pop3Fetcher := pop3pkg.NewPOP3Fetcher(w.db, w.config)
		count, err = pop3Fetcher.SyncMailbox(pop3Client)
	}

	if err != nil {
		w.updateAccountStatus("error", err.Error())
		return
	}

	// 同步成功，更新状态和时间
	now := time.Now()
	w.db.Model(&models.MailAccount{}).Where("id = ?", w.account.ID).
		Updates(map[string]interface{}{
			"last_sync_at": now,
			"status":      "active",
			"error_msg":   "",
		})

	if count > 0 {
		log.Printf("📬 %s 同步完成: 新增 %d 封邮件", w.account.Email, count)

		// 查询最新邮件详情用于 webhook 推送
		var latestMails []struct {
			Subject string `json:"subject"`
			From    string `json:"from"`
			SentAt  time.Time `json:"sent_at"`
			Preview string `json:"preview"`
		}
		w.db.Table("mails").
			Select("subject, `from`, sent_at, text_body").
			Where("account_id = ?", w.account.ID).
			Order("sent_at DESC").
			Limit(count).
			Find(&latestMails)

		mailList := make([]map[string]interface{}, len(latestMails))
		for i, m := range latestMails {
			// 预览：截取正文前 100 字符
			preview := m.Preview
			if len(preview) > 100 { preview = preview[:100] + "..." }
			mailList[i] = map[string]interface{}{
				"subject": m.Subject,
				"from":    m.From,
				"sent_at": m.SentAt.Format("2006-01-02 15:04:05"),
				"preview": preview,
			}
		}

		// 触发 Webhook 通知（包含邮件详情）
		notifier.TriggerByEvent(w.db, "mail.received", map[string]interface{}{
			"account_id":    w.account.ID,
			"account_email": w.account.Email,
			"account_name":  w.account.Name,
			"protocol":      w.account.Protocol,
			"mail_count":    count,
			"mails":         mailList,
		})

		// 推送 SSE 实时事件给前端
		sse.PublishMailReceived(w.account.ID, w.account.Email, count, mailList)

		// 发送 Web Push 离线推送通知（通过 notifier 包桥接，避免循环依赖）
		notifier.SendPushNotification(
			1,
			fmt.Sprintf("📧 您有 %d 封新邮件", count),
			fmt.Sprintf("来自 %s", w.account.Email),
			map[string]interface{}{"account_id": w.account.ID},
		)
	}
}

// isIMAP 判断当前账号是否为 IMAP 协议
func (w *AccountWorker) isIMAP() bool {
	return w.account.Protocol != "pop3" && w.account.Protocol != "pop3-no-ssl"
}

// idleLoop 进入 IDLE 监听循环，等待服务器推送新邮件通知
// 
// go-imap/v2 的 IDLE 实现机制:
//   - Idle().Wait() 只在连接断开或手动 Close() 时返回
//   - 收到 EXISTS (新邮件) 不会让 Wait() 返回
//   - 必须通过 UnilateralDataHandler 回调检测新邮件到达
//
// 本函数采用"IDLE 短周期轮询"策略:
//   1. 启动 IDLE 并设置 Mailbox handler 监听 EXISTS 事件
//   2. 收到 EXISTS 后立即关闭 IDLE，返回主循环执行 syncOnce()
//   3. 如果 25 分钟无事件则自动重启 IDLE（IMAP 规定最长 29 分钟）
func (w *AccountWorker) idleLoop() error {
	// 创建带 UnilateralDataHandler 的 IMAP 客户端
	// 用于接收服务器的单方面数据推送 (EXISTS, FETCH, EXPUNGE 等)
	mailboxCh := make(chan struct{}, 1) // 非缓冲: 收到 EXISTS 时通知
	
	client, err := w.newIMAPClientWithHandler(mailboxCh)
	if err != nil {
		return err
	}
	defer client.Close()

	if err := client.Authenticate(); err != nil {
		return err
	}

	_, err = client.SelectINBOX()
	if err != nil {
		return err
	}

	// 进入 IDLE 模式
	log.Printf("🔄 IDLE 监听中: %s", w.account.Email)

	idleCmd, err := client.Client.Idle()
	if err != nil {
		return fmt.Errorf("IDLE 命令失败: %w", err)
	}

	// 等待以下任一事件:
	//   1. mailboxCh - 收到 EXISTS (新邮件到达)
	//   2. 25分钟超时 - IMAP 规定的安全重启间隔
	//   3. stopCh/shutdownCh - 停止信号
	select {
	case <-mailboxCh:
		// ⭐ 收到服务器推送的新邮件通知！
		idleCmd.Close()
		log.Printf("📬 IDLE 收到新邮件通知: %s", w.account.Email)
		return nil // 返回主循环执行 syncOnce()

	case <-time.After(25 * time.Minute):
		// 超时保底（IMAP 规定最长29分钟）
		idleCmd.Close()
		log.Printf("⏰ IDLE 超时重启: %s", w.account.Email)
		return nil // 返回主循环重新进入 IDLE

	case <-w.stopCh:
		idleCmd.Close()
		log.Printf("⏹️  IDLE 停止信号: %s", w.account.Email)
		return nil
		
	case <-w.shutdownCh:
		idleCmd.Close()
		return nil
	}
}

// newIMAPClientWithHandler 创建带有 UnilateralDataHandler 的 IMAP 客户端
// 用于 IDLE 模式下接收服务器的实时推送
func (w *AccountWorker) newIMAPClientWithHandler(mailboxCh chan struct{}) (*IMAPClient, error) {
	host := w.account.ImapHost
	port := w.account.Port
	addr := fmt.Sprintf("%s:%d", host, port)

	// TLS 配置（复用 client.go 中的配置逻辑）
	tlsConfig := &tls.Config{
		ServerName: host,
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		},
	}

	var imapClient *imapclient.Client
	var err error

	// 直连方式创建客户端（与 NewIMAPClient 保持一致）
	imapClient, err = imapclient.DialTLS(addr, &imapclient.Options{
		TLSConfig: tlsConfig,
		UnilateralDataHandler: &imapclient.UnilateralDataHandler{
			// ⭐ 关键: 监听 Mailbox 状态变化 (EXISTS/EXPUNGE 等)
			Mailbox: func(data *imapclient.UnilateralDataMailbox) {
				if data.NumMessages != nil {
					// 收件箱邮件数量变化 → 新邮件到达或删除
					// 向 channel 发送信号（非阻塞，防止重复触发）
					select {
					case mailboxCh <- struct{}{}:
						log.Printf("📥 [%s] 检测到邮箱状态变更 (当前 %d 封)", 
							w.account.Email, *data.NumMessages)
					default:
						// 已有待处理的通知，忽略
					}
				}
			},
			// 可选: 监听新邮件的详细数据
			Fetch: func(msg *imapclient.FetchMessageData) {
				// 通常 EXISTS 之后会跟随 FETCH 数据
				// 这里可以进一步处理邮件详情
				log.Printf("📧 [%s] 收到 FETCH 推送 (seq=%d)", 
					w.account.Email, msg.SeqNum)
			},
		},
	})
	
	if err != nil {
		return nil, fmt.Errorf("连接 %s 失败: %w", addr, err)
	}

	return &IMAPClient{
		Client:  imapClient,
		Account: w.account,
		config:  w.config,
	}, nil
}

// Stop 停止此 Worker
func (w *AccountWorker) Stop() {
	select {
	case w.stopCh <- struct{}{}:
	default:
	}
}

// updateAccountStatus 更新账号状态到数据库
func (w *AccountWorker) updateAccountStatus(status, errMsg string) {
	w.db.Model(&models.MailAccount{}).Where("id = ?", w.account.ID).
		Updates(map[string]interface{}{
			"status":    status,
			"error_msg": errMsg,
		})
	if status == "error" {
		log.Printf("❌ 同步错误 (%s): %s", w.account.Email, errMsg)
	}
}

// isIdleClosed 判断错误是否为 IDLE 正常结束
func isIdleClosed(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "idle terminated") ||
		contains(msg, "connection closed") ||
		contains(msg, "EOF")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
