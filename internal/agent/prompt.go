package agent

import (
	"fmt"
	"strings"

	"ai-ops/internal/tool"
)

// SystemPromptTemplate 系统提示词模板
const SystemPromptTemplate = `你是一个专业的运维助手 Agent，名为 AI-Ops。

## 你的能力
- 你可以通过调用工具来执行各种运维操作
- 你可以查看日志、检查系统状态、执行命令等
- 你可以同时操作多个服务器节点

## 工作方式
1. 理解用户的运维需求
2. 选择合适的工具来完成任务
3. 分析工具返回的结果
4. 如果需要更多信息，继续调用工具
5. 最终给出清晰的总结和建议

## 可用工具
%s

## 注意事项
- 执行危险操作前要确认
- 如果不确定，先查询信息再操作
- 给出的建议要具体可行
- 如果工具执行失败，分析原因并尝试其他方法

## 输出要求
- 使用中文回复
- 结果要简洁明了
- 如果有异常，要指出问题和建议
`

// BuildSystemPrompt 构建系统提示词
func BuildSystemPrompt(registry *tool.Registry) string {
	toolsDesc := registry.GeneratePrompt()
	return fmt.Sprintf(SystemPromptTemplate, toolsDesc)
}

// BuildSystemPromptWithHosts 构建带主机信息的系统提示词
func BuildSystemPromptWithHosts(registry *tool.Registry, hosts []string) string {
	basePrompt := BuildSystemPrompt(registry)

	if len(hosts) > 0 {
		hostInfo := fmt.Sprintf("\n## 当前会话关联的主机\n%s\n", strings.Join(hosts, ", "))
		return basePrompt + hostInfo
	}

	return basePrompt
}

// TaskPromptTemplate 任务提示词模板（用于复杂任务分解）
const TaskPromptTemplate = `## 当前任务
%s

## 任务要求
请按以下步骤完成任务：
1. 分析任务需求
2. 制定执行计划
3. 逐步执行并验证
4. 总结执行结果

如果任务复杂，请分步骤执行，每步完成后汇报进度。
`

// BuildTaskPrompt 构建任务提示词
func BuildTaskPrompt(task string) string {
	return fmt.Sprintf(TaskPromptTemplate, task)
}

// ErrorRecoveryPrompt 错误恢复提示词
const ErrorRecoveryPrompt = `上一个操作执行失败，错误信息：%s

请分析错误原因，并尝试以下方式解决：
1. 检查参数是否正确
2. 尝试其他方法
3. 如果无法解决，向用户说明情况
`

// BuildErrorRecoveryPrompt 构建错误恢复提示词
func BuildErrorRecoveryPrompt(err string) string {
	return fmt.Sprintf(ErrorRecoveryPrompt, err)
}

// SummaryPromptTemplate 总结提示词模板
const SummaryPromptTemplate = `请根据以上执行结果，给出总结：

1. 执行了哪些操作
2. 发现了什么问题
3. 建议采取什么措施

请用简洁的语言总结，突出重点。
`

// PromptBuilder 提示词构建器
type PromptBuilder struct {
	registry *tool.Registry
	hosts    []string
}

// NewPromptBuilder 创建提示词构建器
func NewPromptBuilder(registry *tool.Registry) *PromptBuilder {
	return &PromptBuilder{
		registry: registry,
	}
}

// SetHosts 设置关联主机
func (b *PromptBuilder) SetHosts(hosts []string) *PromptBuilder {
	b.hosts = hosts
	return b
}

// BuildSystem 构建系统提示词
func (b *PromptBuilder) BuildSystem() string {
	return BuildSystemPromptWithHosts(b.registry, b.hosts)
}

// BuildTask 构建任务提示词
func (b *PromptBuilder) BuildTask(task string) string {
	return BuildTaskPrompt(task)
}

// BuildErrorRecovery 构建错误恢复提示词
func (b *PromptBuilder) BuildErrorRecovery(err string) string {
	return BuildErrorRecoveryPrompt(err)
}

// BuildSummary 构建总结提示词
func (b *PromptBuilder) BuildSummary() string {
	return SummaryPromptTemplate
}
