# 已知问题

记录用户反馈的问题及处理进度。

## 状态说明

| 状态 | 说明 |
|------|------|
| 🔄 未修复 | 问题已确认，待处理 |
| ✅ 已修复未发布 | 代码已修复，等待下一个版本发布 |
| 🎉 已修复 | 已在正式版本中发布 |

---

## 问题列表

### 1. 邮箱管理页面中等宽度下信息与按钮重叠

- **状态**：✅ 已修复未发布
- **反馈时间**：2026-06-26
- **问题描述**：在 768px ~ 900px 宽度区间，邮箱管理页面的桌面端 Grid 布局会导致邮箱地址信息与右侧操作按钮（编辑、同步、删除）发生重叠，影响使用体验。
- **修复方案**：将 `AccountManage.vue` 的响应式断点从 `768px` 调整为 `900px`，使中等屏幕更早切换到卡片布局。
- **涉及文件**：`web/src/views/AccountManage.vue`

### 2. 163/126 网易邮箱登录失败 (Unsafe Login)

- **状态**：✅ 已修复未发布
- **反馈时间**：2026-06-25
- **问题描述**：使用 163、126 等网易邮箱时，IMAP 登录返回 "SELECT Unsafe Login" 错误，导致无法正常收信。原因是网易邮箱要求客户端在登录前必须发送 ID 命令声明身份（符合 RFC 2971 规范）。
- **修复方案**：在 IMAP 登录前主动发送 ID 命令，声明客户端信息（Name: MagicMail, Version: 1.0.0, Vendor: MagicCode）。若服务器不支持 ID 命令则仅记录日志，不阻塞登录流程。
- **涉及文件**：`server/imap/client.go`

### 3. 密码解密失败导致账号列表查询异常

- **状态**：✅ 已修复未发布
- **反馈时间**：2026-06-26
- **问题描述**：当某个邮箱账号的密码加密数据损坏或密钥不匹配时，`AfterFind` 钩子中的解密失败会阻断整个账号列表查询接口（500 错误），导致用户无法查看任何账号。
- **修复方案**：
  1. 解密失败时仅记录警告日志并清空密码字段，不再返回错误阻断查询；
  2. 新增 `AccountListDTO` 专用列表查询模型，避免列表场景触发 `AfterFind` 解密逻辑；
  3. 新增账号健康检查接口 `/api/v1/accounts/health`，便于排查异常账号。
- **涉及文件**：`server/models/mail_account.go`, `server/services/account_service.go`, `server/handlers/account_handler.go`, `server/services/health_check_service.go`, `web/src/stores/accountStore.js`
