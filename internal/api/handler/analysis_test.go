package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-ops/internal/llm"
	"ai-ops/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// MockAnalysisRepository 模拟分析仓库
type MockAnalysisRepository struct {
	CreateFunc  func(analysis *model.Analysis) error
	GetByIDFunc func(id string) (*model.Analysis, error)
	ListFunc    func(sessionID string, limit int) ([]*model.Analysis, error)
	DeleteFunc  func(id string) error
	analyses    map[string]*model.Analysis
}

func NewMockAnalysisRepository() *MockAnalysisRepository {
	return &MockAnalysisRepository{
		analyses: make(map[string]*model.Analysis),
	}
}

func (m *MockAnalysisRepository) Create(analysis *model.Analysis) error {
	if m.CreateFunc != nil {
		return m.CreateFunc(analysis)
	}
	if m.analyses == nil {
		m.analyses = make(map[string]*model.Analysis)
	}
	m.analyses[analysis.ID] = analysis
	return nil
}

func (m *MockAnalysisRepository) GetByID(id string) (*model.Analysis, error) {
	if m.GetByIDFunc != nil {
		return m.GetByIDFunc(id)
	}
	if analysis, ok := m.analyses[id]; ok {
		return analysis, nil
	}
	return nil, gorm.ErrRecordNotFound
}

func (m *MockAnalysisRepository) List(sessionID string, limit int) ([]*model.Analysis, error) {
	if m.ListFunc != nil {
		return m.ListFunc(sessionID, limit)
	}
	analyses := make([]*model.Analysis, 0)
	for _, analysis := range m.analyses {
		if sessionID == "" || analysis.SessionID == sessionID {
			analyses = append(analyses, analysis)
		}
	}
	if limit > 0 && len(analyses) > limit {
		analyses = analyses[:limit]
	}
	return analyses, nil
}

func (m *MockAnalysisRepository) Delete(id string) error {
	if m.DeleteFunc != nil {
		return m.DeleteFunc(id)
	}
	delete(m.analyses, id)
	return nil
}

func setupAnalysisHandlerWithMockLLM(mockServer *httptest.Server) (*AnalysisHandler, *gin.Engine, *MockAnalysisRepository) {
	mockRepo := NewMockAnalysisRepository()
	// 使用mock server的URL创建真实的OpenAIClient
	realLLM := llm.NewOpenAIClient(llm.OpenAIConfig{
		Endpoint: mockServer.URL,
		Model:    "test",
		Timeout:  5 * time.Second,
	})
	handler := NewAnalysisHandler(realLLM, mockRepo)

	r := gin.New()
	api := r.Group("/api")
	{
		analysis := api.Group("/analysis")
		{
			analysis.POST("/analyze", handler.Analyze)
			analysis.GET("/history", handler.GetHistory)
			analysis.GET("/:id", handler.GetAnalysis)
			analysis.DELETE("/:id", handler.DeleteAnalysis)
		}
	}

	return handler, r, mockRepo
}

func TestAnalysisHandler_Analyze_Success(t *testing.T) {
	// 创建mock HTTP server来模拟LLM响应
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"id":    "test-id",
			"model": "test",
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "分析结果：所有节点运行正常",
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     10,
				"completion_tokens": 20,
				"total_tokens":      30,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	_, router, mockRepo := setupAnalysisHandlerWithMockLLM(mockServer)

	reqBody := AnalyzeRequest{
		Results: []interface{}{
			map[string]interface{}{
				"host":   "host1",
				"status": "success",
			},
			map[string]interface{}{
				"host":   "host2",
				"status": "success",
			},
		},
		Question: "请分析这些节点的状态",
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/analysis/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	// 验证返回的分析结果
	var analysisResp AnalyzeResponse
	respData, _ := json.Marshal(resp.Data)
	err = json.Unmarshal(respData, &analysisResp)
	require.NoError(t, err)
	assert.NotEmpty(t, analysisResp.ID)
	assert.Contains(t, analysisResp.AnalysisResult, "所有节点运行正常")
	assert.Equal(t, "请分析这些节点的状态", analysisResp.Question)

	// 验证已保存到仓库
	saved, err := mockRepo.GetByID(analysisResp.ID)
	require.NoError(t, err)
	assert.NotNil(t, saved)
}

func TestAnalysisHandler_Analyze_EmptyResults(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	_, router, _ := setupAnalysisHandlerWithMockLLM(mockServer)

	reqBody := AnalyzeRequest{
		Results: []interface{}{}, // 空结果
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/analysis/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeParamError, resp.Code)
	assert.Contains(t, resp.Message, "结果数据不能为空")
}

func TestAnalysisHandler_Analyze_InvalidRequest(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	_, router, _ := setupAnalysisHandlerWithMockLLM(mockServer)

	// 缺少必需字段
	reqBody := map[string]interface{}{}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/analysis/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code) // 所有响应都返回200，错误码在响应体中

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.NotEqual(t, CodeSuccess, resp.Code)
}

func TestAnalysisHandler_Analyze_LLMError(t *testing.T) {
	// 创建一个返回错误的mock server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer mockServer.Close()

	_, router, _ := setupAnalysisHandlerWithMockLLM(mockServer)

	reqBody := AnalyzeRequest{
		Results: []interface{}{
			map[string]interface{}{
				"host":   "host1",
				"status": "success",
			},
		},
	}

	body, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/analysis/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeLLMError, resp.Code)
}

func TestAnalysisHandler_GetHistory_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	_, router, mockRepo := setupAnalysisHandlerWithMockLLM(mockServer)

	// 创建一些测试数据
	analysis1 := &model.Analysis{
		ID:             "test-id-1",
		ResultsData:    "{}",
		AnalysisResult: "Test analysis 1",
		CreatedAt:      time.Now(),
	}
	analysis2 := &model.Analysis{
		ID:             "test-id-2",
		ResultsData:    "{}",
		AnalysisResult: "Test analysis 2",
		CreatedAt:      time.Now().Add(-time.Hour),
	}
	mockRepo.Create(analysis1)
	mockRepo.Create(analysis2)

	req, _ := http.NewRequest("GET", "/api/analysis/history?limit=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	analyses, ok := resp.Data.([]interface{})
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(analyses), 2)
}

func TestAnalysisHandler_GetAnalysis_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	_, router, mockRepo := setupAnalysisHandlerWithMockLLM(mockServer)

	analysis := &model.Analysis{
		ID:             "test-id-1",
		ResultsData:    "{}",
		AnalysisResult: "Test analysis",
		CreatedAt:      time.Now(),
	}
	mockRepo.Create(analysis)

	req, _ := http.NewRequest("GET", "/api/analysis/test-id-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)
}

func TestAnalysisHandler_GetAnalysis_NotFound(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	_, router, _ := setupAnalysisHandlerWithMockLLM(mockServer)

	req, _ := http.NewRequest("GET", "/api/analysis/non-existent-id", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeNotFound, resp.Code)
}

func TestAnalysisHandler_DeleteAnalysis_Success(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer mockServer.Close()

	_, router, mockRepo := setupAnalysisHandlerWithMockLLM(mockServer)

	analysis := &model.Analysis{
		ID:             "test-id-1",
		ResultsData:    "{}",
		AnalysisResult: "Test analysis",
		CreatedAt:      time.Now(),
	}
	mockRepo.Create(analysis)

	req, _ := http.NewRequest("DELETE", "/api/analysis/test-id-1", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, CodeSuccess, resp.Code)

	// 验证已删除
	_, err = mockRepo.GetByID("test-id-1")
	assert.Error(t, err)
}
