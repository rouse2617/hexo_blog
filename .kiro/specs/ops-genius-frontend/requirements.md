# Requirements Document

## Introduction

OpsGenius Frontend 是一个基于 React + TypeScript 的智能运维平台前端系统。它为 SRE/运维工程师和开发人员提供对话式界面，通过自然语言与 Agent 后端交互，实时查看 MCP Server 状态，监控 Agent 执行日志，并对高风险操作进行人工确认。

## Glossary

- **OpsGenius_Frontend**: 智能运维平台的前端应用系统
- **MCP_Server**: Model Context Protocol 服务器，提供运维工具集成能力
- **Agent**: 后端智能代理，负责任务解析和工具调用
- **Chat_Interface**: 对话式交互界面组件
- **Log_Viewer**: Agent 执行日志查看器组件
- **Context_Modal**: 上下文日志详情弹窗组件
- **MCP_Status_Panel**: MCP Server 状态面板组件
- **Confirmation_Card**: 人工确认卡片组件
- **User**: 使用系统的 SRE/运维工程师或开发人员
- **Message**: 用户与 Agent 之间的对话消息
- **Log_Entry**: Agent 执行过程中产生的日志条目
- **Agent_Reference**: 日志中对特定 Agent 的引用
- **Streaming_Response**: 流式返回的 Agent 响应内容

## Requirements

### Requirement 1: 对话式交互界面

**User Story:** 作为运维工程师，我希望通过自然语言与系统对话，以便快速完成故障排查和日常巡检任务。

#### Acceptance Criteria

1. WHEN User 输入自然语言消息并提交，THE Chat_Interface SHALL 将消息发送到后端并显示在对话区
2. WHEN Agent 返回响应，THE Chat_Interface SHALL 以流式方式实时展示响应内容
3. WHEN 响应内容包含 Markdown 格式，THE Chat_Interface SHALL 正确渲染 Markdown 语法
4. WHEN 响应内容包含代码块，THE Chat_Interface SHALL 提供语法高亮显示
5. WHEN User 正在输入时，THE Chat_Interface SHALL 禁用提交按钮直到输入非空内容

### Requirement 2: MCP Server 状态监控

**User Story:** 作为运维工程师，我希望实时查看已连接的 MCP Server 状态，以便了解系统可用的工具能力。

#### Acceptance Criteria

1. WHEN OpsGenius_Frontend 启动，THE MCP_Status_Panel SHALL 显示所有已配置的 MCP Server 列表
2. WHEN MCP_Server 连接状态改变，THE MCP_Status_Panel SHALL 更新对应服务器的状态指示器
3. WHEN User 点击某个 MCP_Server，THE MCP_Status_Panel SHALL 展开显示该服务器提供的工具列表
4. WHEN MCP_Server 处于离线状态，THE MCP_Status_Panel SHALL 以红色标识显示
5. WHEN MCP_Server 处于在线状态，THE MCP_Status_Panel SHALL 以绿色标识显示

### Requirement 3: Agent 执行日志查看

**User Story:** 作为运维工程师，我希望查看 Agent 的执行日志，以便了解任务处理过程和调试问题。

#### Acceptance Criteria

1. WHEN Agent 执行任务，THE Log_Viewer SHALL 实时显示新产生的 Log_Entry
2. WHEN Log_Entry 包含 Agent_Reference，THE Log_Viewer SHALL 将引用渲染为可点击的链接
3. WHEN User 点击 Agent_Reference，THE Log_Viewer SHALL 触发显示该 Agent 的上下文日志
4. WHEN Log_Entry 超过 100 条，THE Log_Viewer SHALL 自动滚动到最新日志
5. WHEN User 手动滚动日志区域，THE Log_Viewer SHALL 暂停自动滚动直到用户滚动到底部

### Requirement 4: 上下文日志详情查看

**User Story:** 作为运维工程师，我希望查看特定 Agent 的详细上下文日志，以便深入了解该 Agent 的执行细节。

#### Acceptance Criteria

1. WHEN User 点击 Agent_Reference，THE Context_Modal SHALL 弹出并显示该 Agent 的上下文日志
2. WHEN Context_Modal 显示时，THE Context_Modal SHALL 按时间顺序展示所有上下文日志条目
3. WHEN User 点击关闭按钮或模态框外部区域，THE Context_Modal SHALL 关闭并清空状态
4. WHEN 上下文日志为空，THE Context_Modal SHALL 显示"暂无上下文日志"提示
5. WHEN Context_Modal 打开时，THE OpsGenius_Frontend SHALL 阻止背景内容滚动

### Requirement 5: 人工确认机制

**User Story:** 作为运维工程师，我希望对高风险操作进行人工确认，以便防止误操作导致的生产事故。

