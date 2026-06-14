<div align="center">

<img src="./public/images/icon_512.png" alt="logo" width="128" height="128">

# Magicmail - 魔法邮箱

<a href="https://160621.xyz/magicmail" target="_blank">使用文档</a> | <a href="https://github.com/magiccode1412/magicmail" target="_blank">GitHub</a>

一套完整的邮件代收系统，基于 **Go (Fiber + GORM + SQLite)** 后端 + **Vue3 PWA** 前端。通过 IMAP 协议代理收取多个邮箱账号的邮件，统一存储至本地数据库，以现代化 PWA 客户端呈现。

</div>

## 交流&打赏

<table>
  <tr>
    <td align="center">
      <a href="https://qm.qq.com/q/wWS78gByRa">点此加入QQ群</a>
      <br>
      <img src="./public/images/qq-group.jpg" alt="qq-group" height="256px">
    </td>
    <td align="center">
      <a href="https://pd.qq.com/s/eveskv89x">点此加入QQ频道</a>
      <br>
      <img src="./public/images/qq-channel.jpg" alt="qq-channel" height="256px">
    </td>
    <td align="center">
      <a href="https://pd.qq.com/s/eveskv89x">支付宝</a>
      <br>
      <img src="./public/images/alipay.png" alt="qq-channel" height="256px">
    </td>
    <td align="center">
      <a href="https://pd.qq.com/s/eveskv89x">微信</a>
      <br>
      <img src="./public/images/wechat.png" alt="qq-channel" height="256px">
    </td>
  </tr>
</table>

## 功能特性

### 后端服务
- **IMAP 收信**：集成 `go-imap/v2`，支持 TLS 连接、认证、全量/增量同步
- **IMAP IDLE 实时监听**：后台常驻协程运行 IMAP IDLE 推送新邮件通知
- **POP3 收信**：支持 POP3 协议收取邮件（自动降级模式）
- **SMTP 发信**：通过 SMTP 协议发送邮件，支持 HTML 正文、附件、抄送/密送
- **SSE 实时推送**：Server-Sent Events 服务端推送，新邮件 < 1 秒到达前端（含指数退避重连）
- **Web Push 浏览器推送**：基于 VAPID (Web Push Protocol)，即使页面关闭也能收到新邮件通知
- **多邮箱管理**：支持增删改查多个 IMAP/POP3 邮箱账号，支持启用/停用、手动触发同步
- **自动去重**：基于 Message-ID 全局去重
- **邮件解析**：MIME multipart 解析、字符集转换、附件提取
- **混合附件缓存**：小文件立即缓存 + 大文件懒加载按需下载，可配置阈值和自动清理
- **HTTP/SOCKS5 代理**：支持按账号独立配置代理，IMAP/POP3 收信和 SMTP 发信均通过代理连接
- **草稿箱**：支持邮件草稿的保存、编辑、删除和批量管理
- **批量操作**：支持批量删除邮件和草稿
- **Webhook 通知**：新邮件到达时推送通知到外部服务（支持自定义 Header/Body）
- **RESTful API**：完整 CRUD 接口（认证 / 邮箱 / 邮件 / 草稿 / 附件 / Webhook / 推送订阅）
- **JWT 认证**：用户注册登录，接口鉴权保护
- **跨平台编译**：纯 Go SQLite 驱动，无需 CGO，支持交叉编译（含 Windows）

### 前端客户端
- **PWA 支持**：可安装到桌面/主屏幕、离线缓存访问
- **深色模式**：跟随系统自动切换浅色/深色主题
- **完全响应式**：手机、平板、PC 自适应布局
- **现代 UI**：玻璃态侧边栏、流畅动画、毛玻璃效果
- **设置中心**：主题色定制、Webhook 管理
- **SSE 实时更新**：邮件列表 < 1 秒自动刷新，断线指数退避重连
- **Web Push 通知**：浏览器原生推送通知，支持订阅/取消订阅管理
- **版本更新检测**：启动时自动检查新版本并提示用户

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端 | Go 1.21+, Fiber v2, GORM, modernc.org/sqlite (纯 Go, 无 CGO), go-imap/v2 |
| 前端 | Vue3 Composition API, Vite 5, Pinia, Vue Router |
| PWA | vite-plugin-pwa, Service Worker (Workbox) |
| 实时推送 | SSE (Server-Sent Events), Web Push (VAPID / Web Push Protocol) |
| 样式 | 原生 CSS + CSS 变量主题系统 |

