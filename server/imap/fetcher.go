// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

package imap

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"magicmail/config"
	"magicmail/models"

	"gorm.io/gorm"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
)

// Fetcher 邮件拉取器 - 负责从 IMAP 服务器拉取邮件并解析存储
type Fetcher struct {
	db            *gorm.DB
	config        *config.Config
	folder        string // 当前同步的文件夹名（inbox/sent）
	SyncedMailIDs []uint // 本次同步成功入库的邮件ID列表（精确追踪，用于webhook）
}

// NewFetcher 创建拉取器实例
func NewFetcher(db *gorm.DB, cfg *config.Config) *Fetcher {
	return &Fetcher{
		db:     db,
		config: cfg,
	}
}

// SyncMailbox 同步指定邮箱账号的 INBOX，返回新增/更新的邮件数量
func (f *Fetcher) SyncMailbox(client *IMAPClient) (int, error) {
	return f.syncMailbox(client, "INBOX", "inbox")
}

// SyncSentMailbox 同步已发送文件夹（Sent），返回新增/更新的邮件数量
func (f *Fetcher) SyncSentMailbox(client *IMAPClient) (int, error) {
	return f.syncMailbox(client, "Sent", "sent")
}

// syncMailbox 同步指定邮箱账号的指定 IMAP 文件夹
func (f *Fetcher) syncMailbox(client *IMAPClient, mailboxName, folder string) (int, error) {
	f.folder = folder
	// 注意：不在此处重置 SyncedMailIDs，由调用方（worker）在创建 Fetcher 后统一管理
	// 因为同一 Fetcher 可能被多次调用（INBOX + Sent），需要累积所有ID

	mbox, err := client.SelectMailbox(mailboxName)
	if err != nil {
		// Sent 文件夹可能不存在或无权限，静默跳过不报错
		if folder == "sent" {
			log.Printf("⚠️  %s 的 %s 文件夹不可用，跳过同步: %v", client.Account.Email, mailboxName, err)
			return 0, nil
		}
		return 0, err
	}

	if mbox.NumMessages == 0 {
		log.Printf("📭 收件箱为空: %s", client.Account.Email)
		return 0, nil
	}

	// 构建序列集：获取所有邮件（使用显式范围而非 1:* 避免部分 IMAP 兼容性问题）
	seqSet := imap.SeqSet{}
	seqSet.AddRange(1, mbox.NumMessages) // 显式指定 1 到 NumMessages

	// 定义获取选项：信封 + UID + 标志 + 大小 + 内部日期（用于日期过滤）
	fetchOptions := &imap.FetchOptions{
		Envelope:    true,
		UID:         true,
		Flags:       true,
		RFC822Size:  true,
		InternalDate: true,
	}

	fetchCmd := client.Client.Fetch(seqSet, fetchOptions)

	newCount := 0
	syncMode := client.Account.SyncMode
	syncDays := client.Account.SyncDays
	if syncDays <= 0 {
		syncDays = 30 // 默认30天
	}
	cutoffTime := time.Now().AddDate(0, 0, -syncDays)

	log.Printf("📬 开始同步 %s: 模式=%s, 天数=%d, 收件箱共 %d 封邮件",
		client.Account.Email, syncMode, syncDays, mbox.NumMessages)

	// ⭐ 第一阶段：拉取所有信封数据（不在此阶段调 fetchBody，避免 IMAP 命令 pipeline 冲突）
	// 原因：同一连接上信封 FETCH 进行中时发第二个 FETCH，163 等服务器不支持 pipeline 会不响应
	var envelopes []*imapclient.FetchMessageBuffer
	for {
		msg := fetchCmd.Next()
		if msg == nil {
			break
		}
		buf, err := msg.Collect()
		if err != nil {
			log.Printf("⚠️  收集消息数据失败: %v", err)
			continue
		}
		envelopes = append(envelopes, buf)
	}
	fetchCmd.Close() // 显式关闭信封 FETCH，确保命令完成后再发下一个 FETCH

	log.Printf("📬 信封拉取完成 %s: 共 %d 封，开始拉取正文", client.Account.Email, len(envelopes))

	// ⭐ 第二阶段：逐个拉取邮件正文（在信封 FETCH Close 后执行，避免命令冲突）
	for _, buf := range envelopes {
		// 根据 SyncMode 判断是否需要同步这封邮件
		if !shouldSync(buf, syncMode, cutoffTime) {
			continue
		}

		parsed, _, err := f.parseMessage(client, buf)
		if err != nil {
			log.Printf("⚠️  解析邮件失败 (UID=%d): %v", buf.UID, err)
			// 连接超时说明 IMAP 连接已废，提前终止同步避免每封等 60 秒
			if strings.Contains(err.Error(), "连接已断开") {
				log.Printf("🔌 IMAP 连接已断开，终止本次同步 %s", client.Account.Email)
				return newCount, err
			}
			continue
		}
		if parsed == nil {
			continue // 已存在（去重跳过）
		}

		// ⭐ 记录本次成功入库的邮件ID（用于精确触发 webhook）
		f.SyncedMailIDs = append(f.SyncedMailIDs, parsed.ID)
		log.Printf("✅ [%s] 入库邮件 ID=%d, UID=%d, subject=%q", f.folder, parsed.ID, buf.UID, parsed.Subject)

		// 邮件和附件已在 parseMessage 中统一入库，无需额外操作
		newCount++
	}

	log.Printf("📬 同步完成 %s (模式=%s): 新增/更新 %d 封邮件, IDs=%v", client.Account.Email, syncMode, newCount, f.SyncedMailIDs)
	return newCount, nil
}

