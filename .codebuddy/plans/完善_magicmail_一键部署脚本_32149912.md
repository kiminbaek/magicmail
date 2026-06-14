---
name: 完善 magicmail 一键部署脚本
overview: 完善 deploy.sh 脚本，增加跨平台支持、完整依赖检查、magicmail 全局命令入口、安装后信息汇总等功能
todos:
  - id: enhance-platform-detection
    content: "增强平台检测模块: 添加 is_docker()、detect_distro()、get_package_manager() 函数，覆盖 Linux 各发行版/macOS/Docker 环境"
    status: completed
  - id: enhance-deps-check
    content: "重构依赖检查与安装: 实现 install_packages() 通用包安装、check_system_resources() 资源预检、check_port_conflict() 端口冲突检测、check_firewall() 防火墙提示"
    status: completed
    dependencies:
      - enhance-platform-detection
  - id: create-cli-wrapper
    content: "创建 magicmail CLI wrapper 生成函数: 在 /usr/local/bin/magicmail 生成支持 status/start/stop/restart/update/uninstall/logs/version/doctor/help 子命令的 shell 脚本"
    status: completed
  - id: enhance-install-flow
    content: "增强 cmd_install 主流程: 串接资源检测->端口检查->依赖安装->下载->服务创建->wrapper 注册->启动->汇总全链路"
    status: completed
    dependencies:
      - enhance-deps-check
      - create-cli-wrapper
  - id: add-doctor-version
    content: "新增 doctor 自检和 version 子命令: cmd_doctor() 全面检测运行环境健康度，cmd_version() 显示版本对比"
    status: completed
  - id: unify-summary-and-cleanup
    content: 实现统一 print_summary 汇总输出，增强 cmd_update/cmd_uninstall 流程，添加 --yes 非交互模式，完善 main() 路由
    status: completed
    dependencies:
      - enhance-install-flow
      - add-doctor-version
---

## 产品概述

完善 Magicmail 邮件管理系统的一键部署脚本 `deploy.sh`，使其成为一个功能完备、跨平台兼容的部署管理工具，并在安装完成后提供全局 `magicmail` 命令行工具供用户日常使用。

## 核心功能

### 1. 全局 `magicmail` 命令行入口

- 安装时在 `/usr/local/bin/magicmail` 创建 CLI wrapper 脚本（与现有 `server/magicmail.service` 中 `ExecStart=/usr/local/bin/magicmail` 路径一致）
- 支持子命令: `status`, `start`, `stop`, `restart`, `update`, `uninstall`, `logs`, `version`, `doctor`, `help`
- 示例用法: `magicmail status` 查看运行状态、`magicmail update` 一键更新、`magicmail doctor` 环境自检

### 2. 多平台深度适配

- **Linux 发行版**: Ubuntu/Debian (apt), CentOS/RHEL (yum/dnf), Arch (pacman), Alpine (apk), OpenSUSE (zypper)
- **macOS**: LaunchDaemon + Homebrew 依赖支持
- **CPU 架构**: amd64/x86_64, arm64/aarch64 (已有基础，需完善错误提示)
- **Docker 容器内运行检测**：自动识别容器环境并调整行为

### 3. 智能依赖检查与自动安装

- 安装前自动检测并安装缺失的工具依赖: curl/wget（下载工具）、tar/gzip（解压工具）
- 支持上述所有发行版的包管理器自动选择
- 系统资源预检: 磁盘空间 (>=200MB)、内存 (>=256MB)
- 端口占用检测: 8080 端口冲突预警

### 4. 完整的安装/更新/卸载流程

- **install**: 环境检测 -> 依赖安装 -> 目录初始化 -> 二进制下载 -> 服务创建 -> 符号链接创建 -> 启动服务 -> 信息汇总
- **update**: 版本对比 -> 停止服务 -> 下载新版 -> 替换二进制 -> 重启服务 -> 变更汇总
- **uninstall**: 确认交互 -> 停止服务 -> 移除服务文件 -> 清理程序文件 -> 可选删除数据 -> 清理结果汇总

### 5. 统一信息汇总输出 (`print_summary`)

- 每个操作完成后统一调用汇总函数，打印: 版本号、访问地址、数据目录、日志位置、服务状态、常用命令速查
- 执行完毕后脚本自动退出，无残留挂起

### 6. 辅增强功能

- `doctor` 自检命令: 检测二进制存在性、服务状态、端口监听、磁盘空间、权限等
- `version` 子命令: 显示已安装版本和最新版本对比
- 防火墙开放提示 (ufw/firewalld/iptables)
- 非交互模式支持 (`--yes` / `-y` 跳过确认)

## 技术栈

- **语言**: Bash 4.0+ (兼容 POSIX sh 尽可能)
- **目标文件**: `/workspace/deploy.sh` (单文件修改，约 617 行 → 预计扩展至 ~950 行)
- **依赖工具**: curl/wget, tar, systemctl(linux)/launchctl(macos)

## 实现方案

### 核心架构设计

采用**单一脚本 + CLI Wrapper 分离**的架构：

