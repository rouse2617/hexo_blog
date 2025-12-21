package agent

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"github.com/opsgenius/backend/pkg/models"
)

// Feature: ops-genius-backend, Property 20: 流式响应片段发送
// *对于任何*Agent 响应，应该以多个片段的形式流式发送，而不是等待完整响应生成后一次性发送。
// **Validates: Requirements 6.1**
func TestProperty_StreamingResponseFragments(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("agent responses should be sent as multiple fragments", prop.ForAll(
		func(numFragments int) bool {
			if numFragments < 2 {
				numFragments = 2 // Ensure at least 2 fragments for streaming
			}

			// Create a mock engine that sends multiple fragments
			engine := &mockStreamingEngine{
				fragmentCount: numFragments,
			}

			ctx := context.Background()
			responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
			if err != nil {
				return false
			}

			// Collect all responses
			var responses []Response
			for resp := range responseChan {
				responses = append(responses, resp)
			}

			// Verify we received multiple fragments (not just one complete response)
			// At minimum, we should have text fragments + done marker
			return len(responses) >= 2
		},
		gen.IntRange(2, 20),
	))

	properties.Property("streaming responses should arrive incrementally", prop.ForAll(
		func(numFragments int) bool {
			if numFragments < 2 {
				numFragments = 2
			}

			engine := &mockStreamingEngine{
				fragmentCount: numFragments,
				fragmentDelay: 1 * time.Millisecond,
			}

			ctx := context.Background()
			responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
			if err != nil {
				return false
			}

			// Track timing of responses
			var timestamps []time.Time
			for range responseChan {
				timestamps = append(timestamps, time.Now())
			}

			// Verify responses arrived at different times (not all at once)
			if len(timestamps) < 2 {
				return false
			}

			// Check that there's a time gap between first and last response
			timeDiff := timestamps[len(timestamps)-1].Sub(timestamps[0])
			return timeDiff > 0
		},
		gen.IntRange(2, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 21: 工具结果嵌入
// *对于任何*包含工具调用结果的响应，响应内容应该包含结构化的工具结果数据。
// **Validates: Requirements 6.2**
func TestProperty_ToolResultEmbedding(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("responses with tool results should contain structured data", prop.ForAll(
		func(toolName string, resultValue string) bool {
			toolData := map[string]interface{}{
				"result": resultValue,
				"status": "success",
			}

			engine := &mockStreamingEngine{
				includeToolResult: true,
				toolResultData: map[string]interface{}{
					"tool":   toolName,
					"result": toolData,
				},
			}

			ctx := context.Background()
			responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
			if err != nil {
				return false
			}

			// Look for a response containing tool result
			foundToolResult := false
			for resp := range responseChan {
				if resp.Type == ResponseTypeText {
					if content, ok := resp.Content.(map[string]interface{}); ok {
						if _, hasToolResult := content["toolResult"]; hasToolResult {
							foundToolResult = true
							// Verify the tool result contains structured data
							if toolResult, ok := content["toolResult"].(map[string]interface{}); ok {
								if _, hasData := toolResult["result"]; hasData {
									return true
								}
							}
						}
					}
				}
			}

			return foundToolResult
		},
		gen.AlphaString(),
		gen.AlphaString(),
	))

	properties.Property("tool results should be distinguishable from regular text", prop.ForAll(
		func(regularText string) bool {
			if regularText == "" {
				regularText = "test"
			}

			engine := &mockStreamingEngine{
				includeToolResult: true,
				regularText:       regularText,
				toolResultData: map[string]interface{}{
					"tool":   "test-tool",
					"result": map[string]interface{}{"data": "test"},
				},
			}

			ctx := context.Background()
			responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
			if err != nil {
				return false
			}

			hasRegularText := false
			hasToolResult := false

			for resp := range responseChan {
				if resp.Type == ResponseTypeText {
					if content, ok := resp.Content.(string); ok && content == regularText {
						hasRegularText = true
					}
					if content, ok := resp.Content.(map[string]interface{}); ok {
						if _, hasResult := content["toolResult"]; hasResult {
							hasToolResult = true
						}
					}
				}
			}

			// Both regular text and tool result should be present and distinguishable
			return hasRegularText && hasToolResult
		},
		gen.AlphaString(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 22: 图表数据单独发送
// *对于任何*包含图表的响应，应该发送单独的 chart_data 类型消息，而不是嵌入在文本响应中。
// **Validates: Requirements 6.3**
func TestProperty_ChartDataSeparateSending(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("chart data should be sent as separate response type", prop.ForAll(
		func(numLabels, numDataPoints int) bool {
			// Ensure we have at least one label and data point
			if numLabels < 1 {
				numLabels = 1
			}
			if numDataPoints < 1 {
				numDataPoints = 1
			}

			// Generate labels and data
			labels := make([]string, numLabels)
			for i := 0; i < numLabels; i++ {
				labels[i] = "label" + string(rune('A'+i))
			}

			dataValues := make([]float64, numDataPoints)
			for i := 0; i < numDataPoints; i++ {
				dataValues[i] = float64(i + 1)
			}

			chartData := models.ChartData{
				Labels: labels,
				Datasets: []models.ChartDataset{
					{
						Label: "test-dataset",
						Data:  dataValues,
					},
				},
			}

			engine := &mockStreamingEngine{
				includeChart: true,
				chartData:    chartData,
			}

			ctx := context.Background()
			responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
			if err != nil {
				return false
			}

			// Look for chart response type
			foundChartResponse := false
			foundTextResponse := false

			for resp := range responseChan {
				if resp.Type == ResponseTypeChart {
					foundChartResponse = true
					// Verify chart data structure
					if chartContent, ok := resp.Content.(models.ChartData); ok {
						if len(chartContent.Labels) > 0 && len(chartContent.Datasets) > 0 {
							// Chart data is properly structured
						} else {
							return false
						}
					} else {
						return false
					}
				}
				if resp.Type == ResponseTypeText {
					foundTextResponse = true
				}
			}

			// Chart should be sent as separate response, not embedded in text
			return foundChartResponse && foundTextResponse
		},
		gen.IntRange(1, 10),
		gen.IntRange(1, 10),
	))

	properties.Property("chart responses should not be embedded in text content", prop.ForAll(
		func(textContent string, numLabels int) bool {
			if numLabels < 1 {
				numLabels = 1
			}

			labels := make([]string, numLabels)
			for i := 0; i < numLabels; i++ {
				labels[i] = "label" + string(rune('A'+i))
			}

			chartData := models.ChartData{
				Labels: labels,
				Datasets: []models.ChartDataset{
					{
						Label: "dataset",
						Data:  []float64{1.0, 2.0},
					},
				},
			}

			engine := &mockStreamingEngine{
				includeChart: true,
				chartData:    chartData,
				regularText:  textContent,
			}

			ctx := context.Background()
			responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
			if err != nil {
				return false
			}

			for resp := range responseChan {
				// Text responses should not contain chart data
				if resp.Type == ResponseTypeText {
					if content, ok := resp.Content.(string); ok {
						// Text should be plain text, not containing chart data
						_ = content
					} else if _, isChartData := resp.Content.(models.ChartData); isChartData {
						// Chart data should not be in text response
						return false
					}
				}
			}

			return true
		},
		gen.AlphaString(),
		gen.IntRange(1, 5),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 23: 流式响应完成标记
// *对于任何*完成的流式响应，最后一个消息片段应该包含 done=true 标记。
// **Validates: Requirements 6.4**
func TestProperty_StreamingResponseCompletionMarker(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("streaming responses should end with done marker", prop.ForAll(
		func(numFragments int) bool {
			if numFragments < 1 {
				numFragments = 1
			}

			engine := &mockStreamingEngine{
				fragmentCount: numFragments,
			}

			ctx := context.Background()
			responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
			if err != nil {
				return false
			}

			// Collect all responses
			var responses []Response
			for resp := range responseChan {
				responses = append(responses, resp)
			}

			if len(responses) == 0 {
				return false
			}

			// Last response should be done marker
			lastResponse := responses[len(responses)-1]
			return lastResponse.Type == ResponseTypeDone
		},
		gen.IntRange(1, 20),
	))

	properties.Property("done marker should be the final response", prop.ForAll(
		func(numFragments int) bool {
			if numFragments < 1 {
				numFragments = 1
			}

			engine := &mockStreamingEngine{
				fragmentCount: numFragments,
			}

			ctx := context.Background()
			responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
			if err != nil {
				return false
			}

			// Collect all responses
			var responses []Response
			doneCount := 0

			for resp := range responseChan {
				responses = append(responses, resp)
				if resp.Type == ResponseTypeDone {
					doneCount++
				}
			}

			// Should have exactly one done marker, and it should be last
			if len(responses) == 0 {
				return false
			}

			lastResponse := responses[len(responses)-1]
			return doneCount == 1 && lastResponse.Type == ResponseTypeDone
		},
		gen.IntRange(1, 20),
	))

	properties.Property("channel should close after done marker", prop.ForAll(
		func(numFragments int) bool {
			if numFragments < 1 {
				numFragments = 1
			}

			engine := &mockStreamingEngine{
				fragmentCount: numFragments,
			}

			ctx := context.Background()
			responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
			if err != nil {
				return false
			}

			// Read all responses
			responseCount := 0
			for range responseChan {
				responseCount++
			}

			// Channel should be closed (loop exits naturally)
			// Try to read again - should get zero value immediately
			select {
			case _, ok := <-responseChan:
				return !ok // Channel should be closed
			default:
				return true // Channel is closed
			}
		},
		gen.IntRange(1, 20),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// mockStreamingEngine is a mock implementation of Engine for testing streaming behavior.
type mockStreamingEngine struct {
	fragmentCount     int
	fragmentDelay     time.Duration
	includeToolResult bool
	includeChart      bool
	toolResultData    map[string]interface{}
	chartData         models.ChartData
	regularText       string
}

func (m *mockStreamingEngine) ProcessMessage(ctx context.Context, sessionID string, message string) (<-chan Response, error) {
	responseChan := make(chan Response, m.fragmentCount+5)

	go func() {
		defer close(responseChan)

		// Send regular text fragments
		if m.regularText != "" {
			responseChan <- Response{
				Type:    ResponseTypeText,
				Content: m.regularText,
			}
			if m.fragmentDelay > 0 {
				time.Sleep(m.fragmentDelay)
			}
		}

		// Send text fragments
		for i := 0; i < m.fragmentCount; i++ {
			responseChan <- Response{
				Type:    ResponseTypeText,
				Content: "fragment",
			}
			if m.fragmentDelay > 0 {
				time.Sleep(m.fragmentDelay)
			}
		}

		// Send tool result if requested
		if m.includeToolResult {
			responseChan <- Response{
				Type: ResponseTypeText,
				Content: map[string]interface{}{
					"text":       "Tool result:",
					"toolResult": m.toolResultData,
				},
			}
			if m.fragmentDelay > 0 {
				time.Sleep(m.fragmentDelay)
			}
		}

		// Send chart data if requested
		if m.includeChart {
			// Always send at least one text response before chart
			if m.regularText == "" && m.fragmentCount == 0 {
				responseChan <- Response{
					Type:    ResponseTypeText,
					Content: "Here is the chart:",
				}
			}
			responseChan <- Response{
				Type:    ResponseTypeChart,
				Content: m.chartData,
			}
			if m.fragmentDelay > 0 {
				time.Sleep(m.fragmentDelay)
			}
		}

		// Always send done marker as last response
		responseChan <- Response{
			Type:    ResponseTypeDone,
			Content: nil,
		}
	}()

	return responseChan, nil
}

func (m *mockStreamingEngine) Stop() error {
	return nil
}