// parseMessage 解析单封邮件，返回解析后的 Mail 对象；若已存在则返回 nil（去重）
func (f *Fetcher) parseMessage(client *IMAPClient, buf *imapclient.FetchMessageBuffer) (*models.Mail, *BodyResult, error) {
	envelope := buf.Envelope
	if envelope == nil {
		return nil, nil, fmt.Errorf("邮件信封为空")
	}

	messageID := envelope.MessageID
	if messageID == "" {
		// 无 Message-ID 时用 UID+时间戳生成唯一标识
		messageID = fmt.Sprintf("<auto-%d-%s@proxy>", buf.UID, time.Now().Format("20060102150405"))
	}

	// 去重：根据 Message-ID + AccountID 判断是否已存在
	var existing int64
	f.db.Model(&models.Mail{}).
		Where("message_id = ? AND account_id = ?", messageID, client.Account.ID).
		Count(&existing)
	if existing > 0 {
		return nil, nil, nil // 已存在，跳过
	}

	// 获取发件人（v2 的 Address 是值类型）
	fromAddr := extractIMAPAddressList(envelope.From)
	toAddr := extractIMAPAddressList(envelope.To)
	ccAddr := extractIMAPAddressList(envelope.Cc)

	subject := decodeHeader(envelope.Subject)

	sentAt := envelope.Date
	if sentAt.IsZero() {
		sentAt = time.Now()
	}

	// 判断已读/标星状态
	isRead := false
	isStarred := false
	for _, flag := range buf.Flags {
		if flag == imap.FlagSeen {
			isRead = true
		}
		if flag == imap.FlagFlagged {
			isStarred = true
		}
	}

	// 构建邮件对象（先不包含正文和附件信息）
	mailObj := &models.Mail{
		AccountID:  client.Account.ID,
		Folder:     f.folder,
		MessageID:  messageID,
		MessageUID: uint32(buf.UID),
		From:       fromAddr,
		To:         toAddr,
		Cc:         ccAddr,
		Subject:    subject,
		SentAt:     sentAt,
		IsRead:     isRead,
		IsStarred:  isStarred,
		Size:       buf.RFC822Size,
		CreatedAt:  time.Now(),
	}

	// ⭐ 先入库以获取 mailID（大附件流式写入需要）
	if err := f.db.Create(mailObj).Error; err != nil {
		return nil, nil, fmt.Errorf("创建邮件记录失败: %w", err)
	}

	// 拉取完整邮件体（正文 + 附件），传入 mailID 支持大附件流式写入
	bodySection, err := f.fetchBody(client, buf.UID, mailObj.ID)
	if err != nil {
		log.Printf("⚠️  拉取邮件体失败 (UID=%d), 删除不完整邮件记录 (mail_id=%d): %v", buf.UID, mailObj.ID, err)
		// ⭐ 关键修复：删除没有正文/附件的不完整邮件记录，避免后续因去重而无法重新同步
		f.db.Delete(mailObj)
		return nil, nil, err
	}
	if bodySection == nil {
		log.Printf("⚠️  邮件体返回空 (UID=%d), 删除不完整邮件记录 (mail_id=%d)", buf.UID, mailObj.ID)
		f.db.Delete(mailObj)
		return nil, nil, fmt.Errorf("邮件体为空")
	}

	// 更新邮件的正文信息
	updateData := map[string]interface{}{}
	if bodySection.TextBody != "" {
		updateData["text_body"] = bodySection.TextBody
	}
	if bodySection.HTMLBody != "" {
		updateData["html_body"] = bodySection.HTMLBody
	}
	if len(bodySection.Attachments) > 0 {
		updateData["has_attachment"] = true
	}
	if len(updateData) > 0 {
		if uErr := f.db.Model(mailObj).Updates(updateData).Error; uErr != nil {
			log.Printf("⚠️  更新邮件正文失败 (mail_id=%d): %v", mailObj.ID, uErr)
		}
	} else {
		// ⭐ 调试日志：记录无内容的邮件，帮助定位问题
		log.Printf("🔍 [DEBUG] 邮件体解析完成但无正文/附件 (mail_id=%d, UID=%d, subject=%q)",
			mailObj.ID, buf.UID, mailObj.Subject)
	}

	// ⭐ 保存所有附件到数据库（包括懒加载模式的元数据）
	for i := range bodySection.Attachments {
		att := &bodySection.Attachments[i]
		att.MailID = mailObj.ID

		if attErr := f.db.Create(att).Error; attErr != nil {
			log.Printf("⚠️  保存附件记录失败 (mail_id=%d, file=%s): %v",
				mailObj.ID, att.Filename, attErr)
			// 清理可能已创建的文件
			if att.FilePath != "" {
				os.Remove(att.FilePath)
			}
		}
	}

	return mailObj, nil, nil
}

