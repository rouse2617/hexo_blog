# Implementation Plan: OpsGenius Backend

## Overview

本实现计划将 OpsGenius Backend 分解为可执行的开发任务。我们将采用增量开发方式，先搭建基础架构和核心组件，然后逐步添加高级功能。每个任务都包含明确的实现目标和对应的需求引用。

## Tasks

- [x] 1. 项目初始化和基础架构搭建
  - 初始化 Go 模块（go mod init）
  - 安装依赖：gorilla/websocket、langchaingo、viper、zap、testify、gopter
  - 创建基础目录结构（cmd、internal、pkg）
  - 配置 .gitignore 和 Makefile
  - _Requirements: 所有需求的基础_

- [x] 2. 实现配置管理
  - [x] 2.1 创建配置模型和加载器
    - 实现 `internal/config/config.go` 定义配置结构
    - 使用 viper 支持 YAML/JSON/ENV 配置
    - 实现配置验证逻辑
    - _Requirements: 11.1, 11.2, 11.3, 11.4_

  - [x] 2.2 编写配置加载的单元测试
    - 测试有效配置加载
    - 测试配置格式错误处理
    - _Requirements: 11.1, 11.5_

  - [x] 2.3 编写配置加载的属性测试
    - **Property 31: 配置文件解析正确性**
    - **Validates: Requirements 11.1**

- [x] 3. 实现数据模型
  - 创建 `pkg/models/message.go` 定义 WebSocket 消息类型
  - 创建 `pkg/models/session.go` 定义会话模型
  - 创建 `pkg/models/log.go` 定义日志模型
  - 创建 `pkg/models/mcp.go` 定义 MCP 协议模型
  - _Requirements: 1.1, 2.1, 8.1, 9.1_

- [x] 4. 实现 WebSocket 服务器
  - [x] 4.1 创建 WebSocket 服务器核心
    - 实现 `internal/websocket/server.go` WebSocket 服务器
    - 实现连接管理和 ID 分配
    - 实现消息接收和发送
    - _Requirements: 1.1, 1.2, 1.3_

  - [x] 4.2 编写 WebSocket 服务器的单元测试
    - 测试连接建立和断开
    - 测试消息解析错误处理
    - _Requirements: 1.1, 1.4_

  - [x] 4.3 编写 WebSocket 服务器的属性测试
    - **Property 1: WebSocket 连接 ID 唯一性**
    - **Property 3: 消息序列化往返一致性**
    - **Validates: Requirements 1.1, 1.3**

  - [x] 4.4 实现消息路由和处理器
    - 实现 `internal/websocket/handler.go` 消息处理器
    - 根据消息类型路由到对应处理函数
    - _Requirements: 1.2_

  - [x] 4.5 编写消息路由的属性测试
    - **Property 2: 消息类型路由正确性**
    - **Validates: Requirements 1.2**

  - [x] 4.6 实现连接资源管理
    - 实现连接断开时的资源清理
    - 实现心跳机制（5 分钟超时）
    - _Requirements: 1.4, 1.5_

  - [x] 4.7 编写资源管理的属性测试
    - **Property 4: 连接断开资源清理**
    - **Validates: Requirements 1.4**

- [x] 5. Checkpoint - 确保 WebSocket 基础功能完整
  - 确保所有测试通过，如有问题请询问用户

- [x] 6. 实现数据库和缓存层
  - [ ] 6.1 实现 PostgreSQL 连接和会话存储
    - 实现 `internal/session/store.go` 会话持久化
    - 创建数据库 schema 和迁移脚本
    - 实现 CRUD 操作
    - _Requirements: 8.1, 8.5_

  - [ ] 6.2 实现 Redis 缓存
    - 实现 `internal/session/cache.go` 会话缓存
    - 实现缓存读写和过期策略
    - _Requirements: 8.4, 8.5_

  - [ ] 6.3 编写会话存储的属性测试
    - **Property 28: 会话数据持久化往返一致性**
    - **Validates: Requirements 8.5**

  - [ ] 6.4 实现降级策略
    - 实现数据库失败时降级到 Redis
    - 实现 Redis 失败时降级到内存
    - _Requirements: 10.4_

- [x] 7. 实现会话管理
  - [x] 7.1 创建会话管理器
    - 实现 `internal/session/manager.go` 会话管理
    - 实现会话创建、获取、更新、删除
    - 实现会话过期检查（1 小时）
    - _Requirements: 8.1, 8.2, 8.3, 8.4_

  - [x] 7.2 编写会话管理的单元测试
    - 测试会话过期逻辑
    - 测试会话恢复
    - _Requirements: 8.1, 8.4_

  - [x] 7.3 编写会话管理的属性测试
    - **Property 26: 会话创建或恢复**
    - **Property 27: 消息追加到会话历史**
    - **Validates: Requirements 8.1, 8.2, 8.3**

