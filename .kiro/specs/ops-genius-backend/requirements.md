# Requirements Document

## Introduction

OpsGenius Backend 是一个基于 Go 语言的智能运维平台后端系统。它通过 LangChain/LangGraph 集成大语言模型，实现自然语言意图识别和任务编排，通过 MCP (Model Context Protocol) 协议连接各种运维工具（日志、监控、配置管理），并通过 WebSocket 与前端实时通信。

## Glossary

- **OpsGenius_Backend**: 智能运维平台的后端服务系统
- **Agent_Engine**: 基于 LLM 的智能代理引擎，负责意图识别和任务编排
- **MCP_Client**: MCP 协议客户端，用于连接和调用 MCP Server
- **MCP_Server**: 提供运维工具能力的外部服务（如 Log-MCP、Metric-MCP、K8s-MCP）
- **WebSocket_Server**: WebSocket 服务器，负责与前端实时通信
- **Task_Orchestrator**: 任务编排器，负责多步骤工作流管理
- **Intent_Parser**: 意图解析器，从自然语言中提取结构化信息
- **Session**: 用户会话，包含对话历史和上下文
- **Tool_Call**: 对 MCP Server 工具的调用请求
- **Confirmation_Manager**: 确认管理器，处理高风险操作的人工确认流程
- **Log_Collector**: 日志收集器，收集 Agent 执行日志

## Requirements

### Requirement 1: WebSocket 实时通信

**User Story:** 作为后端系统，我需要通过 WebSocket 与前端实时通信，以便提供流式响应和实时状态更新。

#### Acceptance Criteria

1. WHEN 前端客户端连接，THE WebSocket_Server SHALL 建立连接并分配唯一的连接 ID
2. WHEN 接收到客户端消息，THE WebSocket_Server SHALL 解析消息类型并路由到对应的处理器
3. WHEN 需要发送消息到前端，THE WebSocket_Server SHALL 序列化消息并通过 WebSocket 连接发送
4. WHEN 客户端断开连接，THE WebSocket_Server SHALL 清理连接资源并保存会话状态
5. WHEN 连接空闲超过 5 分钟，THE WebSocket_Server SHALL 发送心跳消息检测连接状态

### Requirement 2: 自然语言意图识别

**User Story:** 作为 Agent 引擎，我需要识别用户的自然语言意图，以便准确理解用户需求并执行相应操作。

#### Acceptance Criteria

1. WHEN 接收到用户消息，THE Intent_Parser SHALL 调用 LLM 提取意图类型（查询、分析、操作等）
2. WHEN 意图包含实体信息，THE Intent_Parser SHALL 提取实体（节点名、服务名、时间范围等）
3. WHEN 意图信息不完整，THE Intent_Parser SHALL 识别缺失的槽位并生成反问
4. WHEN 意图识别完成，THE Intent_Parser SHALL 返回结构化的意图对象
5. WHEN 意图识别失败，THE Intent_Parser SHALL 返回错误并请求用户重新表述

### Requirement 3: MCP Server 连接管理

**User Story:** 作为后端系统，我需要管理与多个 MCP Server 的连接，以便提供丰富的运维工具能力。

#### Acceptance Criteria

1. WHEN 系统启动，THE MCP_Client SHALL 读取配置并连接所有已配置的 MCP Server
2. WHEN MCP_Server 连接成功，THE MCP_Client SHALL 获取该服务器提供的工具列表
3. WHEN MCP_Server 连接失败，THE MCP_Client SHALL 记录错误并定期重试连接
4. WHEN MCP_Server 断开连接，THE MCP_Client SHALL 更新状态并通知前端
5. WHEN 需要调用工具，THE MCP_Client SHALL 选择对应的 MCP_Server 并发送调用请求

### Requirement 4: MCP 工具调用

**User Story:** 作为 Agent 引擎，我需要调用 MCP Server 提供的工具，以便执行具体的运维操作。

#### Acceptance Criteria

1. WHEN Agent 决定调用工具，THE MCP_Client SHALL 构造工具调用请求并发送到对应的 MCP_Server
2. WHEN 工具调用成功，THE MCP_Client SHALL 返回工具执行结果
3. WHEN 工具调用失败，THE MCP_Client SHALL 返回错误信息并记录日志
4. WHEN 工具调用超时（30 秒），THE MCP_Client SHALL 取消请求并返回超时错误
5. WHEN 工具为只读操作，THE MCP_Client SHALL 直接执行；WHEN 工具为写操作，THE MCP_Client SHALL 请求人工确认

### Requirement 5: 任务编排和工作流

**User Story:** 作为 Agent 引擎，我需要编排多步骤任务，以便处理复杂的运维场景。

#### Acceptance Criteria

1. WHEN 接收到复杂任务，THE Task_Orchestrator SHALL 将任务分解为多个子任务
2. WHEN 子任务有依赖关系，THE Task_Orchestrator SHALL 按依赖顺序执行子任务
3. WHEN 子任务执行失败，THE Task_Orchestrator SHALL 根据策略决定是否继续或回滚
4. WHEN 子任务执行成功，THE Task_Orchestrator SHALL 将结果传递给下一个子任务
5. WHEN 所有子任务完成，THE Task_Orchestrator SHALL 汇总结果并生成最终响应

### Requirement 6: 流式响应生成

**User Story:** 作为 Agent 引擎，我需要生成流式响应，以便用户能实时看到分析过程。

#### Acceptance Criteria