// BodyResult 邮件体解析结果
type BodyResult struct {
	TextBody    string
	HTMLBody    string
	Attachments []models.Attachment
}

// fetchBody 获取邮件的完整正文和附件
// mailID 用于大附件流式写入（传入 0 表示尚未入库）
func (f *Fetcher) fetchBody(client *IMAPClient, uid imap.UID, mailID uint) (*BodyResult, error) {
	// 使用 UIDSet 获取单封邮件完整内容
	uidSet := imap.UIDSetNum(uid)

	// 获取 BODY[] 完整原始邮件 + BODYSTRUCTURE（用于获取各 Part 精确大小）
	bodySection := &imap.FetchItemBodySection{}
	fetchOptions := &imap.FetchOptions{
		BodySection:   []*imap.FetchItemBodySection{bodySection},
		BodyStructure: &imap.FetchItemBodyStructure{},
	}

	fetchCmd := client.Client.Fetch(uidSet, fetchOptions)
	defer fetchCmd.Close()

	// 使用 Collect 获取结果（带 60 秒超时保护，防止连接断开后永久卡死）
	// 背景：go-imap/v2 beta.4 在连接被服务端断开后，Collect() 不返回错误也不超时，
	// goroutine 永久卡在 channel 等待。此处用 goroutine + select 兜底。
	type collectResult struct {
		msgs []*imapclient.FetchMessageBuffer
		err  error
	}
	ch := make(chan collectResult, 1)
	go func() {
		m, e := fetchCmd.Collect()
		ch <- collectResult{m, e}
	}()

	var msgs []*imapclient.FetchMessageBuffer
	var err error
	select {
	case r := <-ch:
		msgs, err = r.msgs, r.err
	case <-time.After(60 * time.Second):
		log.Printf("⚠️  拉取邮件正文超时 (UID=%d)，IMAP 连接可能已断开", uid)
		fetchCmd.Close()
		return nil, fmt.Errorf("拉取邮件正文超时 (UID=%d)，连接已断开", uid)
	}
	if err != nil {
		return nil, fmt.Errorf("获取邮件体失败: %w", err)
	}
	if len(msgs) == 0 {
		return nil, fmt.Errorf("无消息返回")
	}

	msg := msgs[0]
	// BodySection 是 map[*FetchItemBodySection][]byte
	if len(msg.BodySection) == 0 {
		return nil, fmt.Errorf("邮件体为空")
	}
	// 取第一个 body section 的数据
	var bodyBytes []byte
	for _, data := range msg.BodySection {
		bodyBytes = data
		break
	}
	if len(bodyBytes) == 0 {
		return nil, fmt.Errorf("邮件体为空")
	}

	result := &BodyResult{}

	// ⭐ 从 BODYSTRUCTURE 提取每个 MIME Part 的精确大小（key: "1", "1.1", "1.2" 等）
	partSizes := make(map[string]int64)
	if msg.BodyStructure != nil {
		msg.BodyStructure.Walk(func(path []int, part imap.BodyStructure) bool {
			// 构造 PartID: [1] -> "1", [1, 2] -> "1.2"
			partID := ""
			for i, idx := range path {
				if i > 0 {
					partID += "."
				}
				partID += fmt.Sprintf("%d", idx)
			}
			// 只处理单部分（附件/文本），跳过 multipart
			if singlePart, ok := part.(*imap.BodyStructureSinglePart); ok && partID != "" {
				partSizes[partID] = int64(singlePart.Size)
				log.Printf("🔍 [DEBUG] BODYSTRUCTURE Part=%s, Size=%d, Type=%s/%s",
					partID, singlePart.Size, singlePart.Type, singlePart.Subtype)
			}
			return true // 继续遍历子部分
		})
		log.Printf("🔍 [DEBUG] 从 BODYSTRUCTURE 获取到 %d 个 Part 大小信息", len(partSizes))
	}

	// 准备附件存储目录（仅当 mailID 可用时）
	// 使用 DSN 同级目录下的 attachments 子目录，而非相对路径（避免工作目录不对导致权限错误）
	var baseDir string
	if mailID > 0 {
		dsnDir := filepath.Dir(f.config.Database.DSN)
		if !filepath.IsAbs(dsnDir) {
			absDir, err := filepath.Abs(dsnDir)
			if err == nil {
				dsnDir = absDir
			}
		}
		baseDir = filepath.Join(dsnDir, "attachments")
		if err := os.MkdirAll(baseDir, 0755); err != nil {
			log.Printf("⚠️  创建附件目录失败: %v", err)
			baseDir = ""
		}
	}

	// 使用 go-message 库解析 MIME 结构
	entity, err := message.Read(bytes.NewReader(bodyBytes))
	if err != nil {
		// 解析失败则作为纯文本处理
		result.TextBody = decodeTextContent(bodyBytes)
		return result, nil
	}

	// 检查是否为 multipart
	if mr := entity.MultipartReader(); mr != nil {
		// 多部分邮件：从 PartID "1" 开始递归
		partIdx := 1
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("⚠️  读取 MIME 部分失败: %v", err)
				break
			}
			f.parseEntity(part, result, mailID, baseDir, uint32(uid), fmt.Sprintf("%d", partIdx), partSizes)
			partIdx++
		}
	} else {
		// 单部分邮件：PartID 为 "1"
		f.parseEntity(entity, result, mailID, baseDir, uint32(uid), "1", partSizes)
	}

	// ⭐ 调试日志：记录解析结果摘要
	log.Printf("🔍 [DEBUG] MIME解析完成 (UID=%d): text_len=%d, html_len=%d, attachments=%d",
		uid, len(result.TextBody), len(result.HTMLBody), len(result.Attachments))

	return result, nil
}

