# 安装部署

## 方式一：一键部署脚本（推荐）

> 最简单的方式，一条命令完成所有配置。支持 Linux 各发行版、macOS 和 Docker 环境。

### 快速安装

:::: code-group

```bash [GitHub 原始]
curl -fsSL https://raw.githubusercontent.com/magiccode1412/magicmail/main/deploy.sh -o deploy.sh
```

```bash [jsDelivr 镜像（国内）]
curl -fsSL https://cdn.jsdelivr.net/gh/magiccode1412/magicmail@main/deploy.sh -o deploy.sh
```

::::

```bash
chmod +x deploy.sh && sudo ./deploy.sh install
```

> 脚本内置 **GitHub 镜像自动切换**：当直连 `github.com` 失败时，会依次尝试以下镜像加速：
> - **jsDelivr CDN** — 最快最稳定（仅 raw 文件）
> - **mirror.ghproxy.com** — 全类型通用代理
> - **gh-proxy.com** / **ghfast.top** / **github.moeyy.xyz** — 备用镜像
>
> 无需任何额外配置，全程自动。

脚本会自动：
1. 检测系统环境（操作系统、发行版、CPU 架构）
2. 检查系统资源（磁盘空间、内存）
3. 检测端口占用并提示防火墙配置
4. 自动安装缺失的依赖工具
5. 从 GitHub Release 下载对应平台的二进制文件
6. 注册系统服务并设置开机自启
7. 注册 `magicmail` 全局命令
8. 启动服务并输出汇总信息

### 安装选项

```bash
# 指定端口
sudo ./deploy.sh install --port 3000

# 安装指定版本
sudo ./deploy.sh install --version v1.0.0

# 非交互模式（跳过所有确认，适合 CI/CD）
sudo ./deploy.sh install -y

# 组合使用
sudo ./deploy.sh install --port 9090 --version v1.2.0 -y
```

### magicmail 命令行工具

安装完成后，`magicmail` 命令会注册到 `/usr/local/bin`，可在终端直接使用：

#### 服务管理

```bash
magicmail status       # 查看服务运行状态和关键路径信息
magicmail start        # 启动服务（自动提权 sudo）
magicmail stop         # 停止服务
magicmail restart      # 重启服务
magicmail logs         # 查看最近 100 行日志
magicmail logs 50      # 查看最近 50 行日志
```

#### 维护操作

```bash
magicmail update       # 更新到最新版本（需确认）
magicmail update -y    # 无交互更新
magicmail doctor       # 环境健康自检
magicmail version      # 显示已安装版本和远程最新版对比
magicmail uninstall    # 卸载程序（数据可选保留）
```

#### 帮助信息

```bash
magicmail help         # 查看 CLI 命令帮助
./deploy.sh help       # 查看完整部署脚本帮助
```

### doctor 自检命令

`magicmail doctor` 会全面检查运行环境的健康状态，涵盖以下维度：

| 检查项目 | 说明 |
|----------|------|
| 操作系统 / 内核版本 | 系统基础信息 |
| CPU 架构检测 | amd64 / arm64 |
| 二进制文件 | 是否存在且可执行 |
| 部署脚本 | 是否存在于安装目录 |
| 版本记录 | 已安装版本号 |
| 数据主目录 | `/var/lib/magicmail` |
| 数据库文件 | SQLite 数据库是否存在及大小 |
| 日志文件 | 是否可读 |
| systemd / LaunchDaemon 服务 | 服务文件及运行状态 |
| 端口监听 | 目标端口是否在监听 |
| 网络连通性 | DNS 可达性 |
| CLI 命令 | `magicmail` 命令是否可用 |
| 磁盘空间 | 可用空间是否充足 |
| 内存 | 总内存是否满足最低要求 |

### 支持的平台

| 平台 | 包管理器 | 服务管理 |
|------|----------|----------|
| Ubuntu / Debian / Linux Mint | apt | systemd |
| CentOS / RHEL / Rocky / Alma / Fedora | dnf / yum | systemd |
| Arch Linux / Manjaro | pacman | systemd |
| Alpine Linux | apk | systemd |
| OpenSUSE / SLES | zypper | systemd |
| macOS (Intel & Apple Silicon) | brew | LaunchDaemon |
| Docker 容器 | — | nohup 后台模式 |

::: tip Docker 环境说明
脚本会自动检测 Docker 容器环境，此时跳过 systemd 服务创建，改用 nohup 后台运行模式。
:::

### 安装目录结构

```
/opt/magicmail/              # 程序安装目录
├── magicmail                # 主程序二进制
├── deploy.sh               # 部署脚本副本（供 CLI wrapper 调用）
└── .version                # 版本记录文件

/var/lib/magicmail/          # 数据目录
└── data/
    └── magicmail.db        # SQLite 数据库

/var/log/
└── magicmail.log           # 运行日志

/usr/local/bin/
└── magicmail               # CLI 命令（软链/wrapper）

/etc/systemd/system/
└── magicmail.service        # systemd 服务文件（仅 Linux）
```

### 更新与卸载

```bash
# 更新到最新版本
sudo ./deploy.sh update
# 或
sudo magicmail update

# 卸载（保留数据）
sudo ./deploy.sh uninstall
# 或
sudo magicmail uninstall

# 彻底卸载（含数据），卸载时选择删除数据即可
```

---

## 方式二：直接下载二进制

在 [GitHub Releases](https://github.com/magiccode1412/magicmail/releases) 下载对应平台的预编译二进制：

| 平台 | 文件 |
|------|------|
| Linux amd64 | `magicmail-linux-amd64` |
| Linux arm64 | `magicmail-linux-arm64` |
| macOS Intel | `magicmail-macos-x86_64` |
| macOS Apple Silicon | `magicmail-macos-arm64` |
| Windows amd64 | `magicmail-windows-amd64.exe` |

下载后赋予执行权限并运行：

```bash
chmod +x magicmail-linux-amd64
./magicmail-linux-amd64
```

---

## 方式三：Docker 部署

```bash
docker build -t magicmail .
docker run -d \
  -p 8080:8080 \
  -v ./data:/app/data \
  --name magicmail \
  --restart unless-stopped \
  magicmail
```

也可以通过一键部署脚本在容器内安装（自动适配 Docker 环境）。

---

## 方式四：systemd 手动配置（Linux）

如果需要手动配置 systemd 服务，可参考以下步骤：

```bash
# 复制服务文件
sudo cp server/magicmail.service /etc/systemd/system/

# 根据实际路径修改 ExecStart
sudo systemctl daemon-reload
sudo systemctl enable magicmail
sudo systemctl start magicmail
```

或使用一键部署脚本自动完成以上全部步骤。

---

## 反向代理配置

### Nginx

```nginx
server {
    listen 80;
    server_name mail.example.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE 长连接支持（实时推送必需）
        proxy_buffering off;
        proxy_cache off;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }
}
```

### Caddy

```
mail.example.com {
    reverse_proxy localhost:8080

    # SSE 长连接支持
    flush_interval -1
}
```

::: tip SSE 代理注意事项
Magicmail 使用 SSE (Server-Sent Events) 实现实时邮件推送。反向代理必须禁用响应缓冲，否则前端无法收到实时推送。
:::
