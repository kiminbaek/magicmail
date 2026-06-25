// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package imap

import (
	"crypto/tls"
	"fmt"
	"log"

	"magicmail/config"
	"magicmail/models"
	pop3pkg "magicmail/pop3"
	"magicmail/proxy"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
)

// MailClient 统一邮件客户端接口（IMAP / POP3 共用）
type MailClient interface {
	Authenticate() error
	Close()
}

// IMAPClient 封装 go-imap/v2 客户端连接，提供连接/认证/关闭等基础操作
type IMAPClient struct {
	Client  *imapclient.Client // 底层 IMAP 客户端
	Account *models.MailAccount
	config  *config.Config
}

// NewIMAPClient 创建新的 IMAP 邮件连接实例
func NewIMAPClient(account *models.MailAccount, cfg *config.Config) (*IMAPClient, error) {
	host := account.ImapHost
	port := account.Port
	addr := fmt.Sprintf("%s:%d", host, port)

	// 构建 TLS 配置（默认使用 TLS）
	// 注意：Go 1.24+ 默认不再包含 AES-CBC 等旧版密码套件，
	// 国内邮箱服务器（新浪/163等）可能仍需这些套件，此处显式指定以确保兼容。
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

	// 获取自定义 Dialer（代理）
	customDialer, err := proxy.Dialer(account.ProxyEnabled, account.ProxyURL)
	if err != nil {
		return nil, fmt.Errorf("代理配置错误: %w", err)
	}

	var client *imapclient.Client
	if customDialer != nil {
		// 通过代理建立 TCP 连接，再包装 TLS
		conn, err := customDialer("tcp", addr)
		if err != nil {
			return nil, fmt.Errorf("通过代理连接 %s 失败: %w", addr, err)
		}
		tlsConn := tls.Client(conn, tlsConfig)
		if err := tlsConn.Handshake(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("TLS 握手失败 (%s): %w", addr, err)
		}
		client = imapclient.New(tlsConn, &imapclient.Options{})
	} else {
		// 直连
		client, err = imapclient.DialTLS(addr, &imapclient.Options{
			TLSConfig: tlsConfig,
		})
		if err != nil {
			return nil, fmt.Errorf("连接 %s 失败: %w", addr, err)
		}
	}

	return &IMAPClient{
		Client:  client,
		Account: account,
		config:  cfg,
	}, nil
}

// NewMailClient 根据协议类型创建对应的邮件客户端（IMAP / POP3）
func NewMailClient(account *models.MailAccount, cfg *config.Config) (MailClient, error) {
	switch account.Protocol {
	case "pop3", "pop3-no-ssl":
		return pop3pkg.NewPOP3Client(account, cfg)
	default:
		return NewIMAPClient(account, cfg)
	}
}

// Authenticate 使用 LOGIN 命令认证，并发送客户端ID信息（解决163邮箱 Unsafe Login问题）
func (c *IMAPClient) Authenticate() error {
	if c.Account.Password == "" {
		return fmt.Errorf("密码为空，无法认证")
	}

	// 发送 ID 命令声明客户端身份（RFC 2971）
	// 163/126等网易邮箱要求客户端必须发送ID命令，否则会返回 "SELECT Unsafe Login" 错误
	idData := &imap.IDData{
		Name:    "MagicMail",
		Version: "1.0.0",
		Vendor:  "MagicCode",
	}
	if _, err := c.Client.ID(idData).Wait(); err != nil {
		// ID 命令失败不阻塞登录（部分服务器可能不支持），仅记录日志
		log.Printf("⚠️  发送 IMAP ID 命令失败 (可能服务器不支持): %v", err)
	}

	if err := c.Client.Login(c.Account.Username, c.Account.Password).Wait(); err != nil {
		return fmt.Errorf("IMAP 登录失败 (%s@%s): %w", c.Account.Username, c.Account.Email, err)
	}
	log.Printf("✅ IMAP 认证成功: %s", c.Account.Email)
	return nil
}

// SelectINBOX 选择收件箱并返回状态信息
func (c *IMAPClient) SelectINBOX() (*imap.SelectData, error) {
	return c.SelectMailbox("INBOX")
}

