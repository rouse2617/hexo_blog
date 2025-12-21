// Package agent provides streaming response handling for the AI agent engine.
package agent

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/opsgenius/backend/pkg/models"
	"go.uber.org/zap"
)

// StreamProcessor handles streaming response generation and processing.
type StreamProcessor struct {
	logger *zap.Logger
}

// NewStreamProcessor creates a new stream processor.
func NewStreamProcessor(logger *zap.Logger) *StreamProcessor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &StreamProcessor{
		logger: logger,
	}
}

// StreamOptions configures streaming behavior.
type StreamOptions struct {
	SessionID     string
	MessageID     string
	ChunkSize     int  // Size of text chunks for streaming (0 = auto)
	ExtractCharts bool // Whether to extract and send chart data separately
}

// StreamText streams text content in chunks.
// Requirements: 6.1 - 流式方式发送响应片段到前端
func (sp *StreamProcessor) StreamText(ctx context.Context, text string, opts StreamOptions, responseChan chan<- Response) error {
	if opts.MessageID == "" {
		opts.MessageID = uuid.New().String()
	}

	// Default chunk size if not specified
	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 50 // Default: 50 characters per chunk
	}

	// Split text into chunks and send incrementally
	textRunes := []rune(text)
	totalLength := len(textRunes)

	for i := 0; i < totalLength; i += chunkSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		end := i + chunkSize
		if end > totalLength {
			end = totalLength
		}

		chunk := string(textRunes[i:end])

		// Send chunk
		select {
		case responseChan <- Response{
			Type:    ResponseTypeText,
			Content: chunk,
		}:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// StreamWithToolResult streams text with embedded tool result.
// Requirements: 6.2 - 响应中嵌入结构化数据
func (sp *StreamProcessor) StreamWithToolResult(ctx context.Context, text string, toolResult map[string]interface{}, opts StreamOptions, responseChan chan<- Response) error {
	if opts.MessageID == "" {
		opts.MessageID = uuid.New().String()
	}

	// Send text with embedded tool result
	select {
	case responseChan <- Response{
		Type: ResponseTypeText,
		Content: map[string]interface{}{
			"text":       text,
			"toolResult": toolResult,
		},
	}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// ExtractAndStreamChartData extracts chart data from text/data and sends it separately.
// Requirements: 6.3 - 发送单独的图表数据消息
func (sp *StreamProcessor) ExtractAndStreamChartData(ctx context.Context, data interface{}, opts StreamOptions, responseChan chan<- Response) error {
	if opts.MessageID == "" {
		opts.MessageID = uuid.New().String()
	}

	// Try to extract chart data from various formats
	chartData, err := sp.extractChartData(data)
	if err != nil {
		sp.logger.Warn("Failed to extract chart data",
			zap.Error(err),
			zap.String("sessionID", opts.SessionID))
		return err
	}

	if chartData == nil {
		// No chart data found
		return nil
	}

	// Send chart data as separate response
	select {
	case responseChan <- Response{
		Type:    ResponseTypeChart,
		Content: *chartData,
	}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// extractChartData attempts to extract chart data from various input formats.
func (sp *StreamProcessor) extractChartData(data interface{}) (*models.ChartData, error) {
	// If data is already ChartData, return it
	if chartData, ok := data.(models.ChartData); ok {
		return &chartData, nil
	}

	// If data is a pointer to ChartData
	if chartDataPtr, ok := data.(*models.ChartData); ok {
		return chartDataPtr, nil
	}

	// If data is a map, try to convert it
	if dataMap, ok := data.(map[string]interface{}); ok {
		return sp.extractChartDataFromMap(dataMap)
	}

	// If data is a string, try to parse it
	if dataStr, ok := data.(string); ok {
		return sp.extractChartDataFromString(dataStr)
	}

	return nil, fmt.Errorf("unsupported data type for chart extraction: %T", data)
}

// extractChartDataFromMap extracts chart data from a map.
func (sp *StreamProcessor) extractChartDataFromMap(dataMap map[string]interface{}) (*models.ChartData, error) {
	// Check if map contains chart data fields
	if _, hasLabels := dataMap["labels"]; !hasLabels {
		return nil, fmt.Errorf("map does not contain 'labels' field")
	}

	if _, hasDatasets := dataMap["datasets"]; !hasDatasets {
		return nil, fmt.Errorf("map does not contain 'datasets' field")
	}

	// Extract labels
	labelsInterface, ok := dataMap["labels"]
	if !ok {
		return nil, fmt.Errorf("labels field not found")
	}

	labels, err := sp.extractLabels(labelsInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to extract labels: %w", err)
	}

	// Extract datasets
	datasetsInterface, ok := dataMap["datasets"]
	if !ok {
		return nil, fmt.Errorf("datasets field not found")
	}

	datasets, err := sp.extractDatasets(datasetsInterface)
	if err != nil {
		return nil, fmt.Errorf("failed to extract datasets: %w", err)
	}

	return &models.ChartData{
		Labels:   labels,
		Datasets: datasets,
	}, nil
}

// extractLabels extracts labels from various formats.
func (sp *StreamProcessor) extractLabels(labelsInterface interface{}) ([]string, error) {
	// If already []string
	if labels, ok := labelsInterface.([]string); ok {
		return labels, nil
	}

	// If []interface{}
	if labelsSlice, ok := labelsInterface.([]interface{}); ok {
		labels := make([]string, len(labelsSlice))
		for i, label := range labelsSlice {
			if labelStr, ok := label.(string); ok {
				labels[i] = labelStr
			} else {
				labels[i] = fmt.Sprintf("%v", label)
			}
		}
		return labels, nil
	}

	return nil, fmt.Errorf("unsupported labels type: %T", labelsInterface)
}

// extractDatasets extracts datasets from various formats.
func (sp *StreamProcessor) extractDatasets(datasetsInterface interface{}) ([]models.ChartDataset, error) {
	// If already []models.ChartDataset
	if datasets, ok := datasetsInterface.([]models.ChartDataset); ok {
		return datasets, nil
	}

	// If []interface{}
	if datasetsSlice, ok := datasetsInterface.([]interface{}); ok {
		datasets := make([]models.ChartDataset, 0, len(datasetsSlice))
		for _, datasetInterface := range datasetsSlice {
			if datasetMap, ok := datasetInterface.(map[string]interface{}); ok {
				dataset, err := sp.extractDataset(datasetMap)
				if err != nil {
					return nil, fmt.Errorf("failed to extract dataset: %w", err)
				}
				datasets = append(datasets, dataset)
			} else {
				return nil, fmt.Errorf("unsupported dataset type: %T", datasetInterface)
			}
		}
		return datasets, nil
	}

	return nil, fmt.Errorf("unsupported datasets type: %T", datasetsInterface)
}

// extractDataset extracts a single dataset from a map.
func (sp *StreamProcessor) extractDataset(datasetMap map[string]interface{}) (models.ChartDataset, error) {
	// Extract label
	labelInterface, ok := datasetMap["label"]
	if !ok {
		return models.ChartDataset{}, fmt.Errorf("dataset missing 'label' field")
	}

	label, ok := labelInterface.(string)
	if !ok {
		label = fmt.Sprintf("%v", labelInterface)
	}

	// Extract data
	dataInterface, ok := datasetMap["data"]
	if !ok {
		return models.ChartDataset{}, fmt.Errorf("dataset missing 'data' field")
	}

	data, err := sp.extractDataValues(dataInterface)
	if err != nil {
		return models.ChartDataset{}, fmt.Errorf("failed to extract data values: %w", err)
	}

	return models.ChartDataset{
		Label: label,
		Data:  data,
	}, nil
}

// extractDataValues extracts numeric data values from various formats.
func (sp *StreamProcessor) extractDataValues(dataInterface interface{}) ([]float64, error) {
	// If already []float64
	if data, ok := dataInterface.([]float64); ok {
		return data, nil
	}

	// If []interface{}
	if dataSlice, ok := dataInterface.([]interface{}); ok {
		data := make([]float64, len(dataSlice))
		for i, value := range dataSlice {
			switch v := value.(type) {
			case float64:
				data[i] = v
			case float32:
				data[i] = float64(v)
			case int:
				data[i] = float64(v)
			case int64:
				data[i] = float64(v)
			case int32:
				data[i] = float64(v)
			default:
				return nil, fmt.Errorf("unsupported data value type: %T", value)
			}
		}
		return data, nil
	}

	return nil, fmt.Errorf("unsupported data type: %T", dataInterface)
}

// extractChartDataFromString attempts to extract chart data from a string.
// This is a simple implementation that looks for chart-like patterns.
func (sp *StreamProcessor) extractChartDataFromString(dataStr string) (*models.ChartData, error) {
	// Look for patterns like:
	// "labels: [A, B, C], data: [1, 2, 3]"
	// This is a simplified implementation

	// Extract labels
	labelsPattern := regexp.MustCompile(`labels:\s*\[([^\]]+)\]`)
	labelsMatch := labelsPattern.FindStringSubmatch(dataStr)
	if len(labelsMatch) < 2 {
		return nil, fmt.Errorf("no labels found in string")
	}

	labelsStr := labelsMatch[1]
	labels := strings.Split(labelsStr, ",")
	for i := range labels {
		labels[i] = strings.TrimSpace(labels[i])
	}

	// Extract data
	dataPattern := regexp.MustCompile(`data:\s*\[([^\]]+)\]`)
	dataMatch := dataPattern.FindStringSubmatch(dataStr)
	if len(dataMatch) < 2 {
		return nil, fmt.Errorf("no data found in string")
	}

	dataStr = dataMatch[1]
	dataStrs := strings.Split(dataStr, ",")
	data := make([]float64, 0, len(dataStrs))
	for _, dataStr := range dataStrs {
		dataStr = strings.TrimSpace(dataStr)
		var value float64
		_, err := fmt.Sscanf(dataStr, "%f", &value)
		if err != nil {
			sp.logger.Warn("Failed to parse data value",
				zap.String("value", dataStr),
				zap.Error(err))
			continue
		}
		data = append(data, value)
	}

	return &models.ChartData{
		Labels: labels,
		Datasets: []models.ChartDataset{
			{
				Label: "Data",
				Data:  data,
			},
		},
	}, nil
}

// StreamComplete sends the completion marker.
// Requirements: 6.4 - 发送完成标记
func (sp *StreamProcessor) StreamComplete(ctx context.Context, responseChan chan<- Response) error {
	select {
	case responseChan <- Response{
		Type:    ResponseTypeDone,
		Content: nil,
	}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// StreamError sends an error response.
// Requirements: 6.5 - 发送错误消息并终止流式响应
func (sp *StreamProcessor) StreamError(ctx context.Context, err error, responseChan chan<- Response) error {
	select {
	case responseChan <- Response{
		Type:    ResponseTypeError,
		Content: err.Error(),
	}:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// ProcessAndStream is a high-level method that processes content and streams it appropriately.
// It handles text, tool results, and chart data extraction automatically.
func (sp *StreamProcessor) ProcessAndStream(ctx context.Context, content interface{}, opts StreamOptions, responseChan chan<- Response) error {
	if opts.MessageID == "" {
		opts.MessageID = uuid.New().String()
	}

	// Handle different content types
	switch v := content.(type) {
	case string:
		// Plain text - stream in chunks
		return sp.StreamText(ctx, v, opts, responseChan)

	case map[string]interface{}:
		// Check if it contains tool result
		if toolResult, hasToolResult := v["toolResult"]; hasToolResult {
			text, _ := v["text"].(string)
			if toolResultMap, ok := toolResult.(map[string]interface{}); ok {
				return sp.StreamWithToolResult(ctx, text, toolResultMap, opts, responseChan)
			}
		}

		// Check if it contains chart data
		if opts.ExtractCharts {
			if _, hasLabels := v["labels"]; hasLabels {
				if _, hasDatasets := v["datasets"]; hasDatasets {
					return sp.ExtractAndStreamChartData(ctx, v, opts, responseChan)
				}
			}
		}

		// Generic map - convert to string and stream
		return sp.StreamText(ctx, fmt.Sprintf("%v", v), opts, responseChan)

	case models.ChartData:
		// Chart data - send as separate response
		return sp.ExtractAndStreamChartData(ctx, v, opts, responseChan)

	default:
		// Unknown type - convert to string and stream
		return sp.StreamText(ctx, fmt.Sprintf("%v", v), opts, responseChan)
	}
}