```
deploy.sh          # 部署引擎（保留原有功能，增强各模块）
  ├── 环境检测层   # detect_os/detect_arch/check_system_resources/is_docker
  ├── 依赖管理层   # ensure_deps (多包管理器) / check_port_conflict
  ├── 安装部署层   # cmd_install (增强) / install_binary
  ├── 服务管理层   # svc_start/stop/restart/status/logs (Linux systemd + macOS LaunchDaemon)
  ├── 更新卸载层   # cmd_update / cmd_uninstall (增强汇总输出)
  └── CLI 入口层   # main() + print_summary()

/usr/local/bin/magicmail  # [新建] CLI wrapper，委托给 deploy.sh 或直接实现轻量子命令
```

### 关键技术决策

1. **`magicmail` 命令实现方式**: 采用 wrapper 脚本而非 symlink。Wrapper 脚本内部调用 `deploy.sh` 的对应函数（通过 source 或子进程调用），这样 `magicmail status` 等命令无需 root 权限即可查看状态，而 start/stop 等操作再自动提权。Wrapper 脚本在 `cmd_install()` 中自动生成到 `/usr/local/bin/magicmail`。

2. **包管理器探测策略**: 使用函数数组/链式探测，优先级为 apt > dnf > yum > pacman > apk > zypper > brew，匹配到即止。每种管理器的 install 命令独立封装。

3. **Docker 容器检测**: 通过检查 `/.dockerenv` 文件或 `/proc/1/cgroup` 中包含 docker/lxc 字样来判断。容器内跳过 systemctl 相关操作，改用直接前台/后台运行模式。

4. **print_summary 统一出口**: 所有主要操作（install/update/uninstall）结束时都调用此函数，接受操作类型参数，输出对应的汇总信息面板。

### 数据流

```
用户执行 ./deploy.sh install 或 magicmail install
  → print_banner()
  → detect_os() + detect_arch() + is_docker()
  → check_system_resources() [磁盘/内存]
  → check_port_conflict(8080)
  → ensure_deps() [curl, wget, tar 等]
  → init_directories()
  → get_latest_version() + install_binary()
  → create_service() [systemd/launchdaemon/foreground]
  → create_cli_wrapper() [→ /usr/local/bin/magicmail]
  → svc_start()
  → print_summary("install")  ← 统一汇总
  → exit 0
```

### 关键实现细节

#### magicmail CLI Wrapper 结构

```
#!/usr/bin/env bash
# /usr/local/bin/magicmail - 由 deploy.sh install 自动生成
MAGICMAIL_INSTALL_DIR="/opt/magicmail"
DEPLOY_SCRIPT="${MAGICMAIL_INSTALL_DIR}/deploy.sh"

# 无参数默认 show + status
# 子命令: status|start|stop|restart|update|uninstall|logs|version|doctor|help
# status/version/doctor/logs 无需 root
# start/stop/restart/update/uninstall 需要 root（自动 sudo 提示）
```

#### 依赖安装函数增强

- `ensure_pkg_manager()`: 探测系统包管理器
- `install_pkg_via_pm()`: 通过探测到的包管理器安装指定包列表
- `check_system_resources()`: df 检查磁盘空间 >= 200MB，free 检查内存 >= 256MB
- `check_firewall()`: 检测 ufw/firewalld/iptables 并给出开放 8080 端口的提示命令

#### 平台适配补充

- OpenSUSE: zypper install -y
- Alpine Linux: apk add (musl 兼容性提示)
- macOS: brew install (可选)
- Docker 内部: 跳过 systemd，使用直接后台运行 + pidfile 管理

## 目录结构

```
/workspace/
└── deploy.sh              # [MODIFY] 主部署脚本，全面增强
    新增/修改内容:
    ├── is_docker()                    # [NEW] Docker 容器环境检测
    ├── detect_distro()                # [NEW] Linux 发行版详细检测
    ├── get_package_manager()          # [NEW] 返回可用包管理器名称
    ├── install_packages()             # [NEW] 通用的包安装接口
    ├── check_system_resources()       # [NEW] 磁盘/内存预检
    ├── check_port_conflict()          # [NEW] 端口占用检测
    ├── check_firewall()               # [NEW] 防火墙状态检测与提示
    ├── create_cli_wrapper()           # [NEW] 生成 /usr/local/bin/magicmail 命令
    ├── print_summary()                # [NEW] 统一信息汇总输出函数
    ├── cmd_doctor()                   # [NEW] 环境自检命令
    ├── cmd_version()                  # [NEW] 版本查看命令
    ├── cmd_install()                  # [MODIFY] 注入依赖检查、wrapper 创建、summary
    ├── cmd_update()                   # [MODIFY] 增加 summary 调用
    ├── cmd_uninstall()                 # [MODIFY] 增加 wrapper 清理、summary 调用
    ├── ensure_download_tool()         # [MODIFY] 增加 zypper 支持
    ├── main()                         # [MODIFY] 增加 doctor/version 路由、--yes 支持
```

### SubAgent

- **code-explorer**
- Purpose: 在实现过程中探索 server/ 目录下的配置结构和 main.go 的启动参数，确保 deploy.sh 生成的 service 文件和环境变量与 Go 程序实际读取的配置项完全一致
- Expected outcome: 确认 config 包的环境变量映射关系，保证部署脚本的 Environment 配置准确无误