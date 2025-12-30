# Implementation Plan: Chat UI Optimization

## Overview

本实施计划将 AI-Ops 智能运维平台的聊天界面从单栏宽屏布局改造为专业的三栏运维控制台风格。实施分为 5 个阶段，按优先级从 P0 到 P2 逐步推进。

## 技术栈

- **框架**: Vue 3 + TypeScript
- **状态管理**: Pinia
- **组件库**: Element Plus
- **样式**: Tailwind CSS + CSS Variables
- **测试**: Vitest + fast-check
- **持久化**: IndexedDB (Dexie.js)
- **Diff 库**: diff2html

## 依赖关系

```
Task 1 (基础设施) ──┬──> Task 2 (三栏布局)
                   ├──> Task 3 (消息宽度)
                   ├──> Task 4 (AI思考状态)
                   ├──> Task 5 (审计日志)
                   └──> Task 6 (确认对话框)

Task 5 (审计日志) ──> Task 6 (确认对话框) ──> Task 20 (AI终端)

Task 2 (三栏布局) ──┬──> Task 10 (监控面板)
                   ├──> Task 11 (历史面板)
                   └──> Task 18 (资源监控)

Task 1.2 (UI Store) ──> Task 14 (深色模式)
                    ──> Task 15 (快捷键)
                    ──> Task 16 (WebSocket)
```

## 预估工时汇总

| 阶段 | 任务数 | 预估工时 | 累计 |
|------|--------|----------|------|
| Phase 1 | 7 | 5-6 天 | 5-6 天 |
| Phase 2 | 5 | 5-6 天 | 10-12 天 |
| Phase 3 | 5 | 4-5 天 | 14-17 天 |
| Phase 4 | 5 | 5-6 天 | 19-23 天 |
| Phase 5 | 4 | 3-4 天 | 22-27 天 |

## Tasks

### Phase 1: P0 - 核心布局与安全基础 (预估: 5-6 天)

- [x] 1. 设置项目基础设施 (预估: 1 天)
  - [x] 1.1 创建类型定义文件 `web/src/types/chat-ui.ts`
    - 定义所有组件接口: ChatWindowState, Message, ToolCall, HostMetrics, RiskLevel 等
    - 定义 Store 状态接口: ChatState, MetricsState, AuditLogState, UIState
    - _Requirements: 全部需求的类型基础_
    - _依赖: 无_

  - [x] 1.2 创建 UI Store `web/src/stores/ui.ts`
    - 实现 UIState 接口: theme, connectionStatus, modals, notifications
    - 实现 actions: setTheme, togglePanel, showNotification
    - _Requirements: 13.5, 14.4, 15.2_
    - _依赖: 1.1_

  - [x] 1.3 安装必要依赖
    - 安装 dexie (IndexedDB), diff2html, fast-check
    - 配置 Tailwind CSS dark mode
    - _Requirements: 12.1, 7.1, 13.1_
    - _依赖: 无_

- [x] 2. 实现三栏布局 (Req 1) (预估: 1 天)
  - [x] 2.1 重构 `ChatWindow.vue` 为三栏布局
    - 使用 CSS Grid 实现响应式三栏布局
    - 实现断点逻辑: >1200px 三栏, 768-1200px 两栏, <768px 单栏
    - 添加面板显示/隐藏状态管理
    - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 1.6_
    - _依赖: 1.1, 1.2_

  - [ ]* 2.2 编写响应式布局属性测试
    - **Property 1: Responsive Layout Consistency**
    - **Validates: Requirements 1.1, 1.2, 1.3**

- [x] 3. 实现消息气泡宽度优化 (Req 6) (预估: 0.5 天)
  - [x] 3.1 更新 `MessageItem.vue` 样式
    - 设置最大宽度 800px (1920px+ 屏幕为 1000px)
    - 居中对齐消息卡片
    - 代码块最大高度 300px + 滚动
    - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_
    - _依赖: 1.1_

  - [ ]* 3.2 编写输出高度属性测试
    - **Property 22: Output Height Consistency**
    - **Validates: Requirements 2.2, 6.3**

