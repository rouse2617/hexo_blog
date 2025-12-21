# Implementation Plan: OpsGenius Frontend

## Overview

本实现计划将 OpsGenius Frontend 分解为可执行的开发任务。我们将采用增量开发方式，先搭建基础架构和核心组件，然后逐步添加高级功能。每个任务都包含明确的实现目标和对应的需求引用。

## Tasks

- [x] 1. 安装缺失的依赖包
  - 安装 Tailwind CSS 及相关插件（tailwindcss、postcss、autoprefixer）
  - 安装 Recharts 图表库
  - 安装 react-markdown、remark-gfm 用于 Markdown 渲染
  - 安装 Prism.js 及 React 集成（prismjs、react-syntax-highlighter）
  - 配置 Tailwind CSS（创建 tailwind.config.js 和 postcss.config.js）
  - _Requirements: 所有需求的基础_

- [x] 2. 实现数据模型和类型定义
  - 创建 `types/models.ts` 定义所有接口（Message, MCPServer, LogEntry, ConfirmationRequest 等）
  - 创建 `types/websocket.ts` 定义 WebSocket 消息协议类型
  - _Requirements: 1.1, 2.1, 3.1, 5.1_

- [x] 3. 实现 WebSocket 通信层
  - [x] 3.1 创建 WebSocket 管理器
    - 实现 `utils/websocket.ts` 包含连接、重连、消息发送/接收逻辑
    - 实现指数退避重连策略
    - _Requirements: 9.1, 9.5_

  - [x]* 3.2 编写 WebSocket 管理器的属性测试
    - **Property 23: 会话数据往返一致性**
    - **Validates: Requirements 8.1, 8.2**

- [x] 4. 实现全局状态管理
  - [x] 4.1 创建 Zustand store
    - 实现 `stores/appStore.ts` 包含会话、消息、MCP 状态、日志等状态管理
    - 实现状态更新 actions
    - _Requirements: 1.1, 2.1, 3.1, 5.1, 8.1_

  - [x]* 4.2 编写状态管理的属性测试
    - **Property 24: 新会话创建隔离性**
    - **Validates: Requirements 8.3**

- [x] 5. 实现本地存储功能
  - [x] 5.1 创建本地存储工具
    - 实现 `utils/storage.ts` 包含会话保存、加载、清理功能
    - 处理 localStorage 异常（配额超限等）
    - _Requirements: 8.1, 8.2, 8.5_

  - [x]* 5.2 编写本地存储的属性测试
    - **Property 23: 会话数据往返一致性**
    - **Validates: Requirements 8.1, 8.2**

- [x] 6. Checkpoint - 确保基础架构测试通过
  - 确保所有测试通过，如有问题请询问用户

- [x] 7. 实现 Layout 组件
  - 创建 `components/Layout/Layout.tsx` 实现三栏响应式布局
  - 实现移动端抽屉式菜单
  - 使用 Tailwind CSS 实现响应式样式
  - _Requirements: 7.1, 7.2, 7.4_

- [x]* 7.1 编写 Layout 的属性测试
  - **Property 22: 布局切换状态保持**
  - **Validates: Requirements 7.4**

- [x] 8. 实现 MCP Status Panel 组件
  - [x] 8.1 创建 MCPStatusPanel 组件
    - 实现 `components/MCPStatusPanel/MCPStatusPanel.tsx`
    - 显示 MCP Server 列表和状态指示器
    - 实现展开/折叠工具列表功能
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

  - [x]* 8.2 编写 MCPStatusPanel 的单元测试
    - 测试服务器列表渲染
    - 测试展开/折叠交互
    - _Requirements: 2.1, 2.3_

  - [x]* 8.3 编写 MCPStatusPanel 的属性测试
    - **Property 6: MCP Server 列表完整性**
    - **Property 9: 状态颜色映射正确性**
    - **Validates: Requirements 2.1, 2.4, 2.5**

- [x] 9. 实现 Message 组件
  - [x] 9.1 创建 UserMessage 和 AgentMessage 组件
    - 实现 `components/ChatInterface/UserMessage.tsx`
    - 实现 `components/ChatInterface/AgentMessage.tsx`
    - 支持流式内容显示
    - _Requirements: 1.1, 1.2_

  - [x] 9.2 创建 MarkdownRenderer 组件
    - 实现 `components/ChatInterface/MarkdownRenderer.tsx`
    - 集成 react-markdown 和 remark-gfm
    - _Requirements: 1.3_

  - [x]* 9.3 编写 MarkdownRenderer 的属性测试
    - **Property 3: Markdown 渲染正确性**
    - **Validates: Requirements 1.3**

  - [x] 9.4 创建 CodeBlock 组件
    - 实现 `components/ChatInterface/CodeBlock.tsx`
    - 集成语法高亮（使用 react-syntax-highlighter）
    - _Requirements: 1.4_

  - [x]* 9.5 编写 CodeBlock 的属性测试
    - **Property 4: 代码块高亮完整性**
    - **Validates: Requirements 1.4**

