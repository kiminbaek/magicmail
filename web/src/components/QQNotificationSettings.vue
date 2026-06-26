<!--
  SPDX-License-Identifier: AGPL-3.0-or-later
  Copyright (C) 2026  magiccode (魔法代码) — original MagicMail project
  Copyright (C) 2026  kiminbaek — QQ notification feature

  QQ 邮件通知配置组件
  通过 QwenPaw API 推送新邮件到达通知到 QQ
-->
<template>
  <section class="qq-settings card">
    <!-- 标题栏 -->
    <div class="qq-header">
      <div class="qq-header-info">
        <h3 class="qq-title">QQ 邮件通知</h3>
        <p class="qq-desc">新邮件到达时，通过 QwenPaw 自动推送 QQ 消息通知</p>
      </div>
      <div class="qq-header-actions">
        <label class="qq-toggle" :class="{ 'qq-toggle-on': form.enabled }">
          <input type="checkbox" v-model="form.enabled" />
          <span class="qq-toggle-slider"></span>
        </label>
        <span class="qq-toggle-label">{{ form.enabled ? '已启用' : '已禁用' }}</span>
      </div>
    </div>

    <!-- 加载中 -->
    <div v-if="loading" class="qq-loading">加载配置中…</div>

    <!-- 错误提示 -->
    <div v-else-if="loadError" class="qq-error-banner">
      ⚠️ {{ loadError }}
      <button class="qq-retry" @click="loadConfig">重试</button>
    </div>

    <!-- 配置表单 -->
    <div v-else class="qq-form">
      <!-- 连接配置 -->
      <div class="qq-form-group-title">QwenPaw 连接</div>
      <div class="qq-form-row">
        <div class="qq-field">
          <label>API 地址</label>
          <input type="text" v-model="form.qwenpaw_url" placeholder="http://127.0.0.1:19091"
                 :disabled="!form.enabled" />
          <small>QwenPaw 服务地址，默认本机</small>
        </div>
        <div class="qq-field">
          <label>Agent ID</label>
          <input type="text" v-model="form.agent_id" placeholder="003"
                 :disabled="!form.enabled" />
          <small>发送消息的 Agent 编号</small>
        </div>
      </div>
      <div class="qq-form-row">
        <div class="qq-field">
          <label>目标用户 ID</label>
          <input type="text" v-model="form.target_user" placeholder="用户唯一标识"
                 :disabled="!form.enabled" />
          <small>QwenPaw 用户的 user_id</small>
        </div>
        <div class="qq-field">
          <label>目标会话 ID</label>
          <input type="text" v-model="form.target_session" placeholder="qq:用户标识"
                 :disabled="!form.enabled" />
          <small>格式 qq:&lt;user_id&gt;</small>
        </div>
      </div>

      <!-- 过滤规则 -->
      <div class="qq-form-group-title">过滤规则<span class="qq-hint">（可选，留空 = 不过滤）</span></div>
      <div class="qq-form-row">
        <div class="qq-field">
          <label>发件人白名单</label>
          <textarea v-model="form.filter_from" rows="2" :disabled="!form.enabled"
                    placeholder="example@qq.com, noreply@github.com"></textarea>
          <small>逗号分隔；仅这些发件人的邮件会通知</small>
        </div>
        <div class="qq-field">
          <label>主题关键词白名单</label>
          <textarea v-model="form.filter_subject" rows="2" :disabled="!form.enabled"
                    placeholder="验证码, 订单, 账单"></textarea>
          <small>逗号分隔；主题包含这些词才通知</small>
        </div>
      </div>
      <div class="qq-form-row">
        <div class="qq-field">
          <label>发件人黑名单</label>
          <textarea v-model="form.exclude_from" rows="2" :disabled="!form.enabled"
                    placeholder="newsletter@xx.com"></textarea>
          <small>逗号分隔；这些发件人的邮件不通知</small>
        </div>
        <div class="qq-field">
          <label>排除主题关键词</label>
          <textarea v-model="form.exclude_subject" rows="2" :disabled="!form.enabled"
                    placeholder="退订, unsubscribe"></textarea>
          <small>逗号分隔；主题含这些词不通知</small>
        </div>
      </div>

      <!-- 免打扰时段 -->
      <div class="qq-form-group-title">免打扰时段<span class="qq-hint">（可选）</span></div>
      <div class="qq-form-row qq-time-row">
        <div class="qq-field qq-field-sm">
          <label>开始</label>
          <input type="time" v-model="form.silent_start" :disabled="!form.enabled" />
        </div>
        <span class="qq-time-sep">至</span>
        <div class="qq-field qq-field-sm">
          <label>结束</label>
          <input type="time" v-model="form.silent_end" :disabled="!form.enabled" />
        </div>
        <small class="qq-time-hint">该时段内收到的邮件不推送 QQ 通知</small>
      </div>

      <!-- 消息模板 -->
      <div class="qq-form-group-title">消息模板</div>
      <div class="qq-field">
        <textarea v-model="form.template" rows="5" :disabled="!form.enabled"
                  placeholder="📧 新邮件&#10;发件人：{{.From}}&#10;主题：{{.Subject}}&#10;摘要：{{.Preview}}"></textarea>
        <small class="qq-template-hint" v-pre>
          可用变量：<code>{{.From}}</code> 发件人 ·
          <code>{{.Subject}}</code> 主题 ·
          <code>{{.Preview}}</code> 摘要 ·
          <code>{{.AccountEmail}}</code> 账号 ·
          <code>{{.Time}}</code> 时间
        </small>
      </div>
      <div class="qq-form-row">
        <div class="qq-field qq-field-sm">
          <label>摘要长度</label>
          <input type="number" v-model.number="form.preview_length" min="50" max="500" step="50"
                 :disabled="!form.enabled" />
          <small>邮件正文摘要截取字数（50-500）</small>
        </div>
      </div>

      <!-- 操作按钮 -->
      <div class="qq-actions">
        <button class="qq-btn qq-btn-primary" @click="saveConfig" :disabled="saving || !form.enabled">
          <span v-if="saving" class="qq-spinner"></span>
          {{ saving ? '保存中…' : '💾 保存配置' }}
        </button>
        <button class="qq-btn qq-btn-secondary" @click="sendTest" :disabled="testing || !form.enabled">
          <span v-if="testing" class="qq-spinner"></span>
          {{ testing ? '发送中…' : '🔔 发送测试' }}
        </button>
        <span v-if="actionMsg" class="qq-action-msg" :class="{ 'qq-action-ok': actionOk, 'qq-action-err': !actionOk }">
          {{ actionMsg }}
        </span>
      </div>

      <!-- 统计信息 -->
      <div v-if="config && config.total_sent >= 0" class="qq-stats">
        <span class="qq-stat">累计发送 {{ config.total_sent }} 条</span>
        <span v-if="config.last_sent_at" class="qq-stat">最近发送 {{ formatTime(config.last_sent_at) }}</span>
        <span v-if="config.last_error" class="qq-stat qq-stat-err" :title="config.last_error">⚠️ 有错误</span>
      </div>
    </div>

    <!-- 发送日志 -->
    <div class="qq-logs-section" v-if="!loading && !loadError">
      <div class="qq-logs-header" @click="showLogs = !showLogs">
        <span class="qq-logs-title">📋 发送日志</span>
        <span class="qq-logs-toggle">{{ showLogs ? '▾ 收起' : '▸ 展开' }}</span>
      </div>
      <div v-if="showLogs" class="qq-logs-body">
        <div v-if="logs.length === 0" class="qq-logs-empty">暂无发送记录</div>
        <table v-else class="qq-logs-table">
          <thead>
            <tr>
              <th>时间</th>
              <th>发件人</th>
              <th>主题</th>
              <th>状态</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in logs" :key="log.id" :class="{ 'qq-log-fail': log.status !== 'success' }">
              <td class="qq-log-time">{{ formatTime(log.created_at) }}</td>
              <td class="qq-log-from">{{ log.mail_from || '—' }}</td>
              <td class="qq-log-subject" :title="log.mail_subject">{{ truncate(log.mail_subject, 30) }}</td>
              <td>
                <span class="qq-badge" :class="log.status === 'success' ? 'qq-badge-ok' : 'qq-badge-err'">
                  {{ log.status === 'success' ? '✓ 成功' : '✗ 失败' }}
                </span>
                <span v-if="log.error_msg" class="qq-log-err" :title="log.error_msg">⚠</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { getQQConfig, saveQQConfig, testQQNotification, getQQLogs } from '../api/qq_notification'