- [x] 8. 实现 MCP 客户端
  - [x] 8.1 实现 MCP 协议层
    - 实现 `internal/mcp/protocol.go` JSON-RPC 2.0 协议
    - 实现请求/响应序列化和反序列化
    - _Requirements: 3.1, 4.1_

  - [x] 8.2 编写 MCP 协议的属性测试
    - **Property 13: 工具调用请求格式正确性**
    - **Validates: Requirements 4.1**

  - [x] 8.3 实现 MCP 客户端核心
    - 实现 `internal/mcp/client.go` MCP 客户端
    - 支持 stdio 和 HTTP 两种传输方式
    - 实现连接、断开、工具列表获取
    - _Requirements: 3.1, 3.2, 3.4_

  - [x] 8.4 编写 MCP 客户端的单元测试
    - 测试连接失败处理
    - 测试工具调用超时
    - _Requirements: 3.3, 4.4_

  - [x] 8.5 编写 MCP 客户端的属性测试
    - **Property 10: 工具列表获取**
    - **Property 14: 工具调用结果返回**
    - **Validates: Requirements 3.2, 4.2**

  - [x] 8.6 实现工具调用功能
    - 实现 `internal/mcp/tool.go` 工具调用逻辑
    - 实现超时控制（30 秒）
    - 实现错误处理和日志记录
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [ ] 8.7 编写工具调用的属性测试
    - **Property 15: 只读/写操作权限控制**
    - **Validates: Requirements 4.5**

- [ ] 9. 实现 MCP 连接管理器
  - [ ] 9.1 创建 MCP 管理器
    - 实现 `internal/mcp/manager.go` MCP 连接管理
    - 实现多服务器连接管理
    - 实现连接池和复用
    - 实现状态广播
    - _Requirements: 3.1, 3.4, 3.5, 13.3_

  - [x] 9.2 编写 MCP 管理器的单元测试
    - 测试服务器添加和移除
    - 测试重连机制
    - _Requirements: 3.1, 3.3_

  - [ ] 9.3 编写 MCP 管理器的属性测试
    - **Property 9: MCP Server 连接初始化**
    - **Property 11: MCP 断开状态更新**
    - **Property 12: 工具调用路由正确性**
    - **Property 40: MCP 连接池复用**
    - **Validates: Requirements 3.1, 3.4, 3.5, 13.3**

- [x] 10. Checkpoint - 确保 MCP 集成功能完整
  - 确保所有测试通过，如有问题请询问用户

- [ ] 11. 实现 LLM 集成和意图解析
  - [ ] 11.1 初始化 LLM 客户端
    - 实现 `internal/agent/llm.go` LLM 客户端封装
    - 支持 OpenAI、Anthropic 等多种 Provider
    - 实现重试机制（指数退避，最多 3 次）
    - _Requirements: 11.2, 10.1_

  - [ ] 11.2 编写 LLM 客户端的单元测试
    - 测试 LLM 调用失败重试
    - 测试不同 Provider 初始化
    - _Requirements: 10.1, 11.2_

  - [ ] 11.3 编写 LLM 客户端的属性测试
    - **Property 32: LLM 客户端初始化**
    - **Validates: Requirements 11.2**

  - [ ] 11.4 实现意图解析器
    - 实现 `internal/agent/intent.go` 意图解析
    - 使用 LLM 提取意图类型和实体
    - 识别缺失槽位并生成反问
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [ ] 11.5 编写意图解析器的单元测试
    - 测试意图识别失败处理
    - 测试实体提取
    - _Requirements: 2.2, 2.5_

  - [ ] 11.6 编写意图解析器的属性测试
    - **Property 5: 意图类型有效性**
    - **Property 6: 实体提取完整性**
    - **Property 7: 缺失槽位识别**
    - **Property 8: 意图对象结构完整性**
    - **Validates: Requirements 2.1, 2.2, 2.3, 2.4**

- [ ] 12. 实现任务编排器
  - [ ] 12.1 创建任务编排器
    - 实现 `internal/agent/orchestrator.go` 任务编排
    - 实现任务分解逻辑
    - 实现依赖关系管理
    - 实现子任务执行和结果传递
    - _Requirements: 5.1, 5.2, 5.4, 5.5_

  - [x] 12.2 编写任务编排器的单元测试
    - 测试子任务执行失败处理
    - 测试依赖顺序
    - _Requirements: 5.2, 5.3_

  - [x] 12.3 编写任务编排器的属性测试
    - **Property 16: 任务分解非平凡性**
    - **Property 17: 子任务执行顺序正确性**
    - **Property 18: 子任务结果传递**
    - **Property 19: 任务结果汇总完整性**
    - **Validates: Requirements 5.1, 5.2, 5.4, 5.5**