// parseEntity 解析单个 MIME 实体（文本或附件）
// 支持嵌套的 multipart 结构（如 multipart/mixed > multipart/alternative > text/plain）
// mailID 和 baseDir 用于大附件的流式写入（>5MB 直接写磁盘）
// imapUID 和 partID 用于懒加载模式（大附件只存元数据，按需下载）
// partSizes 是从 BODYSTRUCTURE 获取的各 Part 精确大小（key: "1", "1.2" 等）
func (f *Fetcher) parseEntity(entity *message.Entity, result *BodyResult, mailID uint, baseDir string, imapUID uint32, partID string, partSizes map[string]int64) {
	mediaType, params, _ := entity.Header.ContentType()

	// ⭐ 递归处理嵌套的 multipart 结构
	if strings.HasPrefix(mediaType, "multipart/") {
		if mr := entity.MultipartReader(); mr != nil {
			subIdx := 1
			for {
				part, err := mr.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					log.Printf("⚠️  读取嵌套 MIME 部分失败: %v", err)
					break
				}
				// 子部分的 PartID = 父PartID.子索引
				subPartID := fmt.Sprintf("%s.%d", partID, subIdx)
				f.parseEntity(part, result, mailID, baseDir, imapUID, subPartID, partSizes)
				subIdx++
			}
		}
		return
	}

	// 处理 Content-Disposition 判断是否为附件
	contentDisposition, dispParams, _ := entity.Header.ContentDisposition()
	isAttachment := contentDisposition == "attachment" ||
		(contentDisposition == "inline" && params["name"] != "")

	if isAttachment {
		filename := dispParams["filename"]
		if filename == "" {
			filename = params["name"]
		}
		decodedFilename := decodeRFC2047Filename(filename)

		// ⭐ 优先从 BODYSTRUCTURE 获取精确大小，回退到 Content-Length
		var estimatedSize int64
		if bsSize, ok := partSizes[partID]; ok && bsSize > 0 {
			estimatedSize = bsSize
			log.Printf("🔍 [DEBUG] 使用 BODYSTRUCTURE 大小: Part=%s, Size=%d", partID, bsSize)
		} else {
			contentLength := entity.Header.Get("Content-Length")
			if contentLength != "" {
				fmt.Sscanf(contentLength, "%d", &estimatedSize)
			}
		}

		// ⭐ 根据配置决定缓存策略
		cacheThreshold := f.config.IMAP.GetCacheThreshold()
		maxSize := f.config.IMAP.GetMaxAttachmentSize()

		shouldLazyLoad := (estimatedSize >= cacheThreshold) ||
			(estimatedSize == 0 && mailID > 0) // 无法确定大小时也用懒加载

		if shouldLazyLoad && mailID > 0 {
			// ⭐ 懒加载模式：只保存元数据，不从 IMAP 下载内容
			finalSize := estimatedSize
			if finalSize <= 0 {
				// 无法获取精确大小时，标记为 -1 表示"未知"
				// 前端会显示"未知大小"或类似提示
				finalSize = -1
			}
			att := models.Attachment{
				MailID:      mailID,
				Filename:    decodedFilename,
				ContentType: mediaType,
				Size:        finalSize,
				IMAPUID:     imapUID,
				PartID:      partID,
				IsCached:    false,
				CreatedAt:   time.Now(),
			}
			result.Attachments = append(result.Attachments, att)
			log.Printf("📎 附件懒加载(不下载): %s [Part=%s, Size≈%d]", decodedFilename, partID, finalSize)
			return
		}

		// 小附件或无法懒加载时：正常下载并缓存
		if estimatedSize > maxSize && maxSize > 0 {
			log.Printf("⚠️ 附件超过最大限制 (%d > %d)，跳过: %s", estimatedSize, maxSize, decodedFilename)
			return
		}

		// ⭐ 磁盘空间预检查
		if estimatedSize > 0 && !f.checkDiskSpace(estimatedSize) {
			log.Printf("⚠️  磁盘空间不足，跳过附件: %s (需要 %d bytes)", decodedFilename, estimatedSize)
			return
		}

		// 判断是否需要流式写入磁盘（>5MB 或无法确定大小但 baseDir 可用）
		shouldStream := (estimatedSize > models.MaxDBSize) ||
			(estimatedSize == 0 && mailID > 0 && baseDir != "")

		if shouldStream && mailID > 0 && baseDir != "" {
			// 流式写入磁盘 - 完全不占用内存存储完整内容
			fileName := fmt.Sprintf("%d_%s", mailID, decodedFilename)
			filePath := filepath.Join(baseDir, fileName)

			if outFile, err := os.Create(filePath); err == nil {
				written, copyErr := io.Copy(outFile, entity.Body)
				outFile.Close() // 立即关闭文件句柄

				if copyErr != nil {
					log.Printf("⚠️  写入大附件文件失败: %v", copyErr)
					os.Remove(filePath)
					return
				}

				cacheExpire := time.Now().Add(f.config.IMAP.GetCacheExpireDuration())
				att := models.Attachment{
					MailID:      mailID,
					Filename:    decodedFilename,
					ContentType: mediaType,
					Size:        written,
					FilePath:    filePath,
					IMAPUID:     imapUID,
					PartID:      partID,
					IsCached:    true,
					CacheExpire: &cacheExpire,
					CreatedAt:   time.Now(),
				}
				result.Attachments = append(result.Attachments, att)
				log.Printf("📎 大附件已缓存到本地: %s (%d bytes) [Part=%s]", decodedFilename, written, partID)
			} else {
				log.Printf("⚠️  创建附件文件失败: %v", err)
			}
		} else {
			// 小附件或无 mailID 时：读入内存并存 DB BLOB
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, entity.Body); err == nil {
				att := models.Attachment{
					MailID:      mailID,
					Filename:    decodedFilename,
					ContentType: mediaType,
					Size:        int64(buf.Len()),
					Content:     buf.Bytes(),
					IMAPUID:     imapUID,
					PartID:      partID,
					IsCached:    true, // DB BLOB 视为已缓存
					CreatedAt:   time.Now(),
				}
				result.Attachments = append(result.Attachments, att)
			}
		}
		return
	}

	// 处理文本内容
	switch {
	case strings.HasPrefix(mediaType, "text/plain"):
		textData, _ := io.ReadAll(entity.Body)
		result.TextBody = decodeTextContent(textData)
	case strings.HasPrefix(mediaType, "text/html"):
		htmlData, _ := io.ReadAll(entity.Body)
		result.HTMLBody = decodeTextContent(htmlData)
	}
}