- [x] 4. 实现 AI 思考状态细化 (Req 5) (预估: 0.5 天)
  - [x] 4.1 更新 `ThinkingProcess.vue` 组件
    - 显示具体步骤描述: "正在检索解决方案...", "正在读取 {filename}..."
    - 显示每步耗时 (e.g., "2.3s")
    - 文件引用可点击，支持行号定位
    - 失败时显示错误信息和重试选项
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6_
    - _依赖: 1.1_

- [x] 5. 实现审计日志系统 (Req 10) - 安全优先 (预估: 1.5 天)
  - [x] 5.1 创建审计日志 API `web/src/api/audit.ts`
    - 实现 logCommand, queryLogs, exportLogs 接口
    - 定义危险命令模式检测函数
    - _Requirements: 10.1, 10.2, 10.3, 10.4_
    - _依赖: 1.1_

  - [x] 5.2 创建审计日志 Store `web/src/stores/audit.ts`
    - 实现 AuditLogState 接口
    - 实现 actions: addLog, queryLogs, exportLogs
    - _Requirements: 10.6, 10.7_
    - _依赖: 5.1_

  - [x] 5.3 创建 `AuditLogViewer.vue` 组件
    - 实现日志列表展示、筛选、导出功能
    - 支持按时间、用户、主机、风险级别筛选
    - _Requirements: 10.6, 10.7_
    - _依赖: 5.2_

  - [ ] 5.4 编写审计日志完整性属性测试

    - **Property 13: Audit Log Completeness**
    - **Validates: Requirements 10.1, 10.2**

- [ ] 6. 实现二次确认对话框 (Req 11) - 安全优先 (预估: 1 天)
  - [x] 6.1 创建 `ConfirmationDialog.vue` 组件
    - 实现风险级别样式 (low/medium/high/critical)
    - 高风险需输入 "CONFIRM" 确认
    - 实现倒计时功能 (5秒)
    - 默认焦点在取消按钮
    - _Requirements: 11.1, 11.2, 11.3, 11.4, 11.5, 11.6_
    - _依赖: 1.1, 5.1_

  - [ ] 6.2 编写高风险确认属性测试

    - **Property 14: High-Risk Command Confirmation**
    - **Validates: Requirements 11.1, 11.3, 11.4, 11.6**

- [ ] 7. Checkpoint - Phase 1 验证
  - 确保三栏布局在所有断点正常工作
  - 确保 AI 思考状态显示具体步骤
  - 确保审计日志正确记录所有命令
  - 确保高风险命令触发确认对话框
  - 运行所有属性测试，确保通过

### Phase 2: P1 - 交互优化 (预估: 5-6 天)

- [x] 8. 实现对话卡片运维化改造 (Req 2) (预估: 1.5 天)
  - [x] 8.1 更新 `ToolCallCard.vue` 折叠功能
    - 超过 5 行自动折叠，显示"展开全量日志"按钮
    - 最大高度 300px + 滚动
    - 错误状态默认展开
    - _Requirements: 2.1, 2.2, 2.3_
    - _依赖: 1.1_

  - [x] 8.2 创建 `AggregatedProgressBar.vue` 组件
    - 批量操作显示聚合进度条
    - 点击展开显示每个主机详情
    - _Requirements: 2.4, 2.5_
    - _依赖: 1.1_

  - [x] 8.3 更新 `MessageActions.vue` 添加操作按钮
    - 添加"复制命令"、"推荐执行"、"保存为脚本"按钮
    - "推荐执行"触发风险评估和确认
    - _Requirements: 2.6, 2.7_
    - _依赖: 6.1 (确认对话框)_

  - [ ] 8.4 编写命令输出折叠属性测试

    - **Property 2: Command Output Collapse Behavior**
    - **Validates: Requirements 2.1**
    - _依赖: 8.1_


  - [ ]* 8.5 编写批量操作聚合属性测试
    - **Property 3: Batch Operation Aggregation**
    - **Validates: Requirements 2.3**
    - _依赖: 8.2_

