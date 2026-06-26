// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)
// Copyright (C) 2026  kiminbaek — QQ notification feature

import request from './request'

/** 获取 QQ 通知配置 */
export function getQQConfig() {
  return request.get('/qq-notification/config')
}

/** 保存 QQ 通知配置 */
export function saveQQConfig(data) {
  return request.post('/qq-notification/config', data)
}

/** 发送测试消息 */
export function testQQNotification() {
  return request.post('/qq-notification/test')
}

/** 获取发送日志 */
export function getQQLogs(limit = 50) {
  return request.get('/qq-notification/logs', { params: { limit } })
}

/** 删除日志 */
export function deleteQQLog(id) {
  return request.delete(`/qq-notification/logs/${id}`)
}
