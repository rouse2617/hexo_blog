package agent

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"ai-ops/internal/llm"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"
	"ai-ops/pkg/logger"

	"go.uber.org/zap"
)

// ParallelExecutor 并行工具执行器
type ParallelExecutor struct {
	registry      *tool.Registry
	sshPool       *ssh.Pool
	cache         *ToolCache
	maxConcurrent int
}

// NewParallelExecutor 创建并行执行器
func NewParallelExecutor(registry *tool.Registry, sshPool *ssh.Pool, cache *ToolCache) *ParallelExecutor {
	return &ParallelExecutor{
		registry:      registry,
		sshPool:       sshPool,
		cache:         cache,
		maxConcurrent: 10, // 最大并发数
	}
}

// ExecuteParallel 并行执行工具调用
func (e *ParallelExecutor) ExecuteParallel(ctx context.Context, toolCalls []llm.ToolCall, hosts []string) ([]llm.Message, []ToolCallRecord) {
	if len(toolCalls) == 0 {
		return nil, nil
	}

	// 单个工具调用直接执行
	if len(toolCalls) == 1 {
		return e.executeSingle(ctx, toolCalls[0], hosts)
	}

	// 多个工具并行执行
	return e.executeMultiple(ctx, toolCalls, hosts)
}

// executeSingle 执行单个工具
func (e *ParallelExecutor) executeSingle(ctx context.Context, tc llm.ToolCall, hosts []string) ([]llm.Message, []ToolCallRecord) {
	msg, record := e.executeOne(ctx, tc, hosts)
	return []llm.Message{msg}, []ToolCallRecord{record}
}

// executeMultiple 并行执行多个工具
func (e *ParallelExecutor) executeMultiple(ctx context.Context, toolCalls []llm.ToolCall, hosts []string) ([]llm.Message, []ToolCallRecord) {
	results := make([]struct {
		msg    llm.Message
		record ToolCallRecord
	}, len(toolCalls))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, e.maxConcurrent)

	// 添加 context 取消监听，确保 context 取消时所有 goroutine 能快速退出
	for i, tc := range toolCalls {
		// 检查 context 是否已取消
		select {
		case <-ctx.Done():
			// context 已取消，为剩余的工具填充错误结果
			for j := i; j < len(toolCalls); j++ {
				results[j] = struct {
					msg    llm.Message
					record ToolCallRecord
				}{
					msg: llm.NewToolMessage(tc.ID, tc.Function.Name, "执行取消"),
					record: ToolCallRecord{
						ID:    tc.ID,
						Tool:  tc.Function.Name,
						Error: "执行取消",
					},
				}
			}
			break
		default:
		}

		wg.Add(1)
		go func(idx int, toolCall llm.ToolCall) {
			defer wg.Done()

			// 获取信号量
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				// context 取消，不再等待信号量
				results[idx] = struct {
					msg    llm.Message
					record ToolCallRecord
				}{
					msg: llm.NewToolMessage(toolCall.ID, toolCall.Function.Name, "执行取消"),
					record: ToolCallRecord{
						ID:    toolCall.ID,
						Tool:  toolCall.Function.Name,
						Error: "执行取消",
					},
				}
				return
			}

			// 使用带 context 的执行
			msg, record := e.executeOne(ctx, toolCall, hosts)
			results[idx] = struct {
				msg    llm.Message
				record ToolCallRecord
			}{msg, record}
		}(i, tc)
	}

	wg.Wait()

	// 收集结果（保持顺序）
	messages := make([]llm.Message, len(toolCalls))
	records := make([]ToolCallRecord, len(toolCalls))
	for i, r := range results {
		messages[i] = r.msg
		records[i] = r.record
	}

	return messages, records
}

// executeOne 执行单个工具调用
func (e *ParallelExecutor) executeOne(ctx context.Context, tc llm.ToolCall, hosts []string) (llm.Message, ToolCallRecord) {
	// 解析参数
	var params map[string]interface{}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &params); err != nil {
		logger.Error("解析工具参数失败", zap.Error(err), zap.String("tool", tc.Function.Name))
		record := ToolCallRecord{
			ID:     tc.ID,
			Tool:   tc.Function.Name,
			Params: nil,
			Error:  "参数解析失败: " + err.Error(),
		}
		return llm.NewToolMessage(tc.ID, tc.Function.Name, "错误: 参数解析失败"), record
	}

	// 检查缓存
	if e.cache != nil {
		if cachedResult, cachedErr, hit := e.cache.Get(tc.Function.Name, params); hit {
			logger.Debug("工具缓存命中",
				zap.String("tool", tc.Function.Name),
				zap.Any("params", params),
			)
			record := ToolCallRecord{
				ID:     tc.ID,
				Tool:   tc.Function.Name,
				Params: params,
				Result: cachedResult,
				Error:  cachedErr,
			}
			var resultStr string
			if cachedErr != "" {
				resultStr = "执行失败(缓存): " + cachedErr
			} else {
				resultStr = "执行成功(缓存): " + cachedResult
			}
			return llm.NewToolMessage(tc.ID, tc.Function.Name, resultStr), record
		}
	}

	// 获取工具特定超时
	timeout := GetToolTimeout(tc.Function.Name)
	toolCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 创建工具执行上下文
	toolContext := &tool.Context{
		Hosts:   hosts,
		SSH:     e.sshPool,
		Timeout: timeout,
	}

	logger.Debug("执行工具",
		zap.String("tool", tc.Function.Name),
		zap.Any("params", params),
		zap.Duration("timeout", timeout),
	)

	// 执行工具
	startTime := time.Now()
	result, err := e.registry.Execute(toolContext, tc.Function.Name, params)
	elapsed := time.Since(startTime)

	record := ToolCallRecord{
		ID:     tc.ID,
		Tool:   tc.Function.Name,
		Params: params,
	}

	var resultStr string

	// 检查上下文是否超时
	if toolCtx.Err() == context.DeadlineExceeded {
		record.Error = "执行超时: " + timeout.String()
		resultStr = "执行超时: " + timeout.String()
	} else if err != nil {
		record.Error = err.Error()
		resultStr = "执行错误: " + err.Error()
	} else if !result.Success {
		record.Error = result.Error
		resultStr = "执行失败: " + result.Error
	} else {
		resultBytes, _ := json.Marshal(result.Data)
		record.Result = string(resultBytes)
		resultStr = "执行成功: " + result.Message + "\n结果: " + string(resultBytes)
	}

	logger.Debug("工具执行完成",
		zap.String("tool", tc.Function.Name),
		zap.Duration("elapsed", elapsed),
		zap.Bool("success", record.Error == ""),
	)

	// 写入缓存
	if e.cache != nil {
		e.cache.Set(tc.Function.Name, params, record.Result, record.Error)
	}

	return llm.NewToolMessage(tc.ID, tc.Function.Name, resultStr), record
}

// SetMaxConcurrent 设置最大并发数
func (e *ParallelExecutor) SetMaxConcurrent(n int) {
	if n > 0 {
		e.maxConcurrent = n
	}
}
