package model

import (
	"time"
)

// Analysis 分析历史模型
type Analysis struct {
	ID            string    `json:"id" gorm:"primaryKey"`
	SessionID     string    `json:"session_id" gorm:"index"`              // 会话ID（可选）
	ResultsData   string    `json:"results_data" gorm:"type:text"`        // 执行结果数据（JSON格式）
	AnalysisResult string   `json:"analysis_result" gorm:"type:text"`     // 分析结果
	Question      string    `json:"question" gorm:"type:text"`            // 用户提问（可选）
	CreatedAt     time.Time `json:"created_at"`
}







