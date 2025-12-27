package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-ops/internal/agent"
	"ai-ops/internal/llm"
	"ai-ops/internal/ssh"
	"ai-ops/internal/tool"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupChatHandler() (*ChatHandler, *gin.Engine) {
	// 创建依赖
	pool := ssh.NewPool(ssh.Config{})
	registry := tool.NewRegistry()

	// 使用 mock LLM client
	llmClient := llm.NewOpenAIClient(llm.OpenAIConfig{
		Endpoint: "http://localhost:11434/v1",
		Model:    "test-model",
	})

	aiAgent := agent.NewAgent(llmClient, registry, pool, agent.Config{
		MaxLoops: 5,
	})

	handler := NewChatHandler(aiAgent)

	r := gin.New()
	api := r.Group("/api")
	{
		chat := api.Group("/chat")
		{
			chat.POST("", handler.Chat)
			chat.POST("/stream", handler.StreamChat)
			chat.GET("/sessions", handler.GetSessions)
			chat.GET("/history", handler.GetHistory)
			chat.DELETE("/sessions/:id", handler.DeleteSession)
		}
	}

	return handler, r
}

// TestGetSessions 测试获取会话列表
func TestGetSessions(t *testing.T) {
	_, r := setupChatHandler()

	req, _ := http.NewRequest("GET", "/api/chat/sessions", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.NotNil(t, data["sessions"])
}

// TestGetHistory_MissingSessionID 测试获取历史缺少session_id
func TestGetHistory_MissingSessionID(t *testing.T) {
	_, r := setupChatHandler()

	req, _ := http.NewRequest("GET", "/api/chat/history", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}

// TestGetHistory_WithSessionID 测试获取历史带session_id
func TestGetHistory_WithSessionID(t *testing.T) {
	_, r := setupChatHandler()

	req, _ := http.NewRequest("GET", "/api/chat/history?session_id=test-session", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	data := resp.Data.(map[string]interface{})
	assert.Equal(t, "test-session", data["session_id"])
	assert.NotNil(t, data["messages"])
}

// TestDeleteSession_Success 测试删除会话成功
func TestDeleteSession_Success(t *testing.T) {
	_, r := setupChatHandler()

	req, _ := http.NewRequest("DELETE", "/api/chat/sessions/test-session", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
}

// TestChat_MissingMessage 测试对话缺少消息
func TestChat_MissingMessage(t *testing.T) {
	_, r := setupChatHandler()

	body := map[string]interface{}{
		"session_id": "test-session",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/chat", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}

// TestChat_InvalidJSON 测试对话无效JSON
func TestChat_InvalidJSON(t *testing.T) {
	_, r := setupChatHandler()

	req, _ := http.NewRequest("POST", "/api/chat", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}

// TestStreamChat_MissingMessage 测试流式对话缺少消息
func TestStreamChat_MissingMessage(t *testing.T) {
	_, r := setupChatHandler()

	body := map[string]interface{}{
		"session_id": "test-session",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/chat/stream", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}

// TestStreamChat_InvalidJSON 测试流式对话无效JSON
func TestStreamChat_InvalidJSON(t *testing.T) {
	_, r := setupChatHandler()

	req, _ := http.NewRequest("POST", "/api/chat/stream", bytes.NewBuffer([]byte("invalid")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
}
