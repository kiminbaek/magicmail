# 快速开始

欢迎来到 Magicmail！本页将帮助你在几分钟内完成部署并开始使用。

## 环境要求

- **Go** >= 1.21
- **Node.js** >= 18
- **pnpm** (推荐) 或 npm

## 方式一：构建脚本（推荐）

一键完成前端构建 + Go 编译，输出单二进制文件：

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

构建产物输出到 `bin/magicmail`（或 `bin/magicmail.exe`）。

## 方式二：开发环境

前后端同时启动，支持热重载：

```bash
# 一键启动
./dev.sh start

# 停止
./dev.sh stop
```

或手动分步启动：

```bash
# 终端 1：启动后端 (端口 8080)
cd server && go run .

# 终端 2：启动前端开发服务器 (端口 5173)
cd web && pnpm install && pnpm dev
```

Vite 开发服务器会自动代理 `/api` 请求到后端。

## 方式三：手动生产构建

```bash
# 1. 构建前端
cd web && pnpm build && cd ..

# 2. 编译后端（前端产物自动嵌入二进制）
cd server && go build -o ../bin/magicmail .

# 3. 运行
../bin/magicmail
```

## 启动服务

构建完成后运行二进制文件：

```bash
./bin/magicmail
```

默认监听 **http://localhost:8080**，浏览器打开即可使用。

::: tip 首次使用
首次启动时会自动：
- 创建 SQLite 数据库 (`data/magicmail.db`)
- 生成 JWT 密钥和加密密钥
- 注册管理员账号
:::

## 下一步

- [安装部署](/guide/installation) - 了解更多部署选项
- [功能特性](/guide/features) - 浏览完整功能列表
- [API 文档](/api/overview) - 查看 RESTful 接口说明
