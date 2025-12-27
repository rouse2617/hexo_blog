package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTool 测试用的模拟工具
type MockTool struct {
	name        string
	description string
	params      []tool.Parameter
	execFunc    func(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error)
}

func (m *MockTool) Name() string                 { return m.name }
func (m *MockTool) Description() string          { return m.description }
func (m *MockTool) Parameters() []tool.Parameter { return m.params }
func (m *MockTool) Execute(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
	if m.execFunc != nil {
		return m.execFunc(ctx, params)
	}
	return tool.NewResult(nil, "executed"), nil
}

func setupToolHandler() (*ToolHandler, *gin.Engine, *tool.Registry) {
	pool := ssh.NewPool(ssh.Config{})
	registry := tool.NewRegistry()

	// 注册测试工具
	mockTool := &MockTool{
		name:        "test_tool",
		description: "A test tool for unit testing",
		params: []tool.Parameter{
			{Name: "message", Type: "string", Description: "Test message", Required: true},
			{Name: "count", Type: "int", Description: "Count", Required: false, Default: 1},
		},
		execFunc: func(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
			msg := tool.GetStringParam(params, "message", "")
			count := tool.GetIntParam(params, "count", 1)
			return tool.NewResult(map[string]interface{}{
				"message": msg,
				"count":   count,
			}, "success"), nil
		},
	}
	registry.RegisterBuiltin(mockTool)

	// 注册一个会失败的工具
	failTool := &MockTool{
		name:        "fail_tool",
		description: "A tool that always fails",
		params:      []tool.Parameter{},
		execFunc: func(ctx *tool.Context, params map[string]interface{}) (*tool.Result, error) {
			return tool.NewErrorResult("intentional failure"), nil
		},
	}
	registry.RegisterBuiltin(failTool)

	handler := NewToolHandler(registry, pool)

	r := gin.New()
	api := r.Group("/api")
	{
		tools := api.Group("/tools")
		{
			tools.GET("", handler.ListTools)
			tools.GET("/:name", handler.GetTool)
			tools.POST("/:name/execute", handler.ExecuteTool)
		}
	}

	return handler, r, registry
}

// TestListTools 测试获取工具列表
func TestListTools(t *testing.T) {
	_, r, _ := setupToolHandler()

	req, _ := http.NewRequest("GET", "/api/tools", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, float64(2), data["total"])
	tools := data["tools"].([]interface{})
	assert.Len(t, tools, 2)
}

// TestGetTool_Success 测试获取单个工具成功
func TestGetTool_Success(t *testing.T) {
	_, r, _ := setupToolHandler()

	req, _ := http.NewRequest("GET", "/api/tools/test_tool", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "test_tool", data["name"])
	assert.Equal(t, "A test tool for unit testing", data["description"])
	params := data["parameters"].([]interface{})
	assert.Len(t, params, 2)
}

// TestGetTool_NotFound 测试获取不存在的工具
func TestGetTool_NotFound(t *testing.T) {
	_, r, _ := setupToolHandler()

	req, _ := http.NewRequest("GET", "/api/tools/non_existent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeNotFound, resp.Code)
}

// TestExecuteTool_Success 测试执行工具成功
func TestExecuteTool_Success(t *testing.T) {
	_, r, _ := setupToolHandler()

	body := map[string]interface{}{
		"params": map[string]interface{}{
			"message": "hello world",
			"count":   5,
		},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tools/test_tool/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, true, data["success"])
	assert.Equal(t, "success", data["message"])
	resultData := data["data"].(map[string]interface{})
	assert.Equal(t, "hello world", resultData["message"])
	assert.Equal(t, float64(5), resultData["count"])
}

// TestExecuteTool_WithDefaultParams 测试执行工具使用默认参数
func TestExecuteTool_WithDefaultParams(t *testing.T) {
	_, r, _ := setupToolHandler()

	body := map[string]interface{}{
		"params": map[string]interface{}{
			"message": "test message",
		},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tools/test_tool/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	resultData := data["data"].(map[string]interface{})
	assert.Equal(t, float64(1), resultData["count"]) // 默认值
}

// TestExecuteTool_NotFound 测试执行不存在的工具
func TestExecuteTool_NotFound(t *testing.T) {
	_, r, _ := setupToolHandler()

	body := map[string]interface{}{
		"params": map[string]interface{}{},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tools/non_existent/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeExecError, resp.Code)
}

// TestExecuteTool_Failure 测试执行工具失败
func TestExecuteTool_Failure(t *testing.T) {
	_, r, _ := setupToolHandler()

	body := map[string]interface{}{
		"params": map[string]interface{}{},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tools/fail_tool/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeExecError, resp.Code)
	assert.Contains(t, resp.Message, "intentional failure")
}

// TestExecuteTool_InvalidJSON 测试执行工具无效JSON
func TestExecuteTool_InvalidJSON(t *testing.T) {
	_, r, _ := setupToolHandler()

	req, _ := http.NewRequest("POST", "/api/tools/test_tool/execute", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}

// TestExecuteTool_EmptyParams 测试执行工具空参数
func TestExecuteTool_EmptyParams(t *testing.T) {
	_, r, _ := setupToolHandler()

	body := map[string]interface{}{}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tools/test_tool/execute", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	// 应该成功，因为 params 可以为空（使用默认值）
	assert.Equal(t, CodeSuccess, resp.Code)
}
