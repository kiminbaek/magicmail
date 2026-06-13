#!/usr/bin/env bash
# ============================================================================
# Magicmail 一键部署脚本
# 用法: ./deploy.sh [命令]
#
# 命令:
#   install     安装并启动服务（默认）
#   start       启动服务
#   stop        停止服务
#   restart     重启服务
#   status      查看运行状态
#   update      更新到最新版本
#   uninstall   卸载（保留数据，可选删除）
#   logs        查看日志
# ============================================================================
set -euo pipefail

# ─── 颜色定义 ──────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ─── 配置 ──────────────────────────────────────────
REPO="magiccode1412/magicmail"
REPO_URL="https://github.com/${REPO}"
API_URL="https://api.github.com/repos/${REPO}"
RELEASE_URL="${API_URL}/releases/latest"

# 安装路径
INSTALL_DIR="/opt/magicmail"
BIN_NAME="magicmail"
DATA_DIR="/var/lib/magicmail"
LOG_FILE="/var/log/magicmail.log"
CONFIG_DIR="${INSTALL_DIR}/config"

# systemd 服务名
SERVICE_NAME="magicmail"

# ─── 工具函数 ──────────────────────────────────────
info()    { echo -e "${BLUE}[INFO]${NC}  $*"; }
success() { echo -e "${GREEN}[OK]${NC}    $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
die()     { error "$*"; exit 1; }

print_banner() {
    echo -e "${CYAN}${BOLD}"
    echo "╔══════════════════════════════════════════╗"
    echo "║        Magicmail 邮件管理系统                ║"
    echo "║           一键部署工具                     ║"
    echo "╚══════════════════════════════════════════╝"
    echo -e "${NC}"
}

# 检测操作系统
detect_os() {
    local os_name="$(uname -s)"
    case "${os_name,,}" in
        linux*)
            echo "linux" ;;
        darwin*)
            echo "darwin" ;;
        msys*|mingw*|cygwin*)
            echo "windows" ;;
        *)
            die "不支持的操作系统: ${os_name}" ;;
    esac
}

# 检测 CPU 架构
detect_arch() {
    local arch_name="$(uname -m)"
    case "${arch_name}" in
        x86_64|amd64)
            echo "amd64" ;;
        aarch64|arm64)
            echo "arm64" ;;
        armv7l|armhf)
            die "不支持 ARMv7，请使用 ARM64 (aarch64) 设备" ;;
        i386|i686)
            die "不支持 32 位系统，请使用 64 位系统" ;;
        *)
            die "不支持的 CPU 架构: ${arch_name}" ;;
    esac
}

# 获取下载文件名后缀
get_binary_suffix() {
    local os="$1"
    local arch="$2"
    case "${os}-${arch}" in
        linux-amd64)      echo "" ;;
        linux-arm64)      echo "-arm64" ;;
        darwin-amd64)     echo "-macos-x86_64" ;;
        darwin-arm64)     echo "-macos-arm64" ;;
        windows-amd64)    echo ".exe" ;;
        *)                die "无法确定二进制文件名: ${os}/${arch}" ;;
    esac
}