const loading = ref(true)
const loadError = ref('')
const saving = ref(false)
const testing = ref(false)
const actionMsg = ref('')
const actionOk = ref(false)
const showLogs = ref(false)
const config = ref(null)
const logs = ref([])

const form = reactive({
  enabled: false,
  qwenpaw_url: 'http://127.0.0.1:19091',
  agent_id: '',
  target_user: '',
  target_session: '',
  filter_from: '',
  filter_subject: '',
  exclude_from: '',
  exclude_subject: '',
  silent_start: '',
  silent_end: '',
  template: '📧 新邮件\n发件人：{{.From}}\n主题：{{.Subject}}\n摘要：{{.Preview}}',
  preview_length: 200,
})

function applyConfig(c) {
  config.value = c
  form.enabled = c.enabled || false
  form.qwenpaw_url = c.qwenpaw_url || 'http://127.0.0.1:19091'
  form.agent_id = c.agent_id || ''
  form.target_user = c.target_user || ''
  form.target_session = c.target_session || ''
  form.filter_from = c.filter_from || ''
  form.filter_subject = c.filter_subject || ''
  form.exclude_from = c.exclude_from || ''
  form.exclude_subject = c.exclude_subject || ''
  form.silent_start = c.silent_start || ''
  form.silent_end = c.silent_end || ''
  form.template = c.template || '📧 新邮件\n发件人：{{.From}}\n主题：{{.Subject}}\n摘要：{{.Preview}}'
  form.preview_length = c.preview_length || 200
}

