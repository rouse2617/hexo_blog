# Requirements Document

## Introduction

本文档定义了 AI-Ops 智能运维平台的聊天界面优化需求。目标是将当前的"宽屏散沙"布局改造为专业的运维控制台风格，提升用户体验和操作效率。

### 版本历史

| 版本 | 日期 | 变更说明 |
|------|------|----------|
| v1.0 | 初始版本 | 原始 9 个需求 |
| v2.0 | 2024-01-XX | 基于 agent 分析优化：新增 8 个需求、修复冲突、完善安全性 |

### 优化概要

- **新增需求 8 个**: 审计日志、二次确认、会话持久化、深色模式、快捷键、连接状态、消息搜索、小屏访问
- **安全增强**: Req 9 改为 AI 辅助终端，所有命令需经过风险评估
- **冲突修复**: 统一代码块/卡片高度为 300px，异常状态暂停监控刷新
- **优先级划分**: P0 (MVP + 安全) → P1 (交互优化) → P2 (高级功能)

## Glossary

- **Chat_Window**: 聊天窗口主组件，包含消息列表、输入框等
- **Message_Card**: 消息卡片组件，展示用户和 AI 的对话内容
- **Tool_Call_Card**: 工具调用卡片，展示命令执行过程和结果
- **Context_Panel**: 上下文面板，显示实时资源监控和相关文件
- **Input_Box**: 输入框组件，用户输入消息的区域
- **Slash_Command**: 快捷指令，以 `/` 开头的预定义命令
- **Host_Mention**: 主机提及，以 `@` 开头的主机选择语法
- **Quick_Terminal**: 快捷终端，AI 辅助的命令执行面板
- **Audit_Log**: 审计日志，记录所有命令执行的日志系统
- **Confirmation_Dialog**: 确认对话框，敏感操作前的二次确认弹窗
- **History_Panel_Overlay**: 历史面板遮罩，小屏幕下的历史记录侧滑面板
- **Status_Indicator**: 状态指示器，显示 WebSocket 连接状态
- **Diff_View**: 差异视图，展示配置文件变更对比的视图

## Requirements

### Requirement 1: 三栏布局改造

**User Story:** As a 运维工程师, I want 专业的三栏布局界面, so that 我可以同时查看对话、历史记录和实时监控信息。

#### Acceptance Criteria

1. THE Chat_Window SHALL display a three-column layout on screens wider than 1200px
2. WHEN the screen width is between 768px and 1200px, THE Chat_Window SHALL collapse to a two-column layout (hiding the left panel)
3. WHEN the screen width is less than 768px, THE Chat_Window SHALL display a single-column layout
4. THE left panel SHALL display conversation history with a maximum width of 280px
5. THE center panel SHALL display the chat flow with a maximum width of 800px and centered alignment
6. THE right panel SHALL display real-time resource monitoring with a minimum width of 300px
7. WHEN a user clicks on a history item, THE Chat_Window SHALL switch to that conversation

### Requirement 2: 对话卡片运维化改造

**User Story:** As a 运维工程师, I want 更紧凑的命令输出展示, so that 批量操作时界面不会混乱。

#### Acceptance Criteria

1. WHEN a command output exceeds 5 lines, THE Tool_Call_Card SHALL collapse the output and show an "展开全量日志" button
2. THE Tool_Call_Card SHALL limit the maximum height to 300px with overflow scrolling (FIXED: was 200px)
3. WHEN a command execution fails, THE Tool_Call_Card SHALL be expanded by default to show error details
4. WHEN AI executes commands on multiple hosts, THE Tool_Call_Card SHALL display an aggregated progress bar instead of multiple separate cards
5. WHEN a user clicks the progress bar, THE Tool_Call_Card SHALL expand to show detailed results for each host
6. THE Message_Card SHALL display action buttons below AI responses: "复制命令", "推荐执行", "保存为脚本"
7. WHEN a user clicks "推荐执行", THE System SHALL show risk assessment and require confirmation

### Requirement 3: 输入框增强

**User Story:** As a 运维工程师, I want 快捷指令和节点选择功能, so that 我可以更快速地执行常用操作。

#### Acceptance Criteria