## 项目结构

```
/workspace/
├── .github/                # GitHub Actions CI/CD
│   └── workflows/
│       └── docker-publish.yml   # 多架构 Docker 镜像构建 + 推送至 Docker Hub
├── .gitignore              # Git 忽略规则
├── .env.example            # Docker 环境变量模板
├── build.sh                # 生产构建脚本（前端+后端+嵌入）
├── dev.sh                  # 开发环境启动脚本
├── deploy.sh               # 一键部署脚本（安装/更新/卸载，支持 magicmail 命令）
├── docker-compose.yml      # Docker Compose 编排（含健康检查、资源限制、日志轮转）
├── bin/                    # 编译产物输出目录
│   └── magicmail (.exe)      # 单文件二进制（已嵌入前端）
│
├── server/                 # 后端 (Go)
│   ├── main.go             # 入口文件
│   ├── go.mod              # 模块依赖
│   ├── config/             # 配置管理
│   ├── crypto/             # 加密/解密工具 (AES-256-GCM, VAPID)
│   ├── database/           # 数据库初始化与自动迁移
│   ├── embedfs/            //go:embed 嵌入的前端产物
│   │   └── dist/           # 前端构建输出（编译时复制至此）
│   ├── models/             # GORM 数据模型
│   ├── imap/               # IMAP 连接/拉取/Worker (IDLE + 轮询)
│   ├── pop3/               # POP3 协议支持
│   ├── smtp/               # SMTP 发信支持
│   ├── proxy/              # HTTP CONNECT / SOCKS5 代理客户端
│   ├── sse/                # SSE 实时推送 Broker
│   ├── handlers/           # API 处理器 (含 Mail/Draft/Push Handler)
│   ├── services/           # 业务逻辑层 (含 PushService/DraftService)
│   ├── routes/             # 路由注册 (含 VAPID 密钥初始化)
│   ├── middleware/         # 中间件 (CORS, Auth)
│   ├── notifier/           # Webhook 通知引擎 + Push 推送回调
│   └── magicmail.service     # systemd 服务配置（可选）
│
├── web/                    # 前端 (Vue3)
│   ├── index.html
│   ├── package.json        # pnpm 依赖管理
│   ├── vite.config.js      # Vite + PWA 配置
│   ├── public/             # 静态资源 / PWA 图标
│   └── src/
│       ├── main.js         # 应用入口
│       ├── App.vue         # 根组件
│       ├── router/         # Vue Router
│       ├── api/            # Axios API 封装
│       ├── stores/         # Pinia 状态管理
│       ├── views/          # 页面组件
│       ├── components/     # 通用组件
│       ├── composables/    # 组合式函数 (useSSE / useWebPush / useUpdateCheck)
│       └── styles/         # CSS 变量/主题/全局样式
│
├── fnapp/                  # 飞牛 fnOS 应用配置
├── docs/                   # VitePress 文档站
├── README.md               # 本文档
└── LICENSE                 # AGPL-3.0 开源协议
```

## 快速开始

### 环境要求

- **Go** >= 1.21
- **Node.js** >= 18
- **pnpm** (推荐) 或 npm

---

### 方式一：一键部署（推荐用于服务器）

> 最简单的部署方式，自动完成依赖安装、二进制下载、服务注册、CLI 命令配置。支持 Linux 各发行版 / macOS / Docker。

