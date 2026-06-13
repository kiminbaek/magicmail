// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

import request from './request'

// 获取邮箱列表
export function getAccounts(params = {}) {
  return request.get('/accounts', { params })
}

// 获取单个邮箱详情
export function getAccountById(id) {
  return request.get(`/accounts/${id}`)
}

// 创建邮箱
export function createAccount(data) {
  return request.post('/accounts', data)
}

// 更新邮箱
export function updateAccount(id, data) {
  return request.put(`/accounts/${id}`, data)
}

// 删除邮箱
export function deleteAccount(id) {
  return request.delete(`/accounts/${id}`)
}

// 测试连接
export function testConnection(data) {
  return request.post('/accounts/test-connection', data)
}

// 触发同步
export function triggerSync(id) {
  return request.post(`/accounts/${id}/sync`)
}

// 切换账号状态（停用/启用）
export function toggleAccountStatus(id, status) {
  return request.put(`/accounts/${id}/status`, { status })
}