1. WHEN a user types `/` at the beginning of input or after a space, THE Input_Box SHALL display a dropdown menu of available Slash_Commands
2. WHEN `/` is typed as part of a file path (e.g., after a word), THE Input_Box SHALL NOT trigger the Slash_Command menu
3. THE System SHALL support the following Slash_Commands: `/top`, `/tail`, `/check`, `/disk`, `/memory`, `/process`
4. WHEN a user selects a Slash_Command, THE Input_Box SHALL auto-complete the command template with parameter placeholders
5. WHEN a user types `@`, THE Input_Box SHALL display a dropdown menu of available hosts
6. WHEN a user selects a Host_Mention, THE Input_Box SHALL add the host to the current context
7. THE Input_Box SHALL display currently selected hosts as removable chips above the input area
8. THE Input_Box SHALL support keyboard navigation (Arrow keys + Enter) for dropdown selection
9. THE System SHALL provide inline help for each Slash_Command showing parameters and examples

### Requirement 4: 实时资源监控面板

**User Story:** As a 运维工程师, I want 实时查看服务器状态, so that 我可以在对话中快速了解系统健康状况。

#### Acceptance Criteria

1. THE Context_Panel SHALL display CPU usage percentage for selected hosts (maximum 5 hosts visible at once)
2. THE Context_Panel SHALL display memory usage percentage for selected hosts
3. THE Context_Panel SHALL highlight metrics that exceed warning thresholds (CPU > 80%, Memory > 85%)
4. WHEN AI mentions a file in the conversation, THE Context_Panel SHALL display the file in the "相关文件" section (maximum 3 files)
5. WHEN a user clicks a file in the Context_Panel, THE System SHALL open a file preview modal with syntax highlighting
6. THE Context_Panel SHALL auto-refresh metrics every 60 seconds (OPTIMIZED: was 30s)
7. IF no hosts are selected, THEN THE Context_Panel SHALL display a prompt to select hosts
8. WHEN metrics refresh fails, THE System SHALL display error indicator and stop auto-refresh after 3 consecutive failures
9. WHEN any metric is in abnormal state, THE System SHALL pause auto-refresh to maintain user attention (FIXED conflict with Req 8)

### Requirement 5: AI 思考状态细化

**User Story:** As a 运维工程师, I want 详细的 AI 执行状态反馈, so that 我在等待时能了解 AI 正在做什么。

#### Acceptance Criteria

1. WHEN AI is processing, THE System SHALL display specific step descriptions instead of generic "正在分析..."
2. THE System SHALL use semantic step descriptions: "正在检索解决方案...", "正在读取 {filename}...", "正在主机 {hostname} 上执行 {command}..."
3. WHEN AI references a log file, THE Message_Card SHALL display a clickable link with line number if available
4. WHEN a user clicks the file link, THE System SHALL open file preview and highlight the relevant lines
5. THE thinking process SHALL display elapsed time for each step (e.g., "2.3s", "1.1s")
6. WHEN a step fails, THE System SHALL display error message and retry option

### Requirement 6: 消息气泡宽度优化

**User Story:** As a 运维工程师, I want 更聚焦的消息展示, so that 高分辨率屏幕下阅读更舒适。

#### Acceptance Criteria

1. THE Message_Card SHALL have a maximum width of 800px (1000px on screens wider than 1920px)
2. THE Message_Card SHALL be centered within the chat panel
3. THE code blocks within Message_Card SHALL have a maximum height of 300px with overflow scrolling (FIXED: was 200px, matches Req 2)
4. THE Message_Card SHALL maintain consistent padding and spacing regardless of screen size
5. THE Message_Card SHALL use responsive font sizing for better readability on different screens

### Requirement 7: 配置文件 Diff 视图

**User Story:** As a 运维工程师, I want 在执行配置修改前查看变更对比, so that 我可以确认修改内容是否正确。

#### Acceptance Criteria

1. WHEN AI suggests modifying a configuration file, THE Message_Card SHALL display a Diff view with red/green highlighting
2. THE Diff_View SHALL show deleted lines in red background and added lines in green background
3. THE Diff_View SHALL display line numbers for both old and new content
4. WHEN a user clicks "推荐执行" after viewing Diff, THE System SHALL show risk assessment and require confirmation
5. THE Diff_View SHALL support collapsing unchanged sections to focus on modifications
6. FOR files larger than 1000 lines, THE Diff_View SHALL display a warning and offer to show only changed regions

