// Magicmail Service Worker
// 处理 Web Push 推送事件 + Workbox 预缓存

import { precacheAndRoute } from 'workbox-precaching'
import { registerRoute } from 'workbox-routing'
import { StaleWhileRevalidate, CacheFirst } from 'workbox-strategies'
import { ExpirationPlugin } from 'workbox-expiration'

// --- 预缓存声明（由 vite-plugin-pwa injectManifest 注入） ---
precacheAndRoute(self.__WB_MANIFEST)

// --- Push 事件：收到服务端推送时显示系统通知 ---
self.addEventListener('push', (event) => {
  let data = { title: 'Magicmail', body: '您有新消息', icon: '/icons/icon-192x192.png' }

  if (event.data) {
    try {
      data = event.data.json()
    } catch (_) {
      data.body = event.data.text()
    }
  }

  const options = {
    body: data.body || '您有新消息',
    icon: data.icon || '/icons/icon-192x192.png',
    badge: '/icons/icon-72x72.png',
    vibrate: [200, 100, 200],
    tag: data.tag || 'magicmail-mail',
    requireInteraction: false,
    data: data.data || {},
  }

  event.waitUntil(
    self.registration.showNotification(data.title || 'Magicmail', options)
  )
})

// --- Notification Click：点击通知聚焦或打开应用窗口 ---
self.addEventListener('notificationclick', (event) => {
  event.notification.close()

  // 聚焦已有窗口，或打开新窗口
  event.waitUntil(
    self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then((clientList) => {
      // 如果已有打开的窗口，聚焦到它
      for (const client of clientList) {
        if (client.url.includes('/mails') && 'focus' in client) {
          return client.focus()
        }
      }
      // 没有邮件列表窗口则打开
      if (self.clients.openWindow) {
        return self.clients.openWindow('/mails')
      }
    })
  )
})

// --- 运行时缓存策略 ---

// API 缓存（排除 SSE 流）
registerRoute(
  ({ url }) => url.pathname.startsWith('/api') && !url.pathname.includes('/mails/stream'),
  new StaleWhileRevalidate({
    cacheName: 'api-cache',
    plugins: [
      new ExpirationPlugin({ maxEntries: 100, maxAgeSeconds: 5 * 60 }),
    ],
  })
)

// 图片缓存
registerRoute(
  ({ request }) => request.destination === 'image',
  new CacheFirst({
    cacheName: 'image-cache',
    plugins: [
      new ExpirationPlugin({ maxEntries: 50, maxAgeSeconds: 30 * 24 * 60 * 60 }),
    ],
  })
)