```bash
# 方式 A：原始地址（国外 / 有代理环境）
curl -fsSL https://raw.githubusercontent.com/magiccode1412/magicmail/main/deploy.sh -o magicmail.sh

# 方式 B：jsDelivr CDN 镜像（国内推荐，速度快）
curl -fsSL https://cdn.jsdelivr.net/gh/magiccode1412/magicmail@main/deploy.sh -o magicmail.sh

chmod +x magicmail.sh && sudo ./magicmail.sh install

# 安装指定版本
sudo ./magicmail.sh install --version v1.0.0

# 非交互模式（跳过所有确认）
sudo ./magicmail.sh install -y
```

安装完成后，可通过全局命令 `magicmail` 管理服务：

```bash
magicmail status       # 查看运行状态
magicmail start        # 启动服务
magicmail stop         # 停止服务
magicmail restart      # 重启服务
magicmail logs         # 查看日志（默认100行）
magicmail logs 50      # 查看最近50行日志
magicmail update       # 更新到最新版本
magicmail version      # 版本信息（含远程版本对比）
magicmail doctor       # 环境健康自检
magicmail uninstall    # 卸载程序

# 查看帮助
magicmail help
./magicmail.sh help
```

#### 支持的平台与特性

| 平台 | 包管理器 | 服务管理 | 自动安装依赖 |
|------|----------|----------|:------------:|
| Ubuntu / Debian / Linux Mint | apt | systemd | ✅ |
| CentOS / RHEL / Rocky / Alma | dnf / yum | systemd | ✅ |
| Arch Linux / Manjaro | pacman | systemd | ✅ |
| Alpine Linux | apk | systemd | ✅ |
| OpenSUSE / SLES | zypper | systemd | ✅ |
| macOS (Intel & Apple Silicon) | brew | LaunchDaemon | ✅ |
| Docker 容器 | — | nohup 后台模式 | — |

**部署脚本自动处理：**
- 系统环境检测（OS / 发行版 / CPU 架构 / Docker 环境）
- 资源预检（磁盘空间 >= 200MB / 内存 >= 256MB）
- 端口占用检测 + 防火墙自动提示（ufw / firewalld / iptables）
- 缺失依赖自动安装（curl / wget / tar / gzip）
- **GitHub 镜像自动切换** — 直连失败时依次尝试 jsDelivr / ghproxy / ghfast 等 5 个国内加速镜像
- GitHub Release 自动下载对应平台二进制
- systemd (Linux) 或 LaunchDaemon (macOS) 服务注册 + 开机自启
- `magicmail` 全局 CLI 命令注册到 `/usr/local/bin`
- 安装完成汇总输出 + 自动打开浏览器

---

### 方式二：Docker Compose（容器环境）

```bash
# 1. 复制配置模板（可选，修改端口/时区/资源限制）
cp .env.example .env

# 2. 构建并启动
docker compose up -d --build

# 3. 查看日志
docker compose logs -f

# 停止: docker compose down    # 重启: docker compose restart
```

