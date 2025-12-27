package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSystemHandler() (*SystemHandler, *gin.Engine) {
	configRepo := newMockConfigRepository()
	handler := NewSystemHandler("0.1.0-test", nil, nil, configRepo)

	r := gin.New()
	api := r.Group("/api")
	{
		api.GET("/health", handler.Health)
		system := api.Group("/system")
		{
			system.GET("/info", handler.GetSystemInfo)
			system.GET("/config", handler.GetConfig)
			system.PUT("/config", handler.UpdateConfig)
		}
	}

	return handler, r
}

// TestHealth 测试健康检查
func TestHealth(t *testing.T) {
	_, r := setupSystemHandler()

	req, _ := http.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "ok", data["status"])
	assert.Equal(t, "0.1.0-test", data["version"])
	assert.NotEmpty(t, data["uptime"])
	assert.NotEmpty(t, data["time"])
}

// TestGetSystemInfo 测试获取系统信息
func TestGetSystemInfo(t *testing.T) {
	_, r := setupSystemHandler()

	req, _ := http.NewRequest("GET", "/api/system/info", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "0.1.0-test", data["version"])
	assert.NotEmpty(t, data["go_version"])
	assert.NotEmpty(t, data["os"])
	assert.NotEmpty(t, data["arch"])
	assert.NotNil(t, data["cpus"])
	assert.NotNil(t, data["goroutines"])
	assert.NotNil(t, data["memory"])
	assert.NotEmpty(t, data["uptime"])
	assert.NotEmpty(t, data["start_time"])

	// 检查内存信息
	memory := data["memory"].(map[string]interface{})
	assert.NotNil(t, memory["alloc"])
	assert.NotNil(t, memory["total_alloc"])
	assert.NotNil(t, memory["sys"])
}

// TestGetConfig 测试获取配置
func TestGetConfig(t *testing.T) {
	_, r := setupSystemHandler()

	req, _ := http.NewRequest("GET", "/api/system/config", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.NotNil(t, data["server"])
	assert.NotNil(t, data["llm"])
	assert.NotNil(t, data["agent"])
}

// TestUpdateConfig 测试更新配置
func TestUpdateConfig(t *testing.T) {
	_, r := setupSystemHandler()

	body := map[string]interface{}{
		"agent": map[string]interface{}{
			"max_loops": 20,
		},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/system/config", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
}

// TestUpdateConfig_InvalidJSON 测试更新配置无效JSON
func TestUpdateConfig_InvalidJSON(t *testing.T) {
	_, r := setupSystemHandler()

	req, _ := http.NewRequest("PUT", "/api/system/config", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}