- [x] 13. 实现 Agent 引擎
  - [x] 13.1 创建 Agent 引擎核心
    - 实现 `internal/agent/engine.go` Agent 引擎
    - 整合意图解析、任务编排、MCP 调用
    - 实现流式响应生成
    - _Requirements: 6.1, 6.2, 6.3, 6.4_

  - [x] 13.2 编写 Agent 引擎的单元测试
    - 测试流式响应生成
    - 测试响应生成错误处理
    - _Requirements: 6.1, 6.5_

  - [x] 13.3 编写 Agent 引擎的属性测试
    - **Property 20: 流式响应片段发送**
    - **Property 21: 工具结果嵌入**
    - **Property 22: 图表数据单独发送**
    - **Property 23: 流式响应完成标记**
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4**

  - [x] 13.4 实现流式响应处理
    - 实现 `internal/agent/stream.go` 流式响应
    - 实现响应片段生成和发送
    - 实现图表数据提取和发送
    - _Requirements: 6.1, 6.2, 6.3_

- [x] 14. 实现确认管理器
  - [x] 14.1 创建确认管理器
    - 实现 `internal/confirmation/manager.go` 确认管理
    - 实现确认请求创建和发送
    - 实现超时机制（30 秒）
    - 实现确认/取消响应处理
    - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

  - [x] 14.2 编写确认管理器的单元测试
    - 测试确认超时处理
    - 测试确认/取消响应
    - _Requirements: 7.2, 7.5_

  - [x] 14.3 编写确认管理器的属性测试
    - **Property 24: 确认请求创建和发送**
    - **Property 25: 确认响应处理正确性**
    - **Validates: Requirements 7.1, 7.3, 7.4**

- [x] 15. Checkpoint - 确保 Agent 核心功能完整
  - 确保所有测试通过，如有问题请询问用户

- [ ] 16. 实现日志收集器
  - [ ] 16.1 创建日志收集器
    - 实现 `internal/logger/collector.go` 日志收集
    - 实现日志条目创建和存储
    - 实现 Agent 引用标记
    - _Requirements: 9.1, 9.2, 9.3, 9.5_

  - [ ] 16.2 实现日志上报器
    - 实现 `internal/logger/reporter.go` 日志上报
    - 通过 WebSocket 实时发送日志到前端
    - 实现日志订阅机制
    - _Requirements: 9.4_

  - [ ] 16.3 编写日志收集器的属性测试
    - **Property 29: 日志条目创建和发送**
    - **Property 30: Agent 引用标记**
    - **Validates: Requirements 9.1, 9.2, 9.3, 9.4, 9.5**

- [ ] 17. 实现安全功能
  - [ ] 17.1 实现身份验证
    - 实现 JWT Token 验证
    - 实现 WebSocket 连接身份验证
    - _Requirements: 12.1_

  - [ ] 17.2 编写身份验证的属性测试
    - **Property 35: 客户端身份验证**
    - **Validates: Requirements 12.1**

  - [ ] 17.3 实现 TLS 加密
    - 配置 WSS（WebSocket Secure）
    - 实现证书加载和验证
    - _Requirements: 12.2_

  - [ ] 17.4 编写加密连接的属性测试
    - **Property 36: 加密连接使用**
    - **Validates: Requirements 12.2**

  - [ ] 17.5 实现敏感数据保护
    - 实现 LLM 上下文凭证过滤
    - 实现日志敏感信息脱敏
    - 实现权限验证
    - _Requirements: 12.3, 12.4, 12.5_

  - [ ] 17.6 编写敏感数据保护的属性测试
    - **Property 37: LLM 上下文凭证保护**
    - **Property 38: 日志敏感信息脱敏**
    - **Property 39: 写操作权限验证**
    - **Validates: Requirements 12.3, 12.4, 12.5**

- [ ] 18. 实现性能优化和资源管理
  - [ ] 18.1 实现 Goroutine 池
    - 使用 worker pool 限制并发数
    - 实现任务队列
    - _Requirements: 13.4_

  - [ ] 18.2 编写 Goroutine 池的属性测试
    - **Property 41: Goroutine 并发数限制**
    - **Validates: Requirements 13.4**

  - [ ] 18.3 实现资源限制器
    - 实现连接数限制
    - 实现内存使用监控
    - 实现资源耗尽时的拒绝策略
    - _Requirements: 10.5, 13.1_

  - [ ] 18.4 编写资源限制器的单元测试
    - 测试连接数限制
    - 测试资源耗尽处理
    - _Requirements: 10.5_