数据自动持久化到 `./docker-data/` 目录。详见 [安装部署 > Docker](https://160621.xyz/magicmail/guide/installation)。

> **预构建镜像**：也可直接拉取多架构预构建镜像（amd64 + arm64）：
> ```bash
> # Docker Hub
> docker pull magiccode1412/magicmail:latest
> docker pull magiccode1412/magicmail:v1.2.3   # 指定版本
>
> # GitHub Container Registry (ghcr.io)
> docker pull ghcr.io/magiccode1412/magicmail:latest
> docker pull ghcr.io/magiccode1412/magicmail:v1.2.3
> ```

---

### 方式三：CI/CD 自动构建（Docker Hub）

项目配置了 GitHub Actions 工作流 (`.github/workflows/docker-publish.yml`)，推送 `v*` 标签时自动触发多架构镜像构建并推送至 Docker Hub：

| 项目 | 说明 |
|------|------|
| **触发条件** | 推送 `v*` 格式的 tag（如 `v1.2.3`），或手动触发 |
| **目标架构** | `linux/amd64` + `linux/arm64` |
| **镜像仓库 1** | `docker.io/magiccode1412/magicmail` (Docker Hub) |
| **镜像仓库 2** | `ghcr.io/magiccode1412/magicmail` (GitHub Container Registry) |
| **自动标签** | `v1.2.3` / `1.2.3` / `1.2` / `1` / `latest` |

**使用前需配置 Secrets（Docker Hub）：**

| Secret | 说明 | 必要性 |
|--------|------|--------|
| `DOCKERHUB_USERNAME` | Docker Hub 用户名 | 仅 Docker Hub |
| `DOCKERHUB_TOKEN` | Docker Hub Access Token（[生成地址](https://hub.docker.com/settings/security)）| 仅 Docker Hub |

> **GitHub Container Registry (ghcr.io)** 使用 `GITHUB_TOKEN` 自动认证，无需额外配置 Secret。

---

### 方式四：使用构建脚本（从源码编译）

```bash
# 构建当前平台版本
./build.sh

# 交叉编译 Windows 版本
./build.sh windows amd64

# 交叉编译 macOS Apple Silicon
./build.sh darwin arm64

# 清理构建产物
./build.sh clean
```

构建脚本会自动完成：**安装前端依赖 → 构建前端 → 嵌入到 Go 二进制 → 编译输出到 `bin/`**

---

### 方式四：Windows 部署

Windows 不支持一键部署脚本，推荐以下方式：

**方式 A — 直接运行二进制（最简）：**

1. 从 [GitHub Releases](https://github.com/magiccode1412/magicmail/releases) 下载 `magicmail-windows-amd64.exe`
2. 放入目标目录（如 `C:\magicmail\`），双击或 PowerShell 运行：
```powershell
.\magicmail-windows-amd64.exe
```

**方式 B — 注册为 Windows 服务（开机自启）：**

使用 [NSSM](https://nssm.cc/download) 注册系统服务：
```powershell
nssm install magicmail "C:\magicmail\magicmail-windows-amd64.exe"
nssm start magicmail
# 停止: nssm stop magicmail    卸载: nssm remove magicmail confirm
```

**方式 C — Docker Desktop：**
```powershell
docker pull magiccode1412/magicmail:latest
docker run -d -p 8080:8080 -v C:\magicmail\data:C:\data --name magicmail --restart unless-stopped magiccode1412/magicmail:latest
```

详见 [安装部署 > Windows](https://160621.xyz/magicmail/guide/installation#方式四windows-部署)

---

### 方式五：开发环境

```bash
# 一键启动开发环境（后端 + 前端热重载）
./dev.sh start

# 停止开发环境
./dev.sh stop
```

或手动分步启动：

```bash
# 终端 1：启动后端
cd server && go run .

# 终端 2：启动前端开发服务器（默认 :5173）
cd web && pnpm install && pnpm dev
```

Vite 开发服务器会自动代理 `/api` 请求到后端 `:8080`。

---

### 方式六：手动生产构建

```bash
# 1. 构建前端
cd web && pnpm build && cd ..

# 2. 编译后端（前端产物会嵌入二进制）
cd server && go build -o ../bin/magicmail .

# 3. 运行
../bin/magicmail
```

---

### 运行

后端默认监听 `http://localhost:8080`，API 基础路径 `/api/v1`。

服务启动后浏览器访问 `http://localhost:8080` 即可使用（静态文件由 Fiber 直接托管）。

#### 环境变量配置

所有变量均为可选，不设置则使用默认值。

| 变量 | 默认值 | 说明 |
|------|--------|------|
| **运行模式** | | |
| `MAGICMAIL_ENV` | 空（开发模式） | 设为 `production` 关闭 SQL 日志并启用 release 模式 |
| **服务配置** | | |
| `MAGICMAIL_PORT` | `8080` | HTTP 监听端口 |
| `MAGICMAIL_HOST` | `0.0.0.0` | HTTP 监听地址 |
| `MAGICMAIL_DSN` | `data/magicmail.db` | SQLite 数据库文件路径 |
| `MAGICMAIL_CORS_ORIGINS` | 允许所有来源 | 跨域白名单（逗号分隔，如 `https://example.com,https://app.example.com`）|
| **IMAP 同步** | | |
| `MAGICMAIL_POLL_INTERVAL` | `300`（5 分钟）| IMAP 定时轮询间隔（秒），最低 10 秒 |
| `MAGICMAIL_IDLE_ENABLED` | `true` | 是否启用 IMAP IDLE 实时推送（设为 `false` 或 `0` 关闭）|
| `MAGICMAIL_MAX_CONCURRENT` | `10` | IMAP 最大并发连接数 |
| `MAGICMAIL_SYNC_BATCH_SIZE` | `50` | 每次同步拉取邮件数量上限 |
| **附件缓存（混合模式）** | | |
| `MAGICMAIL_AUTO_CACHE` | `false` | 启用自动缓存（懒加载首次下载后缓存到本地），设为 `true` 或 `1` 开启 |
| `MAGICMAIL_CACHE_THRESHOLD` | `2` (MB) | 附件缓存阈值，小于此值的附件同步时立即缓存，大文件懒加载按需下载 |
| `MAGICMAIL_MIN_DISK_FREE` | `1024` (MB) | 最小保留磁盘空间，不足时停止缓存 |
| `MAGICMAIL_MAX_ATTACHMENT_SIZE` | `50` (MB) | 单附件大小上限，超过此值的附件将被跳过 |
| `MAGICMAIL_CACHE_EXPIRE_DAYS` | `30` (天) | 缓存过期时间，超期的缓存将被自动清理 |

> 💡 安全密钥（`MAGICMAIL_JWT_SECRET` / `MAGICMAIL_ENCRYPT_KEY`）默认自动生成并持久化到数据库，也支持通过环境变量显式指定（优先级高于数据库值）。生产环境建议手动设置以避免数据库丢失导致无法解密。
>
> 💡 附件缓存相关变量（`CACHE_THRESHOLD`、`MIN_DISK_FREE`、`MAX_ATTACHMENT_SIZE`）单位均为 **MB**，直接填数字即可。

## API 文档

> 所有 `/api/v1/*` 接口均需 JWT 认证（除 `/auth/*` 外）

### 认证

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/v1/auth/register` | 用户注册 |
| POST | `/api/v1/auth/login` | 用户登录（返回 JWT Token）|
| GET | `/api/v1/auth/status` | 检查登录状态 |

### 邮箱管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/accounts` | 邮箱列表（脱敏，不含密码）|
| GET | `/api/v1/accounts/:id` | 邮箱详情 |
| POST | `/api/v1/accounts` | 创建邮箱账号 |
| PUT | `/api/v1/accounts/:id` | 更新邮箱信息 |
| DELETE | `/api/v1/accounts/:id` | 删除邮箱及数据 |
| POST | `/api/v1/accounts/test-connection` | 测试 IMAP 连接 |
| POST | `/api/v1/accounts/:id/sync` | 手动触发同步 |
| PUT | `/api/v1/accounts/:id/status` | 启用/停用邮箱账号 |

### 邮件管理

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/mails` | 邮件列表（分页/搜索/筛选）|
| GET | `/api/v1/mails/stats` | 统计数据 |
| GET | `/api/v1/mails/:id` | 邮件详情 |
| PUT | `/api/v1/mails/:id/read` | 标记已读/未读 |
| PUT | `/api/v1/mails/:id/star` | 标记星标 |
| DELETE | `/api/v1/mails/:id` | 删除邮件 |
| POST | `/api/v1/mails/batch-delete` | 批量删除邮件 |
| POST | `/api/v1/mails/send` | 发送邮件 (SMTP) |
| GET | `/api/v1/mails/stream` | SSE 实时推送流 (需 JWT) |
| GET | `/api/v1/mails/stream/health` | SSE 服务健康检查 |

### 草稿箱

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/drafts` | 草稿列表 |
| POST | `/api/v1/drafts` | 保存草稿 |
| GET | `/api/v1/drafts/:id` | 草稿详情 |
| PUT | `/api/v1/drafts/:id` | 更新草稿 |
| DELETE | `/api/v1/drafts/:id` | 删除草稿 |
| POST | `/api/v1/drafts/batch-delete` | 批量删除草稿 |

### 附件

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/attachments/mail/:mail_id` | 附件列表 |
| GET | `/api/v1/attachments/:id/download` | 下载附件（二进制流，支持懒加载/缓存）|

### Webhook 通知

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/webhooks` | Webhook 列表 |
| POST | `/api/v1/webhooks` | 创建 Webhook |
| GET | `/api/v1/webhooks/:id` | Webhook 详情 |
| PUT | `/api/v1/webhooks/:id` | 更新 Webhook |
| DELETE | `/api/v1/webhooks/:id` | 删除 Webhook |
| POST | `/api/v1/webhooks/:id/test` | 测试 Webhook 推送 |
| GET | `/api/v1/webhooks/:id/logs` | 查看推送日志 |

### Web Push 推送订阅

| 方法 | 路径 | 认证 | 说明 |
|------|------|------|------|
| GET | `/api/v1/push/vapid-public-key` | 否 | 获取 VAPID 公钥（用于前端订阅）|
| POST | `/api/v1/push/subscribe` | ✅ | 订阅浏览器推送通知 |
| POST | `/api/v1/push/unsubscribe` | ✅ | 取消订阅 |
| GET | `/api/v1/push/subscriptions` | ✅ | 查看订阅列表 |
| POST | `/api/v1/push/test` | ✅ | 发送测试推送 |

### 其他

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 健康检查 |

## 开发指南

### 添加新的 IMAP 功能

1. 在 `server/imap/client.go` 添加底层操作
2. 在 `server/imap/fetcher.go` 添加解析逻辑
3. 在 `server/imap/worker.go` 添加调度策略
4. 对应更新 Handler / Service / Route

### 添加新页面

1. 在 `web/src/views/` 创建 `.vue` 组件
2. 在 `web/src/router/index.js` 注册路由
3. 如需全局状态，在 `web/src/stores/` 创建 Pinia store
4. 在侧边栏 (`AppSidebar.vue`) 添加导航入口

### 修改主题色

编辑 `web/src/styles/themes.css` 中的 CSS 变量：
- 浅色主题：`:root` 块
- 深色主题：`[data-theme="dark"]` 块

### 数据备份

数据存储在 SQLite 单文件 (`data/magicmail.db`) 中，迁移时直接复制整个 `data/` 目录即可（需先停止服务）。

## 注意事项

- ⚠️ **IMAP IDLE 兼容性**：部分老旧 IMAP 服务器不支持 IDLE，会自动降级为定时轮询
- ⚠️ **SQLite 并发写入**：高并发场景下建议迁移到 PostgreSQL/MySQL
- ✅ **跨平台编译**：使用纯 Go SQLite 驱动 (`modernc.org/sqlite`)，无需 CGO，可直接交叉编译 Linux/macOS/Windows
- ✅ **混合附件缓存**：默认关闭自动缓存（`MAGICMAIL_AUTO_CACHE=false`），大附件采用懒加载模式按需从 IMAP 获取，不占用额外磁盘空间
- 🚀 **一键部署**：使用 `magicmail.sh install` 可在服务器上快速部署，安装后通过 `magicmail` 命令管理服务，支持 `doctor` 自检和 `update` 一键更新
- 🔒 **AGPL-3.0 许可**：本程序基于 AGPLv3 开源协议发布，网络使用需提供源代码获取方式

## License

Copyright (C) 2026 [magiccode (魔法代码)](https://github.com/magiccode1412/magicmail)

This program is free software: you can redistribute it and/or modify it under the terms of the **GNU Affero General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version**.

[查看完整协议文本 →](./LICENSE)