### Requirement 8: 监控面板静默模式

**User Story:** As a 运维工程师, I want 监控面板在正常时自动收起, so that 界面更简洁，异常时能立即引起注意。

#### Acceptance Criteria

1. THE Context_Panel SHALL support a "静默模式" toggle switch
2. WHEN 静默模式 is enabled AND all metrics are normal, THE Context_Panel SHALL collapse to show only host names with status dots
3. WHEN any metric exceeds threshold (CPU > 80% OR Memory > 85%), THE Context_Panel SHALL auto-expand and highlight the abnormal metric
4. THE Context_Panel SHALL display a pulsing animation on abnormal metrics to attract attention
5. WHEN 静默模式 is disabled, THE Context_Panel SHALL always show full metrics
6. WHEN an abnormal metric is detected in silent mode, THE System SHALL pause auto-refresh until user acknowledges (FIXED conflict with Req 4)
7. WHEN user clicks the abnormal metric, THE System SHALL show detailed breakdown and suggestions

### Requirement 9: AI 辅助快捷终端

**User Story:** As a 运维工程师, I want 快速执行简单命令并得到 AI 风险评估, so that 我可以更高效地完成日常操作同时保证安全性。

#### Acceptance Criteria

1. THE Chat_Window SHALL include a collapsible quick terminal bar below the main input area
2. WHEN a user clicks the terminal icon, THE Quick_Terminal SHALL expand to show a command input line
3. WHEN a user enters a command, THE Quick_Terminal SHALL display AI risk assessment and command explanation
4. THE Quick_Terminal SHALL require user confirmation before executing commands marked as medium or high risk
5. THE Quick_Terminal SHALL display command output inline with a compact format
6. THE Quick_Terminal SHALL support command history navigation with Up/Down arrow keys
7. THE Quick_Terminal SHALL auto-collapse 30 seconds after mouse leaves the terminal area
8. THE Quick_Terminal SHALL log all executed commands to the audit system

### Requirement 10: 命令执行审计日志

**User Story:** As a 安全管理员, I want 记录所有危险命令的执行, so that 可以进行安全审计和问题追溯。

#### Acceptance Criteria

1. THE System SHALL log all commands executed through any interface (Chat, Quick_Terminal, Slash_Command)
2. THE Audit_Log SHALL include: timestamp, user_id, command, target_hosts, execution_result, risk_level
3. THE Audit_Log SHALL assign risk levels: low (read operations), medium (write operations), high (destructive operations)
4. THE System SHALL detect dangerous command patterns: `rm -rf`, `dd if=`, `mkfs.`, `shutdown`, `reboot`, `:(){:|:&};:`
5. WHEN a high-risk command is detected, THE System SHALL require additional confirmation
6. THE Audit_Log SHALL support filtering by time range, user, host, and risk level
7. THE Audit_Log SHALL be exportable in CSV format

### Requirement 11: 敏感操作二次确认

**User Story:** As a 运维工程师, I want 在执行危险操作前得到明确警告, so that 避免误操作导致系统故障。

#### Acceptance Criteria

1. WHEN a command contains destructive patterns, THE System SHALL display a confirmation dialog with red warning styling
2. THE Confirmation_Dialog SHALL display the exact command to be executed
3. THE Confirmation_Dialog SHALL show a list of affected hosts
4. THE Confirmation_Dialog SHALL require typing "CONFIRM" to execute high-risk operations
5. THE System SHALL provide a "cancel" option that is the default and focused
6. THE Confirmation_Dialog SHALL display a countdown timer (5 seconds) before enabling the confirm button for critical operations

### Requirement 12: 会话持久化与恢复

**User Story:** As a 运维工程师, I want 刷新页面后恢复之前的对话, so that 不会因为网络问题丢失工作进度。

#### Acceptance Criteria

1. THE System SHALL automatically save conversation messages to IndexedDB every 10 seconds
2. THE System SHALL save input draft content to prevent data loss on page refresh
3. WHEN the page is reloaded, THE System SHALL restore the previous conversation state
4. THE System SHALL support manual export of conversations in Markdown format
5. THE System SHALL support importing previously exported conversations
6. THE System SHALL display "last saved" timestamp to indicate data persistence status