async function loadConfig() {
  loading.value = true
  loadError.value = ''
  try {
    const c = await getQQConfig()
    applyConfig(c || {})
    await loadLogs()
  } catch (e) {
    loadError.value = e.message || '加载配置失败'
  } finally {
    loading.value = false
  }
}

async function loadLogs() {
  try {
    const data = await getQQLogs(20)
    logs.value = Array.isArray(data) ? data : (data?.logs || [])
  } catch {
    // 日志加载失败不影响配置
  }
}

function showMsg(msg, ok) {
  actionMsg.value = msg
  actionOk.value = ok
  setTimeout(() => { actionMsg.value = '' }, 4000)
}

async function saveConfig() {
  saving.value = true
  actionMsg.value = ''
  try {
    const c = await saveQQConfig({ ...form })
    if (c?.config) applyConfig(c.config)
    showMsg('✓ 配置已保存', true)
  } catch (e) {
    showMsg('✗ 保存失败：' + (e.message || '未知错误'), false)
  } finally {
    saving.value = false
  }
}

async function sendTest() {
  testing.value = true
  actionMsg.value = ''
  try {
    await testQQNotification()
    showMsg('✓ 测试消息已发送，请检查 QQ', true)
    await loadLogs()
  } catch (e) {
    showMsg('✗ 发送失败：' + (e.message || '未知错误'), false)
  } finally {
    testing.value = false
  }
}

function truncate(s, n) {
  if (!s) return ''
  return s.length > n ? s.slice(0, n) + '…' : s
}

function formatTime(t) {
  if (!t) return ''
  const d = new Date(t)
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

onMounted(loadConfig)
</script>

<style scoped>
.qq-settings {
  background: #fff;
  border-radius: 6px;
  padding: 20px 24px;
  margin-bottom: 16px;
}

/* 标题栏 */
.qq-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}
.qq-title {
  font-size: 15px;
  font-weight: 600;
  color: #1a1a1a;
  margin: 0 0 4px;
}
.qq-desc {
  font-size: 13px;
  color: #888;
  margin: 0;
}
.qq-header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Toggle 开关 */
.qq-toggle {
  position: relative;
  display: inline-block;
  width: 40px;
  height: 22px;
  cursor: pointer;
}
.qq-toggle input {
  opacity: 0;
  width: 0;
  height: 0;
}
.qq-toggle-slider {
  position: absolute;
  inset: 0;
  background: #d4d4d4;
  border-radius: 11px;
  transition: 0.2s;
}
.qq-toggle-slider::before {
  content: '';
  position: absolute;
  width: 18px;
  height: 18px;
  left: 2px;
  top: 2px;
  background: #fff;
  border-radius: 50%;
  transition: 0.2s;
  box-shadow: 0 1px 2px rgba(0,0,0,0.15);
}
.qq-toggle-on .qq-toggle-slider {
  background: #4a90d9;
}
.qq-toggle-on .qq-toggle-slider::before {
  transform: translateX(18px);
}
.qq-toggle-label {
  font-size: 13px;
  color: #888;
  min-width: 40px;
}

