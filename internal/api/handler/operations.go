package handler

import (
	"time"

	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"

	"github.com/gin-gonic/gin"
)

// OperationsHandler 批量操作处理器
type OperationsHandler struct {
	registry *tool.Registry
	sshPool  *ssh.Pool
}

// NewOperationsHandler 创建批量操作处理器
func NewOperationsHandler(registry *tool.Registry, sshPool *ssh.Pool) *OperationsHandler {
	return &OperationsHandler{
		registry: registry,
		sshPool:  sshPool,
	}
}

// BatchExecuteRequest 批量执行请求
type BatchExecuteRequest struct {
	Operation string                 `json:"operation" binding:"required"` // query_log | run_command | check_cpu | check_memory
	Hosts     []string               `json:"hosts" binding:"required"`     // 目标节点列表
	Params    map[string]interface{} `json:"params"`                       // 操作参数
}

// BatchExecuteResult 批量执行结果
type BatchExecuteResult struct {
	Host    string      `json:"host"`
	Status  string      `json:"status"` // success | error
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
	Elapsed string      `json:"elapsed"`
}

// BatchExecute 批量执行操作
// POST /api/operations/batch-execute
func (h *OperationsHandler) BatchExecute(c *gin.Context) {
	var req BatchExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	// 验证操作类型
	supportedOperations := map[string]bool{
		"query_log":     true,
		"run_command":   true,
		"check_cpu":     true,
		"check_memory":  true,
		"check_disk":    true,
		"check_process": true,
	}
	if !supportedOperations[req.Operation] {
		ParamError(c, "不支持的操作类型: "+req.Operation)
		return
	}

	// 检查工具是否存在
	toolInstance, ok := h.registry.Get(req.Operation)
	if !ok {
		NotFound(c, "工具不存在: "+req.Operation)
		return
	}

	// 检查工具是否启用
	if !h.registry.IsEnabled(req.Operation) {
		ParamError(c, "工具已禁用: "+req.Operation)
		return
	}

	// 构建参数，将 hosts 添加到 params 中
	params := make(map[string]interface{})
	if req.Params != nil {
		for k, v := range req.Params {
			params[k] = v
		}
	}
	params["hosts"] = req.Hosts

	// 创建执行上下文
	ctx := &tool.Context{
		Hosts: req.Hosts,
		SSH:   h.sshPool,
	}

	// 执行工具
	start := time.Now()
	result, err := toolInstance.Execute(ctx, params)
	elapsed := time.Since(start)

	if err != nil {
		ExecError(c, "执行失败: "+err.Error())
		return
	}

	// 处理结果
	results := make([]BatchExecuteResult, 0)

	if !result.Success {
		// 工具执行失败，为每个主机返回错误
		for _, host := range req.Hosts {
			results = append(results, BatchExecuteResult{
				Host:    host,
				Status:  "error",
				Error:   result.Error,
				Elapsed: elapsed.String(),
			})
		}
	} else {
		// 根据返回的数据类型处理结果
		switch data := result.Data.(type) {
		case []map[string]interface{}:
			// 多主机结果（数组）
			for _, item := range data {
				host, _ := item["host"].(string)
				if errStr, hasError := item["error"].(string); hasError {
					results = append(results, BatchExecuteResult{
						Host:    host,
						Status:  "error",
						Error:   errStr,
						Result:  item,
						Elapsed: getElapsedFromItem(item),
					})
				} else {
					results = append(results, BatchExecuteResult{
						Host:    host,
						Status:  "success",
						Result:  item,
						Elapsed: getElapsedFromItem(item),
					})
				}
			}
		case map[string]interface{}:
			// 单主机结果（对象），但传入的是多个主机，可能工具不支持批量
			// 这种情况下，为每个主机返回相同结果（通常不应该发生）
			for _, host := range req.Hosts {
				itemHost, _ := data["host"].(string)
				if itemHost == host || itemHost == "" {
					if errStr, hasError := data["error"].(string); hasError {
						results = append(results, BatchExecuteResult{
							Host:    host,
							Status:  "error",
							Error:   errStr,
							Result:  data,
							Elapsed: elapsed.String(),
						})
					} else {
						results = append(results, BatchExecuteResult{
							Host:    host,
							Status:  "success",
							Result:  data,
							Elapsed: elapsed.String(),
						})
					}
					break
				}
			}
			// 如果结果数量不匹配，补充其他主机的错误
			if len(results) < len(req.Hosts) {
				for _, host := range req.Hosts {
					found := false
					for _, r := range results {
						if r.Host == host {
							found = true
							break
						}
					}
					if !found {
						results = append(results, BatchExecuteResult{
							Host:    host,
							Status:  "error",
							Error:   "工具未返回该主机的结果",
							Elapsed: elapsed.String(),
						})
					}
				}
			}
		default:
			// 未知类型，为每个主机返回错误
			for _, host := range req.Hosts {
				results = append(results, BatchExecuteResult{
					Host:    host,
					Status:  "error",
					Error:   "工具返回了不支持的结果格式",
					Elapsed: elapsed.String(),
				})
			}
		}
	}

	Success(c, results)
}

// getElapsedFromItem 从结果项中获取执行时间
func getElapsedFromItem(item map[string]interface{}) string {
	if elapsed, ok := item["elapsed"].(string); ok {
		return elapsed
	}
	return ""
}