// SelectMailbox 选择指定邮箱（如 INBOX, Sent 等）并返回状态信息
func (c *IMAPClient) SelectMailbox(name string) (*imap.SelectData, error) {
	mbox, err := c.Client.Select(name, nil).Wait()
	if err != nil {
		return nil, fmt.Errorf("选择 %s 失败: %w", name, err)
	}
	return mbox, nil
}

// Close 关闭连接
func (c *IMAPClient) Close() {
	if c.Client != nil {
		if err := c.Client.Logout().Wait(); err != nil {
			log.Printf("⚠️  IMAP 连接关闭异常 (%s): %v", c.Account.Email, err)
		}
	}
}

// DeleteMessage 通过 UID 删除服务器上的邮件（Store + \Deleted 标志 → Expunge/UID Expunge）
func (c *IMAPClient) DeleteMessage(uid uint32) error {
	// 先选择 INBOX（使用读写模式）
	selectData, err := c.SelectINBOX()
	if err != nil {
		return fmt.Errorf("选择 INBOX 失败: %w", err)
	}

	// 检查服务器是否允许设置删除标志（\Deleted 是否在 PermanentFlags 中）
	canDelete := false
	for _, f := range selectData.PermanentFlags {
		if f == imap.FlagDeleted || f == imap.FlagWildcard {
			canDelete = true
			break
		}
	}
	if !canDelete {
		return fmt.Errorf("INBOX 不支持删除操作（PermanentFlags 中无 \\Deleted）")
	}

	// 设置 \Deleted 标志
	uidSet := imap.UIDSetNum(imap.UID(uid))
	storeCmd := c.Client.Store(uidSet, &imap.StoreFlags{
		Op:    imap.StoreFlagsAdd,
		Flags: []imap.Flag{imap.FlagDeleted},
	}, nil)
	if err := storeCmd.Close(); err != nil {
		return fmt.Errorf("标记删除标志失败 (UID=%d): %w", uid, err)
	}

	// 优先尝试 UID EXPUNGE (RFC 4315)，更精确地只删除指定 UID 的邮件
	// 如果服务器不支持，则回退到普通 EXPUNGE
	uidExpungeErr := c.tryUIDExpunge(uid)
	if uidExpungeErr == nil {
		log.Printf("🗑️  已通过 UID EXPUNGE 从源服务器删除邮件 (UID=%d, %s)", uid, c.Account.Email)
		return nil
	}

	log.Printf("[INFO] UID EXPUNGE 不可用，尝试普通 EXPUNGE (UID=%d): %v", uid, uidExpungeErr)

	// 回退：执行普通 Expunge 永久删除所有已标记邮件
	expungeCmd := c.Client.Expunge()
	if err := expungeCmd.Close(); err != nil {
		return fmt.Errorf("Expunge 失败 (UID=%d): %w", uid, err)
	}

	log.Printf("🗑️  已通过 EXPUNGE 从源服务器删除邮件 (UID=%d, %s)", uid, c.Account.Email)
	return nil
}

// tryUIDExpunge 尝试使用 UID EXPUNGE 命令（RFC 4315）
// 只删除指定 UID 的邮件，不影响其他已标记删除的邮件
func (c *IMAPClient) tryUIDExpunge(uid uint32) error {
	uidSet := imap.UIDSetNum(imap.UID(uid))
	uidExpungeCmd := c.Client.UIDExpunge(uidSet)
	if err := uidExpungeCmd.Close(); err != nil {
		return err
	}
	return nil
}

// TestConnection 测试邮箱账号的连接是否可用（根据协议自动选择）
func TestConnection(account *models.MailAccount, cfg *config.Config) error {
	client, err := NewMailClient(account, cfg)
	if err != nil {
		return err
	}
	defer client.Close()

	// POP3 认证后额外验证邮件列表
	if pc, ok := client.(*pop3pkg.POP3Client); ok {
		if err := pc.Authenticate(); err != nil {
			return err
		}
		_, err = pc.MessageCount()
		return err
	}

	// IMAP：认证 + 选择 INBOX
	if ic, ok := client.(*IMAPClient); ok {
		if err := ic.Authenticate(); err != nil {
			return err
		}
		_, err = ic.SelectINBOX()
		return err
	}

	return fmt.Errorf("未知的协议类型: %s", account.Protocol)
}