- [ ] 19. 实现错误处理和恢复
  - 在所有 goroutine 中添加 panic 恢复
  - 实现全局错误处理中间件
  - 实现错误日志记录
  - _Requirements: 10.3_

- [ ] 20. 组装主应用
  - [ ] 20.1 实现应用入口
    - 实现 `cmd/server/main.go` 应用入口
    - 初始化所有组件
    - 实现优雅关闭
    - _Requirements: 所有需求_

  - [ ] 20.2 实现健康检查和指标
    - 使用 gin 实现 HTTP 服务器
    - 实现 /health 健康检查端点
    - 实现 /metrics Prometheus 指标暴露
    - _Requirements: 13.1_

- [ ] 21. 集成测试
  - [ ] 21.1 编写端到端集成测试
    - 测试 WebSocket 到 Agent 到 MCP 完整流程
    - 测试会话持久化和恢复
    - 测试确认流程
    - 使用 testcontainers 启动依赖服务
    - _Requirements: 所有需求_

  - [ ] 21.2 编写 MCP 集成测试
    - 测试与真实 MCP Server 的交互
    - 测试工具调用和结果处理
    - _Requirements: 3.1, 4.1, 4.2_

- [ ] 22. 最终 Checkpoint - 确保所有功能完整且测试通过
  - 运行所有测试（单元测试 + 属性测试 + 集成测试）
  - 检查测试覆盖率（目标：语句 > 80%，分支 > 75%，函数 > 85%）
  - 运行性能基准测试
  - 如有问题请询问用户


- [ ] 23. 容器化和部署
  - [ ] 23.1 创建 Dockerfile
    - 使用多阶段构建优化镜像大小
    - 第一阶段：编译 Go 应用
    - 第二阶段：使用 alpine 作为运行时基础镜像
    - 配置非 root 用户运行
    - _Requirements: 13.5_

  - [ ] 23.2 创建 docker-compose.yml
    - 定义 backend 服务
    - 定义 PostgreSQL 服务
    - 定义 Redis 服务
    - 配置服务间网络
    - 配置数据卷持久化
    - _Requirements: 所有需求_

  - [ ] 23.3 创建 Kubernetes 部署配置
    - 创建 Deployment YAML（backend）
    - 创建 Service YAML（WebSocket 服务）
    - 创建 ConfigMap YAML（配置文件）
    - 创建 Secret YAML（敏感配置）
    - 创建 StatefulSet YAML（PostgreSQL）
    - 创建 StatefulSet YAML（Redis）
    - 创建 Ingress YAML（外部访问）
    - _Requirements: 13.5_

  - [ ] 23.4 创建 Helm Chart
    - 创建 Chart.yaml
    - 创建 values.yaml（可配置参数）
    - 创建模板文件
    - 支持多环境部署（dev/staging/prod）
    - _Requirements: 13.5_

  - [ ] 23.5 编写部署文档
    - 创建 docs/deployment.md
    - 文档包含 Docker 部署步骤
    - 文档包含 Kubernetes 部署步骤
    - 文档包含 Helm 部署步骤
    - 文档包含环境变量配置说明
    - _Requirements: 所有需求_

- [ ] 24. 最终验证和文档
  - [ ] 24.1 端到端测试验证
    - 使用 Docker Compose 启动完整环境
    - 运行所有集成测试
    - 验证所有功能正常工作
    - _Requirements: 所有需求_

  - [ ] 24.2 性能测试
    - 使用 k6 或 wrk 进行压力测试
    - 验证支持 1000+ 并发连接
    - 验证响应时间满足要求（首字响应 < 2s）
    - _Requirements: 13.1, 13.2_

  - [ ] 24.3 编写 README.md
    - 项目介绍
    - 快速开始指南
    - 配置说明
    - API 文档
    - 部署指南链接
    - _Requirements: 所有需求_

  - [ ] 24.4 编写 API 文档
    - WebSocket 消息协议文档
    - MCP 集成指南
    - 配置文件参考
    - 故障排查指南
    - _Requirements: 所有需求_

## Notes

- 所有任务都是必需的，确保全面的测试覆盖和生产就绪
- 每个任务都引用了具体的需求编号，便于追溯
- Checkpoint 任务确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况
- 集成测试使用 testcontainers 确保环境一致性
- 容器化部署支持 Docker、Docker Compose、Kubernetes 和 Helm
- 多阶段 Docker 构建优化镜像大小
- Kubernetes 配置支持水平扩展和高可用