# 获取最新版本号
get_latest_version() {
    # 优先使用本地 git tag（如果是从源码部署）
    if command -v git &>/dev/null && [ -d ".git" ] 2>/dev/null; then
        local tag
        tag="$(git describe --tags --abbrev=0 2>/dev/null || true)"
        if [ -n "${tag}" ]; then
            echo "${tag}"
            return
        fi
    fi

    # 从 GitHub API 获取
    if command -v curl &>/dev/null; then
        local version
        version="$(curl -fsSL "${RELEASE_URL}" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/' || true)"
        if [ -n "${version}" ]; then
            echo "${version}"
            return
        fi
    elif command -v wget &>/dev/null; then
        local version
        version="$(wget -qO- "${RELEASE_URL}" 2>/dev/null | grep '"tag_name"' | head -1 | sed -E 's/.*"tag_name":\s*"([^"]+)".*/\1/' || true)"
        if [ -n "${version}" ]; then
            echo "${version}"
            return
        fi
    fi

    die "无法获取最新版本号，请检查网络连接或手动指定版本"
}

# 下载文件
download_file() {
    local url="$1"
    local dest="$2"

    if command -v curl &>/dev/null; then
        curl -fSL --progress-bar -o "${dest}" "${url}" || die "下载失败: ${url}"
    elif command -v wget &>/dev/null; then
        wget --progress=bar:force:noscroll -O "${dest}" "${url}" || die "下载失败: ${url}"
    else
        die "需要 curl 或 wget 来下载文件"
    fi
}

# ─── 权限检查 ─────────────────────────────────────
check_root() {
    if [[ $EUID -ne 0 ]]; then
        warn "部分操作需要 root 权限，请使用 sudo 运行:"
        echo "  sudo $0 $*"
        exit 1
    fi
}

# ─── 依赖安装 ─────────────────────────────────────
ensure_download_tool() {
    if ! command -v curl &>/dev/null && ! command -v wget &>/dev/null; then
        info "安装下载工具..."
        if command -v apt-get &>/dev/null; then
            apt-get update -qq && apt-get install -y -qq curl
        elif command -v yum &>/dev/null; then
            yum install -y -q curl
        elif command -v dnf &>/dev/null; then
            dnf install -y -q curl
        elif command -v pacman &>/dev/null; then
            pacman -S --noconfirm curl
        elif command -v apk &>/dev/null; then
            apk add curl
        elif command -v brew &>/dev/null; then
            brew install curl
        else
            die "无法自动安装 curl/wget，请手动安装后再试"
        fi
        success "curl 已安装"
    fi
}

# ─── 目录初始化 ───────────────────────────────────
init_directories() {
    mkdir -p "${INSTALL_DIR}"
    mkdir -p "${DATA_DIR}"
    mkdir -p "$(dirname "${LOG_FILE}")"
    success "目录已创建: ${INSTALL_DIR}, ${DATA_DIR}"
}

# ─── 二进制下载与安装 ─────────────────────────────
install_binary() {
    local target_os="$1"
    local target_arch="$2"
    local version="$3"

    local suffix
    suffix="$(get_binary_suffix "${target_os}" "${target_arch}")"
    local binary_url="${REPO_URL}/releases/download/${version}/${BIN_NAME}${suffix}"

    local binary_path="${INSTALL_DIR}/${BIN_NAME}${suffix}"

    info "正在下载 Magicmail ${version} (${target_os}/${target_arch})..."
    info "下载地址: ${binary_url}"

    download_file "${binary_url}" "${binary_path}"

    chmod +x "${binary_path}"

    # 如果有旧版本，先备份再替换
    local final_bin="${INSTALL_DIR}/${BIN_NAME}"
    if [ -f "${final_bin}" ] && [ "${final_bin}" != "${binary_path}" ]; then
        mv "${final_bin}" "${final_bin}.bak.$(date +%Y%m%d%H%M%S)" || true
    fi

    # 如果 suffix 为空（Linux amd64），不需要 mv；否则创建统一入口
    if [ "${suffix}" != "" ]; then
        ln -sf "$(basename "${binary_path}")" "${final_bin}"
    fi

    success "二进制已安装: ${final_bin}"
}

# ─── Linux systemd 服务 ───────────────────────────
create_systemd_service() {
    local service_file="/etc/systemd/system/${SERVICE_NAME}.service"

    cat > "${service_file}" << EOF
[Unit]
Description=Magicmail Mail Management Service
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=${INSTALL_DIR}/${BIN_NAME}
WorkingDirectory=${INSTALL_DIR}
Restart=on-failure
RestartSec=5
Environment=MAGICMAIL_DSN=${DATA_DIR}/magicmail.db

# 日志输出到文件（也可用 journalctl 查看）
StandardOutput=append:${LOG_FILE}
StandardError=append:${LOG_FILE}

# 安全限制
NoNewPrivileges=true
ProtectSystem=strict
ReadWritePaths=${DATA_DIR} ${INSTALL_DIR}
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable "${SERVICE_NAME}"
    success "systemd 服务已创建: ${service_file}"
}

# ─── macOS LaunchDaemon 服务 ─────────────────────
create_launchd_service() {
    local plist_file="/Library/LaunchDaemons/com.magicmail.service.plist"

    cat > "${plist_file}" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.magicmail.service</string>
    <key>ProgramArguments</key>
    <array>
        <string>${INSTALL_DIR}/${BIN_NAME}</string>
    </array>
    <key>WorkingDirectory</key>
    <string>${INSTALL_DIR}</string>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>${LOG_FILE}</string>
    <key>StandardErrorPath</key>
    <string>${LOG_FILE}</string>
    <key>EnvironmentVariables</key>
    <dict>
        <key>MAGICMAIL_DSN</key>
        <string>${DATA_DIR}/magicmail.db</string>
    </dict>
</dict>
</plist>
EOF

    launchctl load "${plist_file}" 2>/dev/null || true
    success "LaunchDaemon 服务已创建: ${plist_file}"
}

# ─── 服务管理函数 ─────────────────────────────────
svc_start() {
    check_root start
    if is_linux; then
        systemctl start "${SERVICE_NAME}" || die "启动失败"
        sleep 1
        svc_status
    else
        launchctl start "com.magicmail.service" 2>/dev/null \
            || "${INSTALL_DIR}/${BIN_NAME}" &
        sleep 1
        svc_status
    fi
    success "服务已启动"
}

svc_stop() {
    check_root stop
    if is_linux; then
        systemctl stop "${SERVICE_NAME}" || warn "服务可能未在运行"
    else
        launchctl stop "com.magicmail.service" 2>/dev/null || pkill -f "${INSTALL_DIR}/${BIN_NAME}" || true
    fi
    success "服务已停止"
}

svc_restart() {
    check_root restart
    if is_linux; then
        systemctl restart "${SERVICE_NAME}" || die "重启失败"
    else
        svc_stop
        sleep 1
        svc_start
    fi
    success "服务已重启"
}

svc_status() {
    if is_linux; then
        if systemctl is-active "${SERVICE_NAME}" &>/dev/null; then
            echo -e "${GREEN}● 服务状态: 运行中${NC}"
            systemctl status "${SERVICE_NAME}" --no-pager -l 2>/dev/null | head -15
        else
            echo -e "${RED}○ 服务状态: 未运行${NC}"
        fi
    else
        if pgrep -f "${INSTALL_DIR}/${BIN_NAME}" &>/dev/null; then
            echo -e "${GREEN}● 服务状态: 运行中${NC}"
            ps aux | grep "[${BIN_NAME:0:1}]${BIN_NAME:1}" | head -5
        else
            echo -e "${RED}○ 服务状态: 未运行${NC}"
        fi
    fi

    # 端口检测
    if command -v ss &>/dev/null; then
        local port_info
        port_info="$(ss -tlnp 2>/dev/null | grep ':8080' || true)"
        if [ -n "${port_info}" ]; then
            echo ""
            info "端口监听: 8080"
        fi
    elif command -v lsof &>/dev/null; then
        if lsof -i :8080 &>/dev/null; then
            echo ""
            info "端口监听: 8080"
        fi
    fi

    echo ""
    info "访问地址: http://localhost:8080"
    info "数据目录: ${DATA_DIR}"
    info "日志文件: ${LOG_FILE}"
}

svc_logs() {
    if [ -f "${LOG_FILE}" ]; then
        tail -100 "${LOG_FILE}" 2>/dev/null || warn "无法读取日志"
    elif is_linux; then
        journalctl -u "${SERVICE_NAME}" --no-pager -n 100 2>/dev/null || warn "无日志可读"
    else
        warn "日志文件不存在: ${LOG_FILE}"
    fi
}

is_linux() { [[ "$(uname -s)" == Linux* ]]; }

# ─── 主安装流程 ───────────────────────────────────
cmd_install() {
    check_root install
    print_banner

    # 1. 环境检测
    local target_os target_arch
    target_os="$(detect_os)"
    target_arch="$(detect_arch)"

    if [ "${target_os}" == "windows" ]; then
        die "Windows 不支持此脚本，请直接从 GitHub Release 下载 exe 文件运行"
    fi

    info "目标平台: ${target_os}/${target_arch}"

    # 2. 安装依赖
    ensure_download_tool

    # 3. 创建目录
    init_directories

    # 4. 获取版本并下载
    local version
    version="$(get_latest_version)"
    info "最新版本: ${version}"

    install_binary "${target_os}" "${target_arch}" "${version}"

    # 5. 验证安装
    local bin_path="${INSTALL_DIR}/${BIN_NAME}"
    if [ ! -x "${bin_path}" ]; then
        # 可能是带后缀的文件
        for f in "${bin_path}".exe "${bin_path}-"*; do
            if [ -x "${f}" ]; then
                bin_path="${f}"
                break
            fi
        done
    fi

    if ! "${bin_path}" --version 2>&1 || true; then
        info "（--version 参数暂不可用，跳过验证）"
    fi

    # 6. 创建服务管理
    if is_linux; then
        create_systemd_service
    else
        create_launchd_service
    fi

    # 7. 启动服务
    echo ""
    info "正在启动服务..."
    if is_linux; then
        systemctl start "${SERVICE_NAME}"
        sleep 2
    else
        "${bin_path}" &
        sleep 2
    fi

    # 8. 显示结果
    echo ""
    echo -e "${GREEN}═══════════════════════════════════════${NC}"
    echo -e "  ${GREEN}✅ Magicmail 安装成功！${NC}"
    echo ""
    echo -e "  访问地址: ${CYAN}http://localhost:8080${NC}"
    echo -e "  数据目录: ${CYAN}${DATA_DIR}${NC}"
    echo -e "  日志文件: ${CYAN}${LOG_FILE}${NC}"
    echo -e "  二进制:   ${CYAN}${bin_path}${NC}"
    echo ""
    if is_linux; then
        echo -e "  常用命令:"
        echo -e "    sudo $0 restart   重启服务"
        echo -e "    sudo $0 status    查看状态"
        echo -e "    sudo $0 logs      查看日志"
        echo -e "    sudo $0 stop      停止服务"
        echo -e "    sudo $0 update    更新版本"
    fi
    echo -e "${GREEN}═══════════════════════════════════════${NC}"

    # 9. 自动打开浏览器（如果可能）
    if command -v xdg-open &>/dev/null; then
        (sleep 1 && xdg-open "http://localhost:8080" &) 2>/dev/null || true
    elif command -v open &>/dev/null; then
        (sleep 1 && open "http://localhost:8080") 2>/dev/null || true
    fi
}

# ─── 更新 ─────────────────────────────────────────
cmd_update() {
    check_root update
    print_banner
    info "检查更新..."

    local target_os target_arch new_version
    target_os="$(detect_os)"
    target_arch="$(detect_arch)"
    new_version="$(get_latest_version)"

    info "最新版本: ${new_version}"

    # 获取当前运行的版本（如果有）
    local current_version=""
    if [ -f "${INSTALL_DIR}/${BIN_NAME}" ]; then
        current_version="$("${INSTALL_DIR}/${BIN_NAME}" --version 2>/dev/null || echo "unknown")"
        info "当前版本: ${current_version}"
    fi

    if [ "${current_version:-unknown}" = "${new_version}" ]; then
        success "当前已是最新版本 (${new_version})"
        return
    fi

    warn "即将从 ${current_version:-未知} 更新至 ${new_version}"
    read -rp "确认继续? [Y/n] " confirm
    if [[ "${confirm:-Y}" =~ ^[Nn]$ ]]; then
        info "已取消"
        return
    fi

    # 先停止服务
    svc_stop

    # 下载新版
    install_binary "${target_os}" "${target_arch}" "${new_version}"

    # 重启服务
    svc_start

    success "更新完成 → ${new_version}"
}

# ─── 卸载 ─────────────────────────────────────────
cmd_uninstall() {
    check_root uninstall
    print_banner

    warn "以下操作将卸载 Magicmail 服务"
    echo ""
    echo "  数据目录: ${DATA_DIR} （包含数据库和配置）"
    echo ""

    read -rp "是否同时删除数据? [y/N] " delete_data
    read -rp "确认卸载? [y/N] " confirm

    if [[ ! "${confirm}" =~ ^[Yy]$ ]]; then
        info "已取消卸载"
        return
    fi

    # 停止服务
    svc_stop 2>/dev/null || true
    sleep 1

    # 移除 systemd / LaunchDaemon
    if is_linux && [ -f "/etc/systemd/system/${SERVICE_NAME}.service" ]; then
        systemctl disable "${SERVICE_NAME}" 2>/dev/null || true
        rm -f "/etc/systemd/system/${SERVICE_NAME}.service"
        systemctl daemon-reload
        info "systemd 服务已移除"
    fi

    if [ -f "/Library/LaunchDaemons/com.magicmail.service.plist" ]; then
        launchctl unload "/Library/LaunchDaemons/com.magicmail.service.plist" 2>/dev/null || true
        rm -f "/Library/LaunchDaemons/com.magicmail.service.plist"
        info "LaunchDaemon 已移除"
    fi

    # 删除二进制
    rm -rf "${INSTALL_DIR}"
    info "程序文件已删除"

    # 可选：删除数据
    if [[ "${delete_data}" =~ ^[Yy]$ ]]; then
        rm -rf "${DATA_DIR}"
        rm -f "${LOG_FILE}"
        info "数据和日志已删除"
    else
        info "数据已保留于: ${DATA_DIR}"
    fi

    success "Magicmail 已卸载"
}

# ─── 入口 ─────────────────────────────────────────
main() {
    local cmd="${1:-install}"
    shift 2>/dev/null || true

    case "${cmd}" in
        install)
            cmd_install ;;
        start)
            svc_start ;;
        stop)
            svc_stop ;;
        restart)
            svc_restart ;;
        status)
            svc_status ;;
        update)
            cmd_update ;;
        uninstall|remove)
            cmd_uninstall ;;
        log|logs)
            svc_logs ;;
        help|--help|-h)
            echo "用法: $0 [命令]"
            echo ""
            echo "命令:"
            echo "  install     安装并启动服务（默认）"
            echo "  start       启动服务"
            echo "  stop        停止服务"
            echo "  restart     重启服务"
            echo "  status      查看运行状态"
            echo "  update      更新到最新版本"
            echo "  uninstall   卸载（保留数据）"
            echo "  logs         查看日志"
            echo ""
            echo "仓库: ${REPO_URL}"
            ;;
        *)
            die "未知命令: ${cmd}。使用 '$0 help' 查看帮助" ;;
    esac
}

main "$@"