1. WHEN Agent 开始生成响应，THE Agent_Engine SHALL 以流式方式发送响应片段到前端
2. WHEN 响应包含工具调用结果，THE Agent_Engine SHALL 在响应中嵌入结构化数据
3. WHEN 响应包含图表数据，THE Agent_Engine SHALL 发送单独的图表数据消息
4. WHEN 响应生成完成，THE Agent_Engine SHALL 发送完成标记
5. WHEN 响应生成过程中出错，THE Agent_Engine SHALL 发送错误消息并终止流式响应

### Requirement 7: 人工确认机制

**User Story:** 作为后端系统，我需要对高风险操作请求人工确认，以便防止误操作。

#### Acceptance Criteria

1. WHEN Agent 需要执行高风险操作，THE Confirmation_Manager SHALL 创建确认请求并发送到前端
2. WHEN 确认请求创建，THE Confirmation_Manager SHALL 设置 30 秒超时定时器
3. WHEN 接收到用户确认，THE Confirmation_Manager SHALL 取消超时定时器并执行操作
4. WHEN 接收到用户取消，THE Confirmation_Manager SHALL 取消超时定时器并终止操作
5. WHEN 确认请求超时，THE Confirmation_Manager SHALL 自动取消操作并通知前端

### Requirement 8: 会话管理

**User Story:** 作为后端系统，我需要管理用户会话，以便维护对话上下文和历史。

#### Acceptance Criteria

1. WHEN 新客户端连接，THE OpsGenius_Backend SHALL 创建新会话或恢复已有会话
2. WHEN 接收到用户消息，THE OpsGenius_Backend SHALL 将消息添加到会话历史
3. WHEN Agent 生成响应，THE OpsGenius_Backend SHALL 将响应添加到会话历史
4. WHEN 会话超过 1 小时无活动，THE OpsGenius_Backend SHALL 将会话标记为过期
5. WHEN 会话数据需要持久化，THE OpsGenius_Backend SHALL 将会话保存到数据库

### Requirement 9: 日志收集和上报

**User Story:** 作为后端系统，我需要收集 Agent 执行日志，以便前端展示和问题排查。

#### Acceptance Criteria

1. WHEN Agent 开始执行任务，THE Log_Collector SHALL 创建任务日志条目
2. WHEN Agent 调用工具，THE Log_Collector SHALL 记录工具调用日志
3. WHEN Agent 生成中间结果，THE Log_Collector SHALL 记录分析日志
4. WHEN 日志条目创建，THE Log_Collector SHALL 通过 WebSocket 实时发送到前端
5. WHEN 日志包含 Agent 引用，THE Log_Collector SHALL 在日志中标记引用信息

### Requirement 10: 错误处理和恢复

**User Story:** 作为后端系统，我需要优雅地处理错误，以便系统能稳定运行。

#### Acceptance Criteria

1. WHEN LLM 调用失败，THE Agent_Engine SHALL 重试最多 3 次，使用指数退避策略
2. WHEN MCP_Server 调用失败，THE Agent_Engine SHALL 记录错误并尝试使用备用工具
3. WHEN 发生 panic，THE OpsGenius_Backend SHALL 捕获 panic 并记录堆栈信息
4. WHEN 数据库操作失败，THE OpsGenius_Backend SHALL 降级为内存存储
5. WHEN 系统资源不足，THE OpsGenius_Backend SHALL 拒绝新连接并返回友好错误

### Requirement 11: 配置管理

**User Story:** 作为系统管理员，我需要通过配置文件管理系统行为，以便灵活调整系统参数。

#### Acceptance Criteria

1. WHEN 系统启动，THE OpsGenius_Backend SHALL 读取配置文件（YAML 或 JSON）
2. WHEN 配置文件包含 LLM 配置，THE OpsGenius_Backend SHALL 初始化 LLM 客户端
3. WHEN 配置文件包含 MCP Server 列表，THE OpsGenius_Backend SHALL 连接所有配置的服务器
4. WHEN 配置文件包含安全设置，THE OpsGenius_Backend SHALL 应用安全策略
5. WHEN 配置文件格式错误，THE OpsGenius_Backend SHALL 拒绝启动并输出详细错误信息

### Requirement 12: 安全性

**User Story:** 作为系统管理员，我需要确保系统安全，以便保护敏感数据和防止未授权访问。

#### Acceptance Criteria

1. WHEN 客户端连接，THE WebSocket_Server SHALL 验证客户端身份（Token 或证书）
2. WHEN 传输敏感数据，THE OpsGenius_Backend SHALL 使用加密连接（WSS）
3. WHEN 调用 MCP_Server，THE OpsGenius_Backend SHALL 不在 LLM 上下文中包含敏感凭证
4. WHEN 记录日志，THE OpsGenius_Backend SHALL 脱敏敏感信息（密码、Token 等）
5. WHEN 执行写操作，THE OpsGenius_Backend SHALL 验证用户权限

### Requirement 13: 性能和可扩展性

**User Story:** 作为系统管理员，我需要系统具有良好的性能和可扩展性，以便支持大规模使用。

#### Acceptance Criteria

1. WHEN 系统运行，THE OpsGenius_Backend SHALL 支持至少 1000 个并发 WebSocket 连接
2. WHEN 处理用户请求，THE Agent_Engine SHALL 在 2 秒内返回首字响应
3. WHEN 调用 MCP_Server，THE MCP_Client SHALL 使用连接池复用连接
4. WHEN 系统负载高，THE OpsGenius_Backend SHALL 使用 goroutine 池限制并发数
5. WHEN 需要水平扩展，THE OpsGenius_Backend SHALL 支持多实例部署（通过 Redis 共享会话）