- [x] 10. 实现 ChatInterface 组件
  - [x] 10.1 创建 MessageInput 组件
    - 实现 `components/ChatInterface/MessageInput.tsx`
    - 实现输入框和提交按钮
    - 控制按钮禁用状态
    - _Requirements: 1.1, 1.5_

  - [x]* 10.2 编写 MessageInput 的属性测试
    - **Property 5: 提交按钮状态控制**
    - **Validates: Requirements 1.5**

  - [x] 10.3 创建 MessageList 组件
    - 实现 `components/ChatInterface/MessageList.tsx`
    - 渲染消息列表
    - 实现自动滚动到底部
    - _Requirements: 1.1, 1.2_

  - [x]* 10.4 编写 MessageList 的属性测试
    - **Property 1: 消息提交完整性**
    - **Property 2: 流式响应实时性**
    - **Validates: Requirements 1.1, 1.2**

  - [x] 10.5 组装 ChatInterface 主组件
    - 实现 `components/ChatInterface/ChatInterface.tsx`
    - 整合 MessageList 和 MessageInput
    - 连接 WebSocket 和状态管理
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5_

- [x] 11. 实现 ConfirmationCard 组件
  - [x] 11.1 创建 ConfirmationCard 组件
    - 实现 `components/ChatInterface/ConfirmationCard.tsx`
    - 显示操作描述、确认和取消按钮
    - 实现超时机制（30秒）
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

  - [x]* 11.2 编写 ConfirmationCard 的单元测试
    - 测试超时场景
    - 测试确认/取消交互
    - _Requirements: 5.3, 5.4, 5.5_

  - [x]* 11.3 编写 ConfirmationCard 的属性测试
    - **Property 18: 确认卡片渲染完整性**
    - **Property 19: 确认操作响应正确性**
    - **Validates: Requirements 5.1, 5.2, 5.3, 5.4**

- [x] 12. Checkpoint - 确保对话界面功能完整
  - 确保所有测试通过，如有问题请询问用户

- [x] 13. 实现 ChartRenderer 组件
  - [x] 13.1 创建 ChartRenderer 组件
    - 实现 `components/ChatInterface/ChartRenderer.tsx`
    - 集成 Recharts 支持折线图、柱状图、饼图
    - 实现数据点悬停提示
    - 处理图表加载失败
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

  - [x]* 13.2 编写 ChartRenderer 的单元测试
    - 测试图表加载失败场景
    - 测试数据采样逻辑
    - _Requirements: 6.5, 10.5_

  - [x]* 13.3 编写 ChartRenderer 的属性测试
    - **Property 20: 图表组件渲染**
    - **Property 21: 图表交互提示**
    - **Validates: Requirements 6.1, 6.2, 6.3, 6.4**

- [x] 14. 实现 LogViewer 组件
  - [x] 14.1 创建 LogEntry 组件
    - 实现 `components/LogViewer/LogEntry.tsx`
    - 渲染日志条目和 Agent 引用
    - 将 Agent 引用渲染为可点击链接
    - _Requirements: 3.1, 3.2, 3.3_

  - [x]* 14.2 编写 LogEntry 的属性测试
    - **Property 11: Agent 引用可点击性**
    - **Property 12: 引用点击触发回调**
    - **Validates: Requirements 3.2, 3.3**

  - [x] 14.3 创建 LogList 组件
    - 实现 `components/LogViewer/LogList.tsx`
    - 实现虚拟滚动（日志超过 1000 条时）
    - 实现自动滚动和手动滚动控制
    - _Requirements: 3.1, 3.4, 3.5, 10.4_

  - [x]* 14.4 编写 LogList 的单元测试
    - 测试虚拟滚动触发条件
    - 测试自动滚动边界情况
    - _Requirements: 3.4, 10.4_

  - [x]* 14.5 编写 LogList 的属性测试
    - **Property 10: 日志实时追加**
    - **Property 13: 滚动行为控制**
    - **Validates: Requirements 3.1, 3.5**

  - [x] 14.6 组装 LogViewer 主组件
    - 实现 `components/LogViewer/LogViewer.tsx`
    - 整合 LogList 和 LogEntry
    - 连接状态管理
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 15. 实现 ContextModal 组件
  - [x] 15.1 创建 ContextModal 组件
    - 实现 `components/ContextModal/ContextModal.tsx`
    - 显示上下文日志列表
    - 实现模态框打开/关闭逻辑
    - 实现背景滚动锁定
    - 处理空日志状态
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

  - [x]* 15.2 编写 ContextModal 的单元测试
    - 测试空日志状态
    - 测试关闭交互
    - _Requirements: 4.3, 4.4_

  - [x]* 15.3 编写 ContextModal 的属性测试
    - **Property 14: 上下文模态框显示正确性**
    - **Property 15: 上下文日志排序**
    - **Property 16: 模态框关闭清理**
    - **Property 17: 模态框滚动锁定**
    - **Validates: Requirements 4.1, 4.2, 4.3, 4.5**