/* 表单 */
.qq-form {
  border-top: 1px solid #f0f0f0;
  padding-top: 16px;
}
.qq-form-group-title {
  font-size: 13px;
  font-weight: 600;
  color: #555;
  margin: 16px 0 10px;
  padding-bottom: 6px;
  border-bottom: 1px solid #f5f5f5;
}
.qq-form-group-title:first-child {
  margin-top: 0;
}
.qq-hint {
  font-weight: 400;
  color: #aaa;
  font-size: 12px;
  margin-left: 6px;
}
.qq-form-row {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
}
.qq-field {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.qq-field label {
  font-size: 13px;
  color: #444;
  margin-bottom: 4px;
  font-weight: 500;
}
.qq-field input,
.qq-field textarea {
  border: 1px solid #ddd;
  border-radius: 4px;
  padding: 7px 10px;
  font-size: 13px;
  color: #333;
  background: #fff;
  transition: border-color 0.15s;
  font-family: inherit;
  resize: vertical;
}
.qq-field input:focus,
.qq-field textarea:focus {
  outline: none;
  border-color: #4a90d9;
}
.qq-field input:disabled,
.qq-field textarea:disabled {
  background: #f7f7f7;
  color: #aaa;
}
.qq-field small {
  font-size: 12px;
  color: #999;
  margin-top: 3px;
}
.qq-field-sm {
  max-width: 200px;
}
.qq-time-row {
  align-items: flex-end;
}
.qq-time-sep {
  font-size: 13px;
  color: #888;
  padding-bottom: 8px;
}
.qq-time-hint {
  font-size: 12px;
  color: #999;
  padding-bottom: 8px;
}
.qq-template-hint code {
  background: #f0f0f0;
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 11px;
  color: #4a90d9;
}

/* 按钮 */
.qq-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #f0f0f0;
}
.qq-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 16px;
  border: none;
  border-radius: 4px;
  font-size: 13px;
  cursor: pointer;
  transition: opacity 0.15s;
}
.qq-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.qq-btn-primary {
  background: #4a90d9;
  color: #fff;
}
.qq-btn-secondary {
  background: #f0f0f0;
  color: #555;
}
.qq-action-msg {
  font-size: 13px;
}
.qq-action-ok { color: #52c41a; }
.qq-action-err { color: #f5222d; }

/* 统计 */
.qq-stats {
  display: flex;
  gap: 16px;
  margin-top: 12px;
  font-size: 12px;
  color: #999;
}
.qq-stat-err { color: #f5222d; }

/* 日志 */
.qq-logs-section {
  margin-top: 20px;
  border-top: 1px solid #f0f0f0;
  padding-top: 14px;
}
.qq-logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  cursor: pointer;
  user-select: none;
}
.qq-logs-title {
  font-size: 13px;
  font-weight: 600;
  color: #555;
}
.qq-logs-toggle {
  font-size: 12px;
  color: #4a90d9;
}
.qq-logs-body {
  margin-top: 10px;
}
.qq-logs-empty {
  font-size: 13px;
  color: #aaa;
  padding: 12px 0;
}
.qq-logs-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
.qq-logs-table th {
  text-align: left;
  padding: 6px 8px;
  color: #888;
  font-weight: 500;
  border-bottom: 1px solid #eee;
}
.qq-logs-table td {
  padding: 6px 8px;
  color: #444;
  border-bottom: 1px solid #f8f8f8;
}
.qq-log-time { color: #999; white-space: nowrap; }
.qq-log-from { max-width: 160px; overflow: hidden; text-overflow: ellipsis; }
.qq-log-subject { max-width: 240px; overflow: hidden; text-overflow: ellipsis; }
.qq-log-fail td { background: #fff5f5; }
.qq-log-err { color: #f5222d; margin-left: 4px; }
.qq-badge {
  display: inline-block;
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
}
.qq-badge-ok { background: #f0f9eb; color: #52c41a; }
.qq-badge-err { background: #fff5f5; color: #f5222d; }

/* 加载/错误 */
.qq-loading {
  padding: 20px;
  text-align: center;
  color: #999;
  font-size: 13px;
}
.qq-error-banner {
  padding: 12px 16px;
  background: #fff5f5;
  border-radius: 4px;
  color: #f5222d;
  font-size: 13px;
  display: flex;
  align-items: center;
  gap: 12px;
}
.qq-retry {
  margin-left: auto;
  padding: 4px 12px;
  border: 1px solid #f5222d;
  border-radius: 4px;
  background: #fff;
  color: #f5222d;
  cursor: pointer;
  font-size: 12px;
}

/* Spinner */
.qq-spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255,255,255,0.4);
  border-top-color: #fff;
  border-radius: 50%;
  animation: qq-spin 0.6s linear infinite;
}
.qq-btn-secondary .qq-spinner {
  border-color: rgba(85,85,85,0.3);
  border-top-color: #555;
}
@keyframes qq-spin {
  to { transform: rotate(360deg); }
}
</style>
