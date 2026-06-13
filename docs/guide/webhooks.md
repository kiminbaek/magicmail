# Webhook 通知

Webhook 允许在新邮件到达时自动推送 HTTP 通知到外部服务，适用于：

- 即时消息推送（钉钉、飞书、企业微信、Slack）
- 自动化工流集成（Zapier、n8n、Make）
- 自定义业务逻辑处理

## 创建 Webhook

1. 进入 **设置中心** → **Webhook 管理**
2. 点击 **新建 Webhook**
3. 配置以下参数：

| 字段 | 说明 |
|------|------|
| 名称 | Webhook 显示名称 |
| URL | 回调地址（需公网可达） |
| 自定义 Header | 额外的 HTTP 请求头（如 Authorization）|
| Body 模板 | 请求体模板，支持变量替换 |

## 变量模板

Body 模板中可以使用以下变量：

| 变量 | 说明 |
|------|------|
| <span v-pre>`{{event}}`</span> | 事件名称（如 `mail.received`） |
| <span v-pre>`{{timestamp}}`</span> | 触发时间（Unix 时间戳） |
| <span v-pre>`{{data.account_id}}`</span> | 邮箱账号 ID |
| <span v-pre>`{{data.account_email}}`</span> | 邮箱地址 |
| <span v-pre>`{{data.account_name}}`</span> | 邮箱显示名称 |
| <span v-pre>`{{data.protocol}}`</span> | 协议类型（imap / pop3） |
| <span v-pre>`{{data.mail_count}}`</span> | 本次新邮件数量 |
| <span v-pre>`{{data.mails}}`</span> | 邮件列表 JSON 数组，每项包含 `subject`、`from`、`sent_at`、`preview` |

> 注意：邮件详情以 JSON 数组形式嵌套在 `data.mails` 中，单条邮件的字段为 `subject`（主题）、`from`（发件人）、`sent_at`（发送时间）、`preview`（正文预览前 100 字）。模板中无法直接使用扁平的 <span v-pre>`{{subject}}`</span> 等变量。

### 示例：[魔法推送](https://github.com/magiccode1412/magicpush)

```json
{
  "title": "📧 新邮件通知 - {{data.account_name}}",
  "content": "## 📧 收到 {{data.mail_count}} 封新邮件\n\n**来源：** {{data.account_name}} <{{data.account_email}}>\n**时间：** {{data.timestamp}}\n\n### 邮件列表\n\n{{data.mails}}",
  "type": "markdown"
}
```

### 示例：飞书机器人

```json
{
  "msg_type": "text",
  "content": {
    "text": "📧 收到 {{data.mail_count}} 封新邮件\n账号：{{data.account_name}} <{{data.account_email}}>\n邮件列表：{{data.mails}}"
  }
}
```

### 示例：钉钉机器人

```json
{
  "msgtype": "markdown",
  "markdown": {
    "title": "📧 新邮件 - {{data.account_name}}",
    "text": "### 📧 收到 {{data.mail_count}} 封新邮件\n> **账号：** {{data.account_name}} <{{data.account_email}}>\n> **时间：** {{data.timestamp}}\n\n#### 邮件列表\n{{data.mails}}"
  }
}
```

## 测试与管理

- **测试推送**：发送模拟通知验证配置是否生效
- **查看日志**：查看每次推送的状态码和响应内容
- **启用/禁用**：临时开关某个 Webhook
- **删除**：移除不再需要的 Webhook

::: warning 公网访问
Webhook URL 必须是公网可达的地址。本地开发可使用 ngrok、frp 等内网穿透工具暴露服务。
:::
