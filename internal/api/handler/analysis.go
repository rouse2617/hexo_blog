package handler

import (
	"encoding/json"
	"fmt"
	"time"

	"ai-ops/internal/llm"
	"ai-ops/internal/model"
	"ai-ops/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AnalysisHandler AI分析处理器
type AnalysisHandler struct {
	llmClient    *llm.OpenAIClient
	analysisRepo repository.AnalysisRepository
}

// NewAnalysisHandler 创建AI分析处理器
func NewAnalysisHandler(llmClient *llm.OpenAIClient, analysisRepo repository.AnalysisRepository) *AnalysisHandler {
	return &AnalysisHandler{
		llmClient:    llmClient,
		analysisRepo: analysisRepo,
	}
}

// AnalyzeRequest 分析请求
type AnalyzeRequest struct {
	Results  []interface{} `json:"results" binding:"required"` // 执行结果数据
	Question string       `json:"question"`                    // 用户提问（可选）
}

// AnalyzeResponse 分析响应
type AnalyzeResponse struct {
	ID            string `json:"id"`
	AnalysisResult string `json:"analysis_result"`
	Question      string `json:"question,omitempty"`
	CreatedAt     string `json:"created_at"`
}

// Analyze 分析批量执行结果
// POST /api/analysis/analyze
func (h *AnalysisHandler) Analyze(c *gin.Context) {
	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		ParamError(c, "参数错误: "+err.Error())
		return
	}

	if len(req.Results) == 0 {
		ParamError(c, "结果数据不能为空")
		return
	}

	// 将结果转换为JSON字符串用于提示词
	resultsJSON, err := json.MarshalIndent(req.Results, "", "  ")
	if err != nil {
		ExecError(c, "序列化结果失败: "+err.Error())
		return
	}

	// 构建分析提示词
	prompt := h.buildAnalysisPrompt(string(resultsJSON), req.Question)

	// 调用LLM进行分析
	ctx := c.Request.Context()
	response, err := h.llmClient.Chat(ctx, []llm.Message{
		{
			Role:    "user",
			Content: prompt,
		},
	})

	if err != nil {
		LLMError(c, "AI分析失败: "+err.Error())
		return
	}

	analysisResult := response.Message.Content
	if analysisResult == "" {
		ExecError(c, "AI未返回分析结果")
		return
	}

	// 保存分析历史
	analysisID := uuid.New().String()
	analysis := &model.Analysis{
		ID:            analysisID,
		SessionID:     "", // 控制台操作不关联会话
		ResultsData:   string(resultsJSON),
		AnalysisResult: analysisResult,
		Question:      req.Question,
		CreatedAt:     time.Now(),
	}

	if err := h.analysisRepo.Create(analysis); err != nil {
		// 分析成功但保存失败，仍然返回结果
		// 记录错误但不影响返回结果
		_ = err
	}

	Success(c, AnalyzeResponse{
		ID:            analysisID,
		AnalysisResult: analysisResult,
		Question:      req.Question,
		CreatedAt:     analysis.CreatedAt.Format(time.RFC3339),
	})
}

// buildAnalysisPrompt 构建分析提示词
func (h *AnalysisHandler) buildAnalysisPrompt(resultsJSON string, question string) string {
	basePrompt := `你是一个专业的运维分析助手。请分析以下批量操作的结果数据，提供专业的分析报告。

结果数据：
` + resultsJSON + `

请从以下角度进行分析：
1. 执行概况：成功/失败数量统计
2. 问题识别：找出异常、错误或潜在问题
3. 性能分析：对比不同节点的性能指标（如适用）
4. 建议措施：针对发现的问题提供解决建议

请用中文回答，格式清晰，分点说明。`

	if question != "" {
		basePrompt += "\n\n用户特别关注的问题：" + question
	}

	return basePrompt
}

// GetHistory 获取分析历史
// GET /api/analysis/history
func (h *AnalysisHandler) GetHistory(c *gin.Context) {
	sessionID := c.Query("session_id")
	limitStr := c.DefaultQuery("limit", "20")
	
	limit := 20
	if parsedLimit, err := parseInt(limitStr, 10); err == nil {
		limit = parsedLimit
	}

	analyses, err := h.analysisRepo.List(sessionID, limit)
	if err != nil {
		ExecError(c, "获取分析历史失败: "+err.Error())
		return
	}

	Success(c, analyses)
}

// GetAnalysis 获取单个分析记录
// GET /api/analysis/:id
func (h *AnalysisHandler) GetAnalysis(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		ParamError(c, "分析ID不能为空")
		return
	}

	analysis, err := h.analysisRepo.GetByID(id)
	if err != nil {
		NotFound(c, "分析记录不存在")
		return
	}

	Success(c, analysis)
}

// DeleteAnalysis 删除分析记录
// DELETE /api/analysis/:id
func (h *AnalysisHandler) DeleteAnalysis(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		ParamError(c, "分析ID不能为空")
		return
	}

	if err := h.analysisRepo.Delete(id); err != nil {
		ExecError(c, "删除分析记录失败: "+err.Error())
		return
	}

	Success(c, gin.H{"message": "删除成功"})
}

// parseInt 解析整数（辅助函数）
func parseInt(s string, base int) (int, error) {
	var result int
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}

