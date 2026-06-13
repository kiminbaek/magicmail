// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2026  magiccode (魔法代码)

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getMails, getMailById, markAsRead as apiMarkRead, deleteMail as apiDeleteMail, batchDeleteMails as apiBatchDelete, getMailStats } from '@/api/mail'

export const useMailStore = defineStore('mail', () => {
  // --- 状态 ---
  const mails = ref([])
  const currentMail = ref(null)
  const total = ref(0)
  const currentPage = ref(1)
  const pageSize = ref(20)
  
  // 筛选条件
  const filters = ref({
    account_id: '',
    folder: '',
    keyword: '',
    is_read: null,
    has_attachment: null,
    sort_by: 'sent_at',
    sort_order: 'desc'
  })

  const loading = ref(false)
  const error = ref(null)
  const stats = ref({})

  // --- 计算属性 ---
  const hasMore = computed(() => mails.value.length < total.value)

  // --- 加载邮件列表 ---
  async function fetchMails(page = 1) {
    loading.value = true
    error.value = null

    try {
      const params = {
        page,
        page_size: pageSize.value,
        ...filters.value
      }

      // 清理空值
      Object.keys(params).forEach(key => {
        if (params[key] === '' || params[key] === null || params[key] === undefined) {
          delete params[key]
        }
      })

      const res = await getMails(params)
      mails.value = res.data || []
      total.value = res.total || 0
      currentPage.value = res.page || page
    } catch (e) {
      error.value = e.message
      console.error('[mailStore] 获取邮件列表失败:', e.message)
    } finally {
      loading.value = false
    }
  }

  // --- 加载邮件详情 ---
  async function fetchMailDetail(id) {
    try {
      const mail = await getMailById(id)
      currentMail.value = mail
      
      // 自动标记已读（如果未读）
      if (!mail.is_read) {
        markAsRead(id, true)
      }
      
      return mail
    } catch (e) {
      console.error('[mailStore] 获取邮件详情失败:', e.message)
      throw e
    }
  }

  // --- 标记已读/未读 ---
  async function markAsRead(id, isRead) {
    try {
      await apiMarkRead(id, isRead)
      // 更新本地状态
      const idx = mails.value.findIndex(m => m.id === id)
      if (idx !== -1) {
        mails.value[idx].is_read = isRead
      }
      if (currentMail.value?.id === id) {
        currentMail.value.is_read = isRead
      }
    } catch (e) {
      console.error('[mailStore] 标记已读失败:', e.message)
    }
  }

  // --- 删除邮件 ---
  async function deleteMail(id) {
    try {
      const res = await apiDeleteMail(id)
      mails.value = mails.value.filter(m => m.id !== id)
      total.value--
      return res
    } catch (e) {
      console.error('[mailStore] 删除邮件失败:', e.message)
      throw e
    }
  }

  // --- 批量删除邮件 ---
  async function batchDeleteMails(ids) {
    try {
      const res = await apiBatchDelete(ids)
      const idSet = new Set(ids)
      mails.value = mails.value.filter(m => !idSet.has(m.id))
      total.value -= res.deleted || ids.length
      return res
    } catch (e) {
      console.error('[mailStore] 批量删除失败:', e.message)
      throw e
    }
  }

  // --- 更新筛选条件并刷新 ---
  function setFilter(key, value) {
    filters.value[key] = value
    return fetchMails(1)
  }

  // --- 重置筛选 ---
  function resetFilters() {
    filters.value = {
      account_id: '',
      folder: '',
      keyword: '',
      is_read: null,
      has_attachment: null,
      sort_by: 'sent_at',
      sort_order: 'desc'
    }
    return fetchMails(1)
  }

  // --- 获取统计信息 ---
  async function fetchStats(accountId) {
    try {
      const params = accountId ? { account_id: accountId } : {}
      const res = await getMailStats(params)
      stats.value = res || {}
      return res
    } catch (e) {
      console.error('[mailStore] 获取统计失败:', e.message)
      return {}
    }
  }

  return {
    mails, currentMail, total, currentPage, pageSize,
    filters, loading, error, hasMore, stats,
    fetchMails, fetchMailDetail, markAsRead, deleteMail, batchDeleteMails,
    setFilter, resetFilters, fetchStats,
  }
})