### Requirement 13: 深色模式支持

**User Story:** As a 运维工程师, I want 在夜间值班时使用深色界面, so that 减少眼睛疲劳。

#### Acceptance Criteria

1. THE System SHALL support three theme modes: light, dark, auto (follow system)
2. THE Dark_Mode SHALL use a dark gray (#1a1a1a) background rather than pure black
3. THE Dark_Mode SHALL maintain sufficient color contrast (WCAG AA compliant)
4. THE Code_Blocks SHALL use a syntax-highlighting theme optimized for dark mode
5. THE Theme preference SHALL be persisted in localStorage
6. THE Theme SHALL be switchable without page reload

### Requirement 14: 快捷键系统

**User Story:** As a 运维工程师, I want 使用键盘快捷键提高操作效率, so that 不需要频繁切换到鼠标。

#### Acceptance Criteria

1. THE System SHALL support the following keyboard shortcuts:
   - `Ctrl+Enter` or `Cmd+Enter`: Send message
   - `Ctrl+K` or `Cmd+K`: Clear input box
   - `Ctrl+H` or `Cmd+H`: Toggle history panel
   - `Ctrl+M` or `Cmd+M`: Toggle monitoring panel
   - `Ctrl+Shift+C`: Copy last command output
   - `Escape`: Close modals/dropdowns
   - `Up/Down`: Navigate input history
   - `Tab`: Accept autocomplete suggestion
2. THE System SHALL display a help modal showing all available shortcuts when `Ctrl+/` is pressed
3. THE Keyboard shortcuts SHALL be customizable in user settings
4. THE System SHALL show a toast notification when an action is performed via shortcut

### Requirement 15: WebSocket 连接状态指示器

**User Story:** As a 运维工程师, I want 实时了解与服务器的连接状态, so that 在网络问题时及时知晓。

#### Acceptance Criteria

1. THE System SHALL display a connection status indicator in the top-right corner
2. THE Status_Indicator SHALL show three states: connected (green), connecting (yellow), disconnected (red)
3. THE Status_Indicator SHALL display a dot icon that pulses when data is being transmitted
4. WHEN connection is lost, THE System SHALL display an auto-dismissing toast notification
5. THE System SHALL attempt automatic reconnection with exponential backoff (1s, 2s, 4s, 8s, max 30s)
6. WHEN reconnection succeeds, THE System SHALL display a success notification
7. THE Status_Indicator SHALL be clickable to show detailed connection info (latency, message queue)

### Requirement 16: 消息搜索与过滤

**User Story:** As a 运维工程师, I want 快速找到历史消息中的关键信息, so that 不需要手动翻阅长对话。

#### Acceptance Criteria

1. THE Chat_Window SHALL provide a search input that opens with `Ctrl+F` or `Cmd+F`
2. THE Search SHALL support searching by message content, command output, and file names
3. THE Search SHALL highlight matching text in messages
4. THE Search SHALL support advanced filters: time range, tool type, host, error level
5. THE Search SHALL display results count and navigation (previous/next)
6. THE Search SHALL preserve search state when switching conversations
7. WHEN a search result is clicked, THE System SHALL scroll to and highlight the message

### Requirement 17: 小屏幕历史面板访问

**User Story:** As a 运维工程师, I want 在小屏幕设备上也能访问历史记录, so that 可以切换会话。

#### Acceptance Criteria

1. WHEN screen width is less than 1200px, THE System SHALL display a hamburger menu button in the top-left
2. WHEN the hamburger menu is clicked, THE History_Panel SHALL slide in from the left as an overlay
3. THE History_Panel_Overlay SHALL have a maximum width of 280px
4. THE History_Panel_Overlay SHALL include a close button and close on backdrop click
5. THE History_Panel_Overlay SHALL support swipe-to-close on touch devices

---

## 优化说明

### 需求冲突修复

1. **Req 2 vs Req 6 高度冲突**: 统一代码块/工具调用卡片最大高度为 300px
2. **Req 4 vs Req 8 刷新时序**: 异常状态时暂停自动刷新，避免覆盖扩展状态
3. **Req 3 / 触发逻辑**: 明确只在输入开头或空格后触发 Slash 菜单，避免与文件路径冲突
4. **"一键执行" 安全性**: 改为"推荐执行"，所有命令需经过风险评估和用户确认

### 实施优先级

| 阶段 | 需求 | 理由 |
|------|------|------|
| **P0 - MVP** | Req 1, Req 6, Req 5, Req 12 | 核心布局和基础体验 |
| **P0 - MVP** | Req 10, Req 11 | 安全必须先行 |
| **P1 - 增强** | Req 3, Req 2, Req 8, Req 17 | 交互优化 |
| **P1 - 增强** | Req 13, Req 14, Req 15 | 用户体验 |
| **P2 - 高级** | Req 4, Req 7, Req 9, Req 16 | 需要更多后端支持 |

### 复杂度评估

| 需求 | 复杂度 | 技术挑战 | 依赖 |
|------|--------|----------|------|
| Req 1 三栏布局 | ⭐⭐⭐ | 响应式断点、面板动画 | - |
| Req 2 卡片运维化 | ⭐⭐⭐⭐ | 聚合状态、进度同步 | SSE 事件 |
| Req 3 输入增强 | ⭐⭐⭐⭐ | 光标追踪、状态共存 | 主机列表 API |
| Req 4 监控面板 | ⭐⭐⭐⭐⭐ | WebSocket、数据聚合 | 监控数据流 |
| Req 5 思考状态 | ⭐⭐⭐ | 后端事件支持 | 思考事件增强 |
| Req 6 宽度优化 | ⭐ | CSS 调整 | - |
| Req 7 Diff 视图 | ⭐⭐⭐⭐⭐ | Diff 算法、大文件优化 | 文件操作 API |
| Req 8 静默模式 | ⭐⭐⭐ | 动画、时序控制 | Req 4 |
| Req 9 快捷终端 | ⭐⭐⭐⭐ | 终端模拟、命令历史 | WebSocket 终端 |
| Req 10 审计日志 | ⭐⭐⭐ | 日志存储、筛选 | - |
| Req 11 二次确认 | ⭐⭐ | 对话框交互 | - |
| Req 12 持久化 | ⭐⭐⭐ | IndexedDB | - |
| Req 13 深色模式 | ⭐⭐ | CSS 变量 | - |
| Req 14 快捷键 | ⭐⭐⭐ | 快捷键管理 | - |
| Req 15 连接状态 | ⭐⭐ | WebSocket 监听 | - |
| Req 16 消息搜索 | ⭐⭐⭐ | 搜索算法、高亮 | - |
| Req 17 小屏访问 | ⭐⭐ | 抽屉动画 | - |

### 技术栈建议

- **状态管理**: Pinia
- **布局方案**: CSS Grid (使用 `fr` 单位自适应)
- **组件库**: Element Plus
- **Diff 库**: diff2html
- **监控图表**: ECharts
- **主题系统**: CSS Variables + Tailwind CSS dark mode
- **持久化**: IndexedDB (Dexie.js)
- **实时通信**: WebSocket / Server-Sent Events
- **命令执行**: WebSocket 终端 (xterm.js)

### 安全考量

1. **命令白名单**: 对于"一键执行"功能，维护可安全执行的命令白名单
2. **危险模式检测**: 检测 `rm -rf`、`dd`、`mkfs` 等危险命令模式
3. **权限验证**: 所有命令执行前验证用户对目标主机的操作权限
4. **审计追踪**: 记录所有敏感操作，支持事后审计
5. **XSS 防护**: 命令输出渲染时进行转义，防止 XSS 攻击

### 性能优化建议

1. **虚拟滚动**: 消息列表使用虚拟滚动处理长对话
2. **按需加载**: 监控面板使用 IntersectionObserver 延迟加载图表
3. **防抖节流**: 输入框、监控刷新等高频操作使用防抖节流
4. **代码分割**: 按路由/功能进行代码分割，减少首屏加载
5. **大文件处理**: Diff 视图对大文件显示警告，仅渲染修改区域

### 可访问性 (A11y)

1. **键盘导航**: 所有交互功能支持键盘操作
2. **ARIA 标签**: 为屏幕阅读器添加适当的 ARIA 标签
3. **焦点管理**: 模态框打开/关闭时正确管理焦点
4. **颜色对比**: 确保文字和背景对比度符合 WCAG AA 标准
5. **动画控制**: 提供"减少动画"选项支持用户偏好设置
