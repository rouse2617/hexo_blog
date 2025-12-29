#!/bin/bash
# ============================================
# AI-Ops Docker 镜像打包脚本
# ============================================

set -e  # 遇到错误立即退出

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 默认配置
IMAGE_NAME="${IMAGE_NAME:-ai-pro}"
IMAGE_TAG="${IMAGE_TAG:-latest}"
REGISTRY="${REGISTRY:-}"
DOCKERFILE="${DOCKERFILE:-Dockerfile}"
BUILD_CONTEXT="${BUILD_CONTEXT:-.}"
PLATFORM="${PLATFORM:-linux/amd64}"
PUSH="${PUSH:-false}"

# 函数：打印信息
info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 函数：显示帮助
show_help() {
    cat << EOF
AI-Ops Docker 镜像打包脚本

用法: $0 [选项]

选项:
    -n, --name NAME        镜像名称 (默认: ai-pro)
    -t, --tag TAG          镜像标签 (默认: latest)
    -r, --registry REGISTRY Docker 仓库地址 (例如: registry.cn-hangzhou.aliyuncs.com)
    -p, --push             构建后推送到仓库
    -m, --multi-arch       构建多架构镜像 (amd64, arm64)
    -h, --help             显示帮助信息

环境变量:
    IMAGE_NAME             镜像名称
    IMAGE_TAG              镜像标签
    REGISTRY               Docker 仓库地址
    PUSH                   是否推送 (true/false)
    PLATFORM               目标平台

示例:
    # 基础构建
    $0

    # 指定名称和标签
    $0 -n my-ai-pro -t v1.0.0

    # 构建并推送到阿里云
    $0 -r registry.cn-hangzhou.aliyuncs.com/mynamespace -p

    # 构建多架构镜像
    $0 -m

    # 构建并推送到 Docker Hub
    $0 -n username/ai-pro -t v1.0.0 -p
EOF
}

# 函数：检查依赖
check_dependencies() {
    info "检查依赖..."

    if ! command -v docker &> /dev/null; then
        error "Docker 未安装，请先安装 Docker"
        exit 1
    fi

    success "依赖检查通过"
}

# 函数：构建镜像
build_image() {
    local full_image="${IMAGE_NAME}:${IMAGE_TAG}"
    local build_cmd="docker build"

    # 添加平台参数
    build_cmd="$build_cmd --platform ${PLATFORM}"

    # 添加构建参数
    build_cmd="$build_cmd -t ${full_image}"
    build_cmd="$build_cmd -f ${DOCKERFILE}"
    build_cmd="$build_cmd ${BUILD_CONTEXT}"

    info "开始构建镜像: ${full_image}"
    info "平台: ${PLATFORM}"
    info "构建上下文: ${BUILD_CONTEXT}"

    if $build_cmd; then
        success "镜像构建成功: ${full_image}"

        # 显示镜像大小
        local image_size=$(docker images "${full_image}" --format "{{.Size}}")
        info "镜像大小: ${image_size}"
    else
        error "镜像构建失败"
        exit 1
    fi
}

# 函数：构建多架构镜像
build_multi_arch() {
    local full_image="${IMAGE_NAME}:${IMAGE_TAG}"

    if [ -n "${REGISTRY}" ]; then
        full_image="${REGISTRY}/${IMAGE_NAME}:${IMAGE_TAG}"
    fi

    info "开始构建多架构镜像: ${full_image}"
    info "架构: linux/amd64, linux/arm64"

    # 创建并使用 buildx 构建器
    docker buildx create --use --name multiarch-builder 2>/dev/null || true

    # 构建并加载
    if docker buildx build \
        --platform linux/amd64,linux/arm64 \
        -t "${full_image}" \
        -f "${DOCKERFILE}" \
        --push \
        "${BUILD_CONTEXT}"; then
        success "多架构镜像构建并推送成功"
    else
        error "多架构镜像构建失败"
        docker buildx rm multiarch-builder 2>/dev/null || true
        exit 1
    fi

    # 清理构建器
    docker buildx rm multiarch-builder 2>/dev/null || true
}

# 函数：推送镜像
push_image() {
    local full_image="${IMAGE_NAME}:${IMAGE_TAG}"

    if [ -n "${REGISTRY}" ]; then
        # 为镜像添加仓库前缀
        docker tag "${full_image}" "${REGISTRY}/${full_image}"
        full_image="${REGISTRY}/${full_image}"
    fi

    info "推送镜像: ${full_image}"

    if docker push "${full_image}"; then
        success "镜像推送成功"
    else
        error "镜像推送失败"
        exit 1
    fi
}

# 函数：显示构建后的使用说明
show_usage() {
    local full_image="${IMAGE_NAME}:${IMAGE_TAG}"

    if [ -n "${REGISTRY}" ]; then
        full_image="${REGISTRY}/${full_image}"
    fi

    cat << EOF

${GREEN}===========================================${NC}
${GREEN}构建完成！${NC}
${GREEN}===========================================${NC}

镜像: ${full_image}

${BLUE}运行容器:${NC}
  docker run -d -p 8080:8080 --name ai-pro ${full_image}

${BLUE}查看日志:${NC}
  docker logs -f ai-pro

${BLUE}停止容器:${NC}
  docker stop ai-pro && docker rm ai-pro

${BLUE}挂载数据目录:${NC}
  docker run -d -p 8080:8080 \\
    -v \$(pwd)/data:/app/data \\
    -v \$(pwd)/config.yaml:/app/config.yaml:ro \\
    --name ai-pro \\
    ${full_image}

EOF
}

# 解析参数
MULTI_ARCH=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -n|--name)
            IMAGE_NAME="$2"
            shift 2
            ;;
        -t|--tag)
            IMAGE_TAG="$2"
            shift 2
            ;;
        -r|--registry)
            REGISTRY="$2"
            shift 2
            ;;
        -p|--push)
            PUSH=true
            shift
            ;;
        -m|--multi-arch)
            MULTI_ARCH=true
            PUSH=true
            shift
            ;;
        -h|--help)
            show_help
            exit 0
            ;;
        *)
            error "未知参数: $1"
            show_help
            exit 1
            ;;
    esac
done

# 主流程
main() {
    echo -e "${BLUE}"
    echo "============================================"
    echo "  AI-Ops Docker 镜像打包脚本"
    echo "============================================"
    echo -e "${NC}"

    check_dependencies

    if [ "$MULTI_ARCH" = true ]; then
        build_multi_arch
    else
        build_image

        if [ "$PUSH" = true ]; then
            push_image
        fi
    fi

    show_usage
}

main
