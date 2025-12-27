# ============================================
# 阶段1: 构建 Go 后端
# ============================================
FROM golang:1.21-alpine AS builder-backend

# 安装必要的构建工具
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

# 复制 go.mod 和 go.sum 先下载依赖（利用缓存）
COPY go.mod go.sum* ./
RUN go mod download

# 复制源代码
COPY . .

# 构建后端二进制文件
RUN CGO_ENABLED=1 GOOS=linux go build -a -ldflags '-linkmode external -extldflags "-static"' -o bin/ai-ops cmd/server/main.go

# ============================================
# 阶段2: 构建 Vue 前端
# ============================================
FROM node:18-alpine AS builder-frontend

WORKDIR /app/web

# 复制前端项目文件
COPY web/package*.json ./

# 安装依赖
RUN npm install

# 复制前端源代码
COPY web/ ./

# 构建前端
RUN npm run build

# ============================================
# 阶段3: 运行阶段
# ============================================
FROM alpine:latest AS runtime

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai

WORKDIR /app

# 从 builder-backend 复制后端二进制文件
COPY --from=builder-backend /app/bin/ai-ops /app/ai-ops

# 从 builder-frontend 复制前端构建产物
COPY --from=builder-frontend /app/web/dist /app/web/dist

# 复制配置文件
COPY config.yaml /app/config.yaml

# 创建数据目录和脚本目录
RUN mkdir -p /app/data /app/scripts

# 暴露端口
EXPOSE 8080

# 设置入口点
ENTRYPOINT ["/app/ai-ops"]