- [x] 16. 实现 SessionManager 组件
  - [x] 16.1 创建 SessionManager 组件
    - 实现 `components/SessionManager/SessionManager.tsx`
    - 显示"新建会话"和"历史会话"按钮
    - 实现历史会话列表（最近 10 条）
    - 实现会话切换功能
    - _Requirements: 8.3, 8.4, 8.5_

  - [x]* 16.2 编写 SessionManager 的单元测试
    - 测试历史会话列表限制（10 条）
    - 测试会话切换
    - _Requirements: 8.4, 8.5_

  - [x]* 16.3 编写 SessionManager 的属性测试
    - **Property 25: 历史会话加载一致性**
    - **Validates: Requirements 8.5**

- [x] 17. Checkpoint - 确保所有组件功能完整
  - 确保所有测试通过，如有问题请询问用户

- [x] 18. 实现错误处理和 Error Boundary
  - [x] 18.1 创建 ErrorBoundary 组件
    - 实现 `components/ErrorBoundary/ErrorBoundary.tsx`
    - 捕获组件渲染错误
    - 显示友好的错误页面
    - _Requirements: 9.4_

  - [x]* 18.2 编写 ErrorBoundary 的属性测试
    - **Property 26: 错误提示友好性**
    - **Validates: Requirements 9.4**

  - [x] 18.3 实现全局错误处理
    - 在 WebSocket 管理器中添加错误处理
    - 在状态管理中添加错误状态
    - 显示连接错误、发送失败等提示
    - _Requirements: 9.1, 9.2, 9.3_

- [x] 19. 实现性能优化
  - 在 MessageList 中实现虚拟滚动（react-window）
  - 在 ChartRenderer 中实现数据采样（超过 500 点）
  - 使用 React.memo 优化组件重渲染
  - 使用 useMemo 和 useCallback 优化计算和回调
  - _Requirements: 10.4, 10.5_

- [x] 20. 组装主应用
  - [x] 20.1 更新 App.tsx
    - 整合所有主要组件（Layout, ChatInterface, LogViewer, ContextModal）
    - 初始化 WebSocket 连接
    - 初始化状态管理
    - 加载历史会话
    - _Requirements: 所有需求_

  - [x] 20.2 添加样式和主题
    - 使用 Tailwind CSS 实现响应式样式
    - 实现暗色/亮色主题切换（可选）
    - 优化移动端体验
    - _Requirements: 7.1, 7.2, 7.3, 7.5_

- [x]* 21. 端到端测试和集成测试
  - [x]* 21.1 编写端到端测试
    - 测试完整的用户流程（发送消息、查看日志、确认操作）
    - 测试 WebSocket 连接和重连
    - 测试会话保存和恢复
    - _Requirements: 1.1, 3.1, 5.1, 8.1_

  - [x]* 21.2 编写集成测试
    - 测试组件间交互
    - 测试状态管理和 WebSocket 集成
    - _Requirements: 所有需求_

- [x] 22. 最终 Checkpoint - 确保所有功能完整且测试通过
  - 运行所有测试（单元测试 + 属性测试 + 集成测试）
  - 检查测试覆盖率（目标：语句 > 80%，分支 > 75%，函数 > 85%）
  - 手动测试所有功能
  - 如有问题请询问用户

## Notes

- 任务标记 `*` 的为可选任务，可以跳过以加快 MVP 开发
- 每个任务都引用了具体的需求编号，便于追溯
- Checkpoint 任务确保增量验证
- 属性测试验证通用正确性属性
- 单元测试验证具体示例和边界情况
- 基础架构（类型定义、状态管理、WebSocket、本地存储）已完成并通过测试
- 下一步需要安装 UI 相关依赖并开始实现组件