// checkDiskSpace 检查磁盘剩余空间是否足够
func (f *Fetcher) checkDiskSpace(requiredBytes int64) bool {
	freeBytes, err := getDiskFreeSpace()
	if err != nil {
		log.Printf("⚠️  获取磁盘信息失败: %v", err)
		return true // 无法获取时默认允许
	}

	minFree := f.config.IMAP.GetMinDiskFree()

	// 剩余空间必须 > 最小保留空间 + 本次写入所需空间
	return freeBytes > (minFree + requiredBytes)
}

// --- 工具函数 ---

// extractAddressList 从 go-message/mail 地址列表中提取格式化的地址字符串
func extractAddressList(addrs []*mail.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if addr.Name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", addr.Name, addr.Address))
		} else {
			parts = append(parts, addr.Address)
		}
	}
	return strings.Join(parts, ", ")
}

// extractIMAPAddressList 从 go-imap/v2 地址列表中提取格式化的地址字符串
func extractIMAPAddressList(addrs []imap.Address) string {
	if len(addrs) == 0 {
		return ""
	}
	parts := make([]string, 0, len(addrs))
	for _, addr := range addrs {
		if addr.Name != "" {
			parts = append(parts, fmt.Sprintf("%s <%s>", addr.Name, addr.Addr()))
		} else {
			parts = append(parts, addr.Addr())
		}
	}
	return strings.Join(parts, ", ")
}