#### Acceptance Criteria

1. WHEN Agent 请求执行高风险操作，THE Confirmation_Card SHALL 在对话区显示确认卡片
2. WHEN Confirmation_Card 显示时，THE Confirmation_Card SHALL 包含操作描述、确认按钮和取消按钮
3. WHEN User 点击确认按钮，THE Confirmation_Card SHALL 发送确认信号到后端并显示"已确认"状态
4. WHEN User 点击取消按钮，THE Confirmation_Card SHALL 发送取消信号到后端并显示"已取消"状态
5. WHEN User 未在 30 秒内响应，THE Confirmation_Card SHALL 自动超时并显示"超时未确认"状态

### Requirement 6: 图表数据可视化

**User Story:** 作为运维工程师，我希望以图表形式查看监控指标，以便直观了解系统运行状态。

#### Acceptance Criteria

1. WHEN Agent 返回包含图表数据的响应，THE Chat_Interface SHALL 渲染对应的图表组件
2. WHEN 图表数据为时间序列，THE Chat_Interface SHALL 使用折线图展示
3. WHEN 图表数据为分布统计，THE Chat_Interface SHALL 使用柱状图或饼图展示
4. WHEN User 鼠标悬停在图表数据点上，THE Chat_Interface SHALL 显示详细数值
5. WHEN 图表数据加载失败，THE Chat_Interface SHALL 显示错误提示并提供重试选项

### Requirement 7: 响应式布局

**User Story:** 作为运维工程师，我希望在不同屏幕尺寸下都能正常使用系统，以便在各种设备上进行运维操作。

#### Acceptance Criteria

1. WHEN 屏幕宽度小于 768px，THE OpsGenius_Frontend SHALL 将 MCP_Status_Panel 折叠为抽屉式菜单
2. WHEN 屏幕宽度大于等于 768px，THE OpsGenius_Frontend SHALL 显示三栏布局（左侧面板、中间对话区、右侧日志区）
3. WHEN User 在移动设备上操作，THE OpsGenius_Frontend SHALL 提供触摸友好的交互元素
4. WHEN 布局切换时，THE OpsGenius_Frontend SHALL 保持用户当前的操作状态
5. WHEN 窗口大小改变，THE OpsGenius_Frontend SHALL 平滑过渡到新布局

### Requirement 8: 会话历史管理

**User Story:** 作为运维工程师，我希望查看和恢复历史会话，以便回顾之前的故障排查过程。

#### Acceptance Criteria

1. WHEN User 发送消息，THE OpsGenius_Frontend SHALL 将会话数据保存到本地存储
2. WHEN User 刷新页面，THE OpsGenius_Frontend SHALL 恢复最近的会话内容
3. WHEN User 点击"新建会话"按钮，THE OpsGenius_Frontend SHALL 清空当前对话并创建新会话
4. WHEN User 点击"历史会话"按钮，THE OpsGenius_Frontend SHALL 显示最近 10 条会话列表
5. WHEN User 选择历史会话，THE OpsGenius_Frontend SHALL 加载并显示该会话的完整内容

### Requirement 9: 错误处理与提示

**User Story:** 作为运维工程师，我希望系统能清晰地提示错误信息，以便快速定位和解决问题。

#### Acceptance Criteria

1. WHEN 后端连接失败，THE OpsGenius_Frontend SHALL 显示连接错误提示并提供重连按钮
2. WHEN 消息发送失败，THE OpsGenius_Frontend SHALL 在消息旁显示失败标识并提供重试选项
3. WHEN MCP_Server 调用超时，THE OpsGenius_Frontend SHALL 在日志中显示超时警告
4. WHEN 发生未预期错误，THE OpsGenius_Frontend SHALL 显示友好的错误提示而非技术堆栈
5. WHEN 错误可恢复，THE OpsGenius_Frontend SHALL 自动重试最多 3 次

### Requirement 10: 性能优化

**User Story:** 作为运维工程师，我希望系统响应迅速，以便在紧急情况下快速获取信息。

#### Acceptance Criteria

1. WHEN OpsGenius_Frontend 首次加载，THE OpsGenius_Frontend SHALL 在 2 秒内完成首屏渲染
2. WHEN User 发送消息，THE Chat_Interface SHALL 在 100ms 内显示消息并开始等待响应
3. WHEN Agent 开始返回流式响应，THE Chat_Interface SHALL 在 200ms 内显示首字内容
4. WHEN Log_Entry 数量超过 1000 条，THE Log_Viewer SHALL 使用虚拟滚动优化渲染性能
5. WHEN 图表数据点超过 500 个，THE Chat_Interface SHALL 对数据进行采样降低渲染负担