- [x] 9. 实现输入框增强 (Req 3) (预估: 1.5 天)
  - [x] 9.1 更新 `InputBox.vue` 添加 Slash Commands
    - 输入 `/` 显示命令下拉菜单
    - 支持 /top, /tail, /check, /disk, /memory, /process
    - 键盘导航 (Arrow + Enter)
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.8, 3.9_
    - _依赖: 1.1_

  - [x] 9.2 添加 Host Mention 功能
    - 输入 `@` 显示主机下拉菜单
    - 选中主机显示为可移除的 chips
    - _Requirements: 3.5, 3.6, 3.7_
    - _依赖: 1.1_

  - [ ]* 9.3 编写 Slash Command 触发属性测试
    - **Property 4: Slash Command Trigger**
    - **Validates: Requirements 3.1, 3.3**
    - _依赖: 9.1_

  - [ ]* 9.4 编写 Host Mention 触发属性测试
    - **Property 5: Host Mention Trigger**
    - **Validates: Requirements 3.4, 3.5**
    - _依赖: 9.2_

- [x] 10. 实现监控面板静默模式 (Req 8) (预估: 1 天)
  - [x] 10.1 创建 `ContextPanel.vue` 组件
    - 显示 CPU、内存使用率
    - 实现静默模式切换
    - 异常时自动展开 + 脉冲动画
    - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5, 8.6, 8.7_
    - _依赖: 2.1 (三栏布局), 1.2 (UI Store)_

  - [ ]* 10.2 编写静默模式行为属性测试
    - **Property 9: Silent Mode Behavior**
    - **Validates: Requirements 8.2, 8.3, 8.5**
    - _依赖: 10.1_

- [x] 11. 实现小屏幕历史面板 (Req 17) (预估: 0.5 天)
  - [x] 11.1 创建 `HistoryPanelOverlay.vue` 组件
    - 实现侧滑抽屉效果
    - 支持点击遮罩关闭
    - 支持触摸滑动关闭
    - _Requirements: 17.1, 17.2, 17.3, 17.4, 17.5_
    - _依赖: 2.1 (三栏布局)_

  - [ ]* 11.2 编写移动端历史面板属性测试
    - **Property 20: Mobile History Panel Behavior**
    - **Validates: Requirements 17.1, 17.2, 17.5**
    - _依赖: 11.1_

- [ ] 12. Checkpoint - Phase 2 验证 (预估: 0.5 天)
  - 确保所有交互功能正常工作
  - 确保 Slash Commands 和 Host Mention 正确触发
  - 运行所有属性测试，确保通过

### Phase 3: P1 - 用户体验增强 (预估: 4-5 天)

- [x] 13. 实现会话持久化 (Req 12) (预估: 1 天)
  - [x] 13.1 创建 IndexedDB 数据库 `web/src/utils/chatDatabase.ts`
    - 使用 Dexie.js 创建 ChatDatabase
    - 定义 messages, sessions, drafts 表
    - _Requirements: 12.1_
    - _依赖: 1.3 (dexie 依赖安装)_

  - [x] 13.2 实现自动保存逻辑
    - 每 10 秒自动保存消息
    - 保存输入草稿
    - 显示"最后保存"时间戳
    - _Requirements: 12.1, 12.2, 12.6_
    - _依赖: 13.1_

  - [x] 13.3 实现会话恢复和导出
    - 页面刷新后恢复会话状态
    - 支持 Markdown 格式导出
    - 支持导入之前导出的对话
    - _Requirements: 12.3, 12.4, 12.5_
    - _依赖: 13.2_

  - [ ]* 13.4 编写会话持久化属性测试
    - **Property 15: Session Persistence Accuracy**
    - **Validates: Requirements 12.1, 12.2, 12.3**
    - _依赖: 13.3_

