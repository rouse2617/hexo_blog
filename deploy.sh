#!/bin/bash
# ============================================
# AI-Ops 快速部署脚本
# ============================================

set -e

# 颜色
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
IMAGE_NAME="${IMAGE_NAME:-ai-pro}"
CONTAINER_NAME="${CONTAINER_NAME:-ai-pro}"
HOST_PORT="${HOST_PORT:-8080}"
DATA_DIR="${DATA_DIR:-./data}"

info() { echo -e "${BLUE}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; }

# 显示帮助
show_help() {
    cat << EOF
AI-Ops 快速部署脚本

用法: $0 [命令]

命令:
    build       构建镜像
    up          启动服务
    down        停止服务
    restart     重启服务
    logs        查看日志
    stats       查看状态
    clean       清理容器和镜像
    shell       进入容器 Shell

环境变量:
    IMAGE_NAME  镜像名称 (默认: ai-pro)
    HOST_PORT   主机端口 (默认: 8080)

示例:
    $0 build    # 构建镜像
    $0 up       # 启动服务
    $0 logs     # 查看日志
EOF
}

# 构建镜像
build() {
    info "构建 Docker 镜像..."
    docker compose build
    success "构建完成"
}

# 启动服务
up() {
    info "启动服务..."

    # 创建数据目录
    mkdir -p "${DATA_DIR}"

    # 启动容器
    docker compose up -d

    success "服务已启动"
    info "访问地址: http://localhost:${HOST_PORT}"
    info "查看日志: $0 logs"
}

# 停止服务
down() {
    info "停止服务..."
    docker compose down
    success "服务已停止"
}

# 重启服务
restart() {
    info "重启服务..."
    docker compose restart
    success "服务已重启"
}

# 查看日志
logs() {
    docker compose logs -f "$@"
}

# 查看状态
stats() {
    docker compose ps
    echo ""
    info "容器资源使用:"
    docker stats "${CONTAINER_NAME}" --no-stream
}

# 清理
clean() {
    warn "这将删除容器和镜像，是否继续? (y/N)"
    read -r confirm
    if [ "$confirm" = "y" ] || [ "$confirm" = "Y" ]; then
        info "清理容器和镜像..."
        docker compose down -v --rmi all
        success "清理完成"
    else
        info "已取消"
    fi
}

# 进入容器
shell() {
    docker compose exec "${CONTAINER_NAME}" /bin/sh
}

# 主流程
main() {
    case "${1:-}" in
        build)
            build
            ;;
        up)
            up
            ;;
        down)
            down
            ;;
        restart)
            restart
            ;;
        logs)
            logs
            ;;
        stats)
            stats
            ;;
        clean)
            clean
            ;;
        shell)
            shell
            ;;
        help|--help|-h|"")
            show_help
            ;;
        *)
            error "未知命令: $1"
            show_help
            exit 1
            ;;
    esac
}

main "$@"