// decodeHeader 解码 RFC 2047 编码的头部字段
func decodeHeader(raw string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(raw)
	if err != nil {
		return raw
	}
	return decoded
}

// decodeRFC2047Filename 解码 RFC 2047 编码的文件名
func decodeRFC2047Filename(raw string) string {
	dec := new(mime.WordDecoder)
	decoded, err := dec.DecodeHeader(raw)
	if err != nil {
		return raw
	}
	return decoded
}

// decodeTextContent 自动检测字符集并解码文本内容
func decodeTextContent(data []byte) string {
	str := string(data)

	// 尝试 UTF-8
	if isUTF8(str) {
		return str
	}

	// 尝试 GBK/GB18030
	if decoded, err := decodeCharset(data, "gbk"); err == nil {
		return decoded
	}

	// 尝试 ISO-8859-1
	if decoded, err := decodeCharset(data, "iso-8859-1"); err == nil {
		return decoded
	}

	return str
}

// isUTF8 简单检查字符串是否为有效 UTF-8
func isUTF8(s string) bool {
	for _, r := range s {
		if r == '\ufffd' {
			return false
		}
	}
	return true
}

// decodeCharset 简单字符集解码（需要 golang.org/x/text 支持）
func decodeCharset(data []byte, charset string) (string, error) {
	return "", fmt.Errorf("charset '%s' not supported without golang.org/x/text", charset)
}

// shouldSync 根据 SyncMode 判断是否需要同步该邮件
// syncMode: unread(只同步未读), all(全部), recent(最近N天)
// cutoffTime: 最近N天的截止时间
func shouldSync(buf *imapclient.FetchMessageBuffer, syncMode string, cutoffTime time.Time) bool {
	switch syncMode {
	case "unread":
		// 只同步未读邮件（没有 Seen 标志）
		for _, flag := range buf.Flags {
			if flag == imap.FlagSeen {
				return false // 已读，跳过
			}
		}
		return true // 未读，同步
	case "recent":
		// 同步最近 N 天的邮件
		if !buf.InternalDate.IsZero() && buf.InternalDate.Before(cutoffTime) {
			return false // 太早了，跳过
		}
		return true
	default:
		// "all" 或其他未知值，全部同步
		return true
	}
}
