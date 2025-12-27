# AI-Ops Makefile
# 兼容 Windows 和 Linux

# 变量定义
APP_NAME := ai-ops
BIN_DIR := bin
MAIN_FILE := cmd/server/main.go
DOCKER_IMAGE := ai-ops
DOCKER_PORT := 8080

# 检测操作系统
ifeq ($(OS),Windows_NT)
    RM := del /Q /F
    RMDIR := rmdir /S /Q
    MKDIR := mkdir
    BIN_EXT := .exe
    SHELL := cmd.exe
    SEP := \\
else
    RM := rm -f
    RMDIR := rm -rf
    MKDIR := mkdir -p
    BIN_EXT :=
    SEP := /
endif

# 默认目标
.DEFAULT_GOAL := help

# ============================================
# 构建命令
# ============================================

## build: 构建后端
.PHONY: build
build:
	@echo Building backend...
	go build -o $(BIN_DIR)$(SEP)$(APP_NAME)$(BIN_EXT) $(MAIN_FILE)
	@echo Build completed: $(BIN_DIR)$(SEP)$(APP_NAME)$(BIN_EXT)

## build-web: 构建前端
.PHONY: build-web
build-web:
	@echo Building frontend...
	cd web && npm install && npm run build
	@echo Frontend build completed

## build-all: 构建全部（后端 + 前端）
.PHONY: build-all
build-all: build build-web
	@echo All builds completed

# ============================================
# 运行命令
# ============================================

## run: 运行开发服务（后端）
.PHONY: run
run:
	@echo Starting backend server...
	go run $(MAIN_FILE)

## run-web: 运行前端开发服务
.PHONY: run-web
run-web:
	@echo Starting frontend dev server...
	cd web && npm run dev

# ============================================
# 测试命令
# ============================================

## test: 运行测试
.PHONY: test
test:
	@echo Running tests...
	go test ./...

## test-v: 运行测试（详细输出）
.PHONY: test-v
test-v:
	@echo Running tests with verbose output...
	go test -v ./...

## test-cover: 运行测试并生成覆盖率报告
.PHONY: test-cover
test-cover:
	@echo Running tests with coverage...
	go test -cover ./...

# ============================================
# Docker 命令
# ============================================

## docker-build: Docker 构建镜像
.PHONY: docker-build
docker-build:
	@echo Building Docker image...
	docker build -t $(DOCKER_IMAGE) .
	@echo Docker image built: $(DOCKER_IMAGE)

## docker-run: Docker 运行容器
.PHONY: docker-run
docker-run:
	@echo Running Docker container...
	docker run -p $(DOCKER_PORT):$(DOCKER_PORT) $(DOCKER_IMAGE)

## docker-run-d: Docker 后台运行容器
.PHONY: docker-run-d
docker-run-d:
	@echo Running Docker container in background...
	docker run -d -p $(DOCKER_PORT):$(DOCKER_PORT) --name $(APP_NAME) $(DOCKER_IMAGE)
	@echo Container started: $(APP_NAME)

## docker-stop: 停止 Docker 容器
.PHONY: docker-stop
docker-stop:
	@echo Stopping Docker container...
	docker stop $(APP_NAME)
	docker rm $(APP_NAME)

## docker-logs: 查看 Docker 容器日志
.PHONY: docker-logs
docker-logs:
	docker logs -f $(APP_NAME)

# ============================================
# 清理命令
# ============================================

## clean: 清理构建产物
.PHONY: clean
clean:
	@echo Cleaning build artifacts...
ifeq ($(OS),Windows_NT)
	-$(RM) $(BIN_DIR)$(SEP)$(APP_NAME)$(BIN_EXT) 2>nul
	-$(RMDIR) $(BIN_DIR) 2>nul
	-$(RMDIR) web$(SEP)dist 2>nul
	-$(RMDIR) web$(SEP)node_modules 2>nul
else
	$(RM) $(BIN_DIR)/$(APP_NAME)
	$(RMDIR) $(BIN_DIR)
	$(RMDIR) web/dist
	$(RMDIR) web/node_modules
endif
	@echo Clean completed

# ============================================
# 开发工具
# ============================================

## fmt: 格式化代码
.PHONY: fmt
fmt:
	@echo Formatting code...
	go fmt ./...

## lint: 代码检查
.PHONY: lint
lint:
	@echo Running linter...
	go vet ./...

## mod-tidy: 整理依赖
.PHONY: mod-tidy
mod-tidy:
	@echo Tidying modules...
	go mod tidy

## mod-download: 下载依赖
.PHONY: mod-download
mod-download:
	@echo Downloading modules...
	go mod download

# ============================================
# 帮助信息
# ============================================

## help: 显示帮助信息
.PHONY: help
help:
	@echo.
	@echo AI-Ops Makefile Commands:
	@echo ============================================
	@echo.
	@echo Build Commands:
	@echo   make build        - Build backend binary
	@echo   make build-web    - Build frontend
	@echo   make build-all    - Build all (backend + frontend)
	@echo.
	@echo Run Commands:
	@echo   make run          - Run backend dev server
	@echo   make run-web      - Run frontend dev server
	@echo.
	@echo Test Commands:
	@echo   make test         - Run tests
	@echo   make test-v       - Run tests (verbose)
	@echo   make test-cover   - Run tests with coverage
	@echo.
	@echo Docker Commands:
	@echo   make docker-build - Build Docker image
	@echo   make docker-run   - Run Docker container
	@echo   make docker-run-d - Run Docker container (detached)
	@echo   make docker-stop  - Stop Docker container
	@echo   make docker-logs  - View Docker logs
	@echo.
	@echo Clean Commands:
	@echo   make clean        - Clean build artifacts
	@echo.
	@echo Dev Tools:
	@echo   make fmt          - Format code
	@echo   make lint         - Run linter
	@echo   make mod-tidy     - Tidy modules
	@echo   make mod-download - Download modules
	@echo.
	@echo   make help         - Show this help
	@echo.