- [x] 14. 实现深色模式 (Req 13) (预估: 1 天)
  - [x] 14.1 创建主题系统 `web/src/composables/useTheme.ts`
    - 支持 light/dark/auto 三种模式
    - 监听系统主题变化
    - 持久化到 localStorage
    - _Requirements: 13.1, 13.5, 13.6_
    - _依赖: 1.2 (UI Store)_

  - [x] 14.2 更新 CSS 变量 `web/src/styles/theme.css`
    - 定义深色模式颜色变量 (#1a1a1a 背景)
    - 确保 WCAG AA 对比度
    - 代码块深色主题
    - _Requirements: 13.2, 13.3, 13.4_
    - _依赖: 1.3 (Tailwind dark mode 配置)_

  - [x] 14.3 创建 `ThemeToggle.vue` 组件
    - 主题切换按钮
    - 无需刷新页面
    - _Requirements: 13.6_
    - _依赖: 14.1, 14.2_

  - [ ]* 14.4 编写主题一致性属性测试
    - **Property 16: Theme Consistency**
    - **Validates: Requirements 13.1, 13.2, 13.5, 13.6**
    - _依赖: 14.3_

- [x] 15. 实现快捷键系统 (Req 14) (预估: 1 天)
  - [x] 15.1 创建快捷键管理器 `web/src/composables/useShortcuts.ts`
    - 注册/注销快捷键
    - 处理浏览器默认行为冲突
    - _Requirements: 14.1, 14.3_
    - _依赖: 1.2 (UI Store)_

  - [x] 15.2 创建 `ShortcutHelpModal.vue` 组件
    - Ctrl+/ 显示快捷键帮助
    - 分类展示所有快捷键
    - _Requirements: 14.2_
    - _依赖: 15.1_

  - [x] 15.3 集成快捷键到各组件
    - Ctrl+Enter 发送消息
    - Ctrl+K 清空输入框
    - Ctrl+H/M 切换面板
    - 执行后显示 toast 通知
    - _Requirements: 14.1, 14.4_
    - _依赖: 15.1, 9.1 (InputBox)_

  - [ ]* 15.4 编写快捷键执行属性测试
    - **Property 17: Keyboard Shortcut Execution**
    - **Validates: Requirements 14.1, 14.3, 14.4_
    - _依赖: 15.3_

- [x] 16. 实现 WebSocket 连接状态 (Req 15) (预估: 1 天)
  - [x] 16.1 创建 WebSocket 管理器 `web/src/composables/useWebSocket.ts`
    - 连接状态管理
    - 指数退避重连 (1s, 2s, 4s, 8s, max 30s)
    - 心跳检测
    - _Requirements: 15.5_
    - _依赖: 1.2 (UI Store)_

  - [x] 16.2 创建 `ConnectionStatusIndicator.vue` 组件
    - 显示连接状态 (绿/黄/红)
    - 数据传输时脉冲动画
    - 点击显示详细信息
    - _Requirements: 15.1, 15.2, 15.3, 15.7_
    - _依赖: 16.1_

  - [x] 16.3 实现连接状态通知
    - 断开时显示 toast
    - 重连成功显示通知
    - _Requirements: 15.4, 15.6_
    - _依赖: 16.1, 1.2 (UI Store notifications)_

  - [ ]* 16.4 编写 WebSocket 重连属性测试
    - **Property 18: WebSocket Reconnection Behavior**
    - **Validates: Requirements 15.2, 15.4, 15.5, 15.6**
    - _依赖: 16.3_

- [ ] 17. Checkpoint - Phase 3 验证 (预估: 0.5 天)
  - 确保会话持久化正常工作
  - 确保深色模式正确应用
  - 确保快捷键正常触发
  - 确保 WebSocket 重连正常
  - 运行所有属性测试，确保通过

### Phase 4: P2 - 高级功能 (预估: 5-6 天)

- [x] 18. 实现实时资源监控面板 (Req 4) (预估: 1.5 天)
  - [x] 18.1 创建 Metrics Store `web/src/stores/metrics.ts`
    - 实现 MetricsState 接口
    - 60 秒自动刷新
    - 连续 3 次失败后停止刷新
    - _Requirements: 4.6, 4.8_
    - _依赖: 1.1 (类型定义)_

  - [x] 18.2 更新 `ContextPanel.vue` 添加监控功能
    - 显示 CPU、内存、磁盘使用率
    - 阈值告警高亮 (CPU>80%, Memory>85%)
    - 最多显示 5 个主机
    - _Requirements: 4.1, 4.2, 4.3, 4.7_
    - _依赖: 10.1 (ContextPanel 基础), 18.1_

  - [x] 18.3 实现相关文件展示
    - AI 提及文件时显示在面板
    - 点击打开文件预览
    - 最多显示 3 个文件
    - _Requirements: 4.4, 4.5_
    - _依赖: 18.2_

  - [ ]* 18.4 编写指标显示完整性属性测试
    - **Property 6: Metrics Display Completeness**
    - **Validates: Requirements 4.1, 4.2**
    - _依赖: 18.2_

  - [ ]* 18.5 编写阈值告警属性测试
    - **Property 7: Threshold Alert Highlighting**
    - **Validates: Requirements 4.3**
    - _依赖: 18.2_

- [x] 19. 实现配置文件 Diff 视图 (Req 7) (预估: 1.5 天)
  - [x] 19.1 创建 `DiffViewer.vue` 组件
    - 使用 diff2html 渲染 Diff
    - 红色删除行、绿色添加行
    - 显示行号
    - _Requirements: 7.1, 7.2, 7.3_
    - _依赖: 1.3 (diff2html 依赖安装)_

  - [x] 19.2 实现 Diff 交互功能
    - 折叠未修改部分
    - 大文件警告 (>1000 行)
    - 应用/拒绝按钮
    - _Requirements: 7.4, 7.5, 7.6_
    - _依赖: 19.1, 6.1 (确认对话框)_

  - [ ]* 19.3 编写 Diff 视图正确性属性测试
    - **Property 11: Diff View Correctness**
    - **Validates: Requirements 7.1, 7.2, 7.3**
    - _依赖: 19.2_

- [x] 20. 实现 AI 辅助快捷终端 (Req 9) (预估: 1.5 天)
  - [x] 20.1 创建 `AIAssistedTerminal.vue` 组件
    - 可折叠终端栏
    - 命令历史导航 (Up/Down)
    - 30 秒无操作自动收起
    - _Requirements: 9.1, 9.2, 9.6, 9.7_
    - _依赖: 2.1 (三栏布局)_

  - [x] 20.2 实现风险评估功能
    - 输入命令显示 AI 风险评估
    - 中/高风险需确认
    - 记录到审计日志
    - _Requirements: 9.3, 9.4, 9.5, 9.8_
    - _依赖: 20.1, 5.2 (审计日志 Store), 6.1 (确认对话框)_

  - [ ]* 20.3 编写 AI 终端风险评估属性测试
    - **Property 10: AI Assisted Terminal Execution**
    - **Property 21: AI Terminal Risk Assessment**
    - **Validates: Requirements 9.3, 9.4, 9.8_
    - _依赖: 20.2_

- [x] 21. 实现消息搜索与过滤 (Req 16) (预估: 1 天)
  - [x] 21.1 创建 `MessageSearchBar.vue` 组件
    - Ctrl+F 打开搜索
    - 搜索消息内容、命令输出、文件名
    - 高亮匹配文本
    - _Requirements: 16.1, 16.2, 16.3_
    - _依赖: 15.1 (快捷键管理器)_

  - [x] 21.2 实现高级搜索功能
    - 时间范围、工具类型、主机、错误级别筛选
    - 结果计数和导航
    - 点击结果滚动到消息
    - _Requirements: 16.4, 16.5, 16.6, 16.7_
    - _依赖: 21.1_

  - [ ]* 21.3 编写搜索结果准确性属性测试
    - **Property 19: Search Result Accuracy**
    - **Validates: Requirements 16.2, 16.3**
    - _依赖: 21.2_

- [ ] 22. Checkpoint - Phase 4 验证 (预估: 0.5 天)
  - 确保监控面板正常刷新
  - 确保 Diff 视图正确渲染
  - 确保 AI 终端风险评估正常
  - 确保搜索功能正常工作
  - 运行所有属性测试，确保通过

### Phase 5: 收尾与优化 (预估: 3-4 天)

- [ ] 23. 创建错误边界组件 (预估: 0.5 天)
  - [ ] 23.1 创建 `ErrorBoundary.vue` 组件
    - 捕获组件错误
    - 显示友好错误信息
    - 提供重试按钮
    - _Requirements: 通用错误处理_
    - _依赖: 1.1 (类型定义)_

- [ ] 24. 性能优化 (预估: 1 天)
  - [ ] 24.1 实现消息列表虚拟滚动
    - 使用 vue-virtual-scroller
    - 支持 1000+ 消息流畅滚动
    - _Requirements: 性能优化_
    - _依赖: 2.1 (三栏布局)_

  - [ ] 24.2 实现组件懒加载
    - 按需加载 DiffViewer, AuditLogViewer
    - 代码分割优化首屏加载
    - _Requirements: 性能优化_
    - _依赖: 19.1 (DiffViewer), 5.3 (AuditLogViewer)_

- [ ] 25. 可访问性优化 (预估: 1 天)
  - [ ] 25.1 添加 ARIA 标签
    - 为所有交互元素添加 ARIA 属性
    - 确保屏幕阅读器兼容
    - _Requirements: 可访问性_
    - _依赖: Phase 1-4 所有组件_

  - [ ] 25.2 焦点管理
    - 模态框打开/关闭时正确管理焦点
    - 键盘导航支持
    - _Requirements: 可访问性_
    - _依赖: 6.1 (确认对话框), 15.1 (快捷键管理器)_

- [ ] 26. Final Checkpoint - 全面验证 (预估: 0.5-1 天)
  - 运行所有单元测试
  - 运行所有属性测试 (22 条)
  - 执行 E2E 测试
  - 视觉回归测试
  - 性能基准测试
  - 确保所有 17 个需求完全覆盖

## Notes

- Tasks marked with `*` are optional property-based tests that can be skipped for faster MVP
- Each task references specific requirements for traceability
- Each sub-task includes `_依赖: X.X_` showing prerequisite tasks
- Checkpoints ensure incremental validation
- Property tests validate universal correctness properties
- Unit tests validate specific examples and edge cases
- Req 5 (AI 思考状态) 已移至 Phase 1 Task 4，作为基础功能优先实现
- 预计总工时: 22-27 天 (26 个主任务，含 5 个 Checkpoint)

## 文件创建清单

### 新建文件
- `web/src/types/chat-ui.ts` - 类型定义
- `web/src/stores/ui.ts` - UI 状态管理
- `web/src/stores/audit.ts` - 审计日志状态
- `web/src/stores/metrics.ts` - 监控指标状态
- `web/src/api/audit.ts` - 审计日志 API
- `web/src/utils/chatDatabase.ts` - IndexedDB 数据库
- `web/src/utils/riskAssessment.ts` - 风险评估工具
- `web/src/composables/useTheme.ts` - 主题管理
- `web/src/composables/useShortcuts.ts` - 快捷键管理
- `web/src/composables/useWebSocket.ts` - WebSocket 管理
- `web/src/styles/theme.css` - 主题样式
- `web/src/components/chat/ContextPanel.vue` - 上下文面板
- `web/src/components/chat/AggregatedProgressBar.vue` - 聚合进度条
- `web/src/components/chat/DiffViewer.vue` - Diff 视图
- `web/src/components/chat/AIAssistedTerminal.vue` - AI 辅助终端
- `web/src/components/chat/HistoryPanelOverlay.vue` - 历史面板遮罩
- `web/src/components/chat/MessageSearchBar.vue` - 消息搜索栏
- `web/src/components/common/ConfirmationDialog.vue` - 确认对话框
- `web/src/components/common/AuditLogViewer.vue` - 审计日志查看器
- `web/src/components/common/ConnectionStatusIndicator.vue` - 连接状态
- `web/src/components/common/ShortcutHelpModal.vue` - 快捷键帮助
- `web/src/components/common/ThemeToggle.vue` - 主题切换
- `web/src/components/common/ErrorBoundary.vue` - 错误边界

### 修改文件
- `web/src/components/chat/ChatWindow.vue` - 三栏布局
- `web/src/components/chat/MessageItem.vue` - 宽度优化
- `web/src/components/chat/ToolCallCard.vue` - 折叠功能
- `web/src/components/chat/InputBox.vue` - Slash Commands + Host Mention
- `web/src/components/chat/MessageActions.vue` - 操作按钮
- `web/src/components/chat/ThinkingProcess.vue` - 思考状态细化
- `web/src/stores/chat.ts` - 扩展状态
- `web/src/App.vue` - 主题提供者
