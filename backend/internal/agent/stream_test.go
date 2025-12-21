package agent

import (
	"context"
	"testing"
	"time"

	"github.com/opsgenius/backend/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestStreamProcessor_StreamText(t *testing.T) {
	t.Run("should stream text in chunks", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		text := "This is a test message that should be chunked"
		opts := StreamOptions{
			SessionID: "test-session",
			MessageID: "test-message",
			ChunkSize: 10,
		}

		go func() {
			err := sp.StreamText(ctx, text, opts, responseChan)
			require.NoError(t, err)
			close(responseChan)
		}()

		// Collect all chunks
		var chunks []string
		for resp := range responseChan {
			assert.Equal(t, ResponseTypeText, resp.Type)
			if content, ok := resp.Content.(string); ok {
				chunks = append(chunks, content)
			}
		}

		// Should have multiple chunks
		assert.Greater(t, len(chunks), 1, "text should be split into multiple chunks")

		// Reassemble chunks should equal original text
		reassembled := ""
		for _, chunk := range chunks {
			reassembled += chunk
		}
		assert.Equal(t, text, reassembled)
	})

	t.Run("should handle empty text", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		opts := StreamOptions{
			SessionID: "test-session",
			ChunkSize: 10,
		}

		err := sp.StreamText(ctx, "", opts, responseChan)
		require.NoError(t, err)
		close(responseChan)

		// Should not send any chunks for empty text
		chunks := 0
		for range responseChan {
			chunks++
		}
		assert.Equal(t, 0, chunks)
	})

	t.Run("should respect context cancellation", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx, cancel := context.WithCancel(context.Background())

		text := "This is a very long text that should be interrupted by context cancellation"
		opts := StreamOptions{
			SessionID: "test-session",
			ChunkSize: 5,
		}

		// Cancel context after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		err := sp.StreamText(ctx, text, opts, responseChan)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})

	t.Run("should use default chunk size when not specified", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		text := "This is a test message with default chunk size"
		opts := StreamOptions{
			SessionID: "test-session",
			ChunkSize: 0, // Use default
		}

		go func() {
			err := sp.StreamText(ctx, text, opts, responseChan)
			require.NoError(t, err)
			close(responseChan)
		}()

		// Should still stream in chunks
		chunks := 0
		for range responseChan {
			chunks++
		}
		assert.Greater(t, chunks, 0)
	})
}

func TestStreamProcessor_StreamWithToolResult(t *testing.T) {
	t.Run("should embed tool result in response", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		text := "Tool execution completed"
		toolResult := map[string]interface{}{
			"tool":   "test-tool",
			"status": "success",
			"data":   "result data",
		}

		opts := StreamOptions{
			SessionID: "test-session",
			MessageID: "test-message",
		}

		err := sp.StreamWithToolResult(ctx, text, toolResult, opts, responseChan)
		require.NoError(t, err)
		close(responseChan)

		// Should receive one response with embedded tool result
		responses := 0
		for resp := range responseChan {
			responses++
			assert.Equal(t, ResponseTypeText, resp.Type)

			if content, ok := resp.Content.(map[string]interface{}); ok {
				assert.Equal(t, text, content["text"])
				assert.Equal(t, toolResult, content["toolResult"])
			} else {
				t.Fatal("content should be a map")
			}
		}

		assert.Equal(t, 1, responses)
	})

	t.Run("should generate message ID if not provided", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		opts := StreamOptions{
			SessionID: "test-session",
			// MessageID not provided
		}

		err := sp.StreamWithToolResult(ctx, "test", map[string]interface{}{}, opts, responseChan)
		require.NoError(t, err)
		// Should not error even without message ID
	})
}

func TestStreamProcessor_ExtractAndStreamChartData(t *testing.T) {
	t.Run("should extract and stream chart data from ChartData struct", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		chartData := models.ChartData{
			Labels: []string{"A", "B", "C"},
			Datasets: []models.ChartDataset{
				{
					Label: "Dataset 1",
					Data:  []float64{1.0, 2.0, 3.0},
				},
			},
		}

		opts := StreamOptions{
			SessionID: "test-session",
			MessageID: "test-message",
		}

		err := sp.ExtractAndStreamChartData(ctx, chartData, opts, responseChan)
		require.NoError(t, err)
		close(responseChan)

		// Should receive chart response
		responses := 0
		for resp := range responseChan {
			responses++
			assert.Equal(t, ResponseTypeChart, resp.Type)

			if chart, ok := resp.Content.(models.ChartData); ok {
				assert.Equal(t, chartData.Labels, chart.Labels)
				assert.Equal(t, len(chartData.Datasets), len(chart.Datasets))
			} else {
				t.Fatal("content should be ChartData")
			}
		}

		assert.Equal(t, 1, responses)
	})

	t.Run("should extract chart data from map", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		dataMap := map[string]interface{}{
			"labels": []string{"X", "Y", "Z"},
			"datasets": []interface{}{
				map[string]interface{}{
					"label": "Test Dataset",
					"data":  []interface{}{10.0, 20.0, 30.0},
				},
			},
		}

		opts := StreamOptions{
			SessionID: "test-session",
		}

		err := sp.ExtractAndStreamChartData(ctx, dataMap, opts, responseChan)
		require.NoError(t, err)
		close(responseChan)

		// Should successfully extract and send chart data
		responses := 0
		for resp := range responseChan {
			responses++
			assert.Equal(t, ResponseTypeChart, resp.Type)
		}

		assert.Equal(t, 1, responses)
	})

	t.Run("should return error for invalid data", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		opts := StreamOptions{
			SessionID: "test-session",
		}

		// Invalid data - just a string without chart structure
		err := sp.ExtractAndStreamChartData(ctx, "not chart data", opts, responseChan)
		assert.Error(t, err)
	})
}

func TestStreamProcessor_StreamComplete(t *testing.T) {
	t.Run("should send done marker", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		err := sp.StreamComplete(ctx, responseChan)
		require.NoError(t, err)
		close(responseChan)

		// Should receive done marker
		responses := 0
		for resp := range responseChan {
			responses++
			assert.Equal(t, ResponseTypeDone, resp.Type)
			assert.Nil(t, resp.Content)
		}

		assert.Equal(t, 1, responses)
	})
}

func TestStreamProcessor_StreamError(t *testing.T) {
	t.Run("should send error response", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		testError := assert.AnError
		err := sp.StreamError(ctx, testError, responseChan)
		require.NoError(t, err)
		close(responseChan)

		// Should receive error response
		responses := 0
		for resp := range responseChan {
			responses++
			assert.Equal(t, ResponseTypeError, resp.Type)
			assert.NotNil(t, resp.Content)
			if errMsg, ok := resp.Content.(string); ok {
				assert.Contains(t, errMsg, testError.Error())
			}
		}

		assert.Equal(t, 1, responses)
	})
}

func TestStreamProcessor_ProcessAndStream(t *testing.T) {
	t.Run("should handle string content", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		content := "Test message"
		opts := StreamOptions{
			SessionID: "test-session",
			ChunkSize: 5,
		}

		go func() {
			err := sp.ProcessAndStream(ctx, content, opts, responseChan)
			require.NoError(t, err)
			close(responseChan)
		}()

		// Should stream text
		responses := 0
		for resp := range responseChan {
			responses++
			assert.Equal(t, ResponseTypeText, resp.Type)
		}

		assert.Greater(t, responses, 0)
	})

	t.Run("should handle map with tool result", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		content := map[string]interface{}{
			"text": "Tool result",
			"toolResult": map[string]interface{}{
				"tool": "test",
				"data": "result",
			},
		}

		opts := StreamOptions{
			SessionID: "test-session",
		}

		err := sp.ProcessAndStream(ctx, content, opts, responseChan)
		require.NoError(t, err)
		close(responseChan)

		// Should send response with embedded tool result
		responses := 0
		for resp := range responseChan {
			responses++
			assert.Equal(t, ResponseTypeText, resp.Type)
			if contentMap, ok := resp.Content.(map[string]interface{}); ok {
				assert.Contains(t, contentMap, "toolResult")
			}
		}

		assert.Equal(t, 1, responses)
	})

	t.Run("should handle ChartData", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		content := models.ChartData{
			Labels: []string{"A", "B"},
			Datasets: []models.ChartDataset{
				{Label: "Test", Data: []float64{1, 2}},
			},
		}

		opts := StreamOptions{
			SessionID: "test-session",
		}

		err := sp.ProcessAndStream(ctx, content, opts, responseChan)
		require.NoError(t, err)
		close(responseChan)

		// Should send chart response
		responses := 0
		for resp := range responseChan {
			responses++
			assert.Equal(t, ResponseTypeChart, resp.Type)
		}

		assert.Equal(t, 1, responses)
	})

	t.Run("should extract chart from map when enabled", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())
		responseChan := make(chan Response, 10)
		ctx := context.Background()

		content := map[string]interface{}{
			"labels": []string{"X", "Y"},
			"datasets": []interface{}{
				map[string]interface{}{
					"label": "Data",
					"data":  []interface{}{1.0, 2.0},
				},
			},
		}

		opts := StreamOptions{
			SessionID:     "test-session",
			ExtractCharts: true,
		}

		err := sp.ProcessAndStream(ctx, content, opts, responseChan)
		require.NoError(t, err)
		close(responseChan)

		// Should send chart response
		responses := 0
		for resp := range responseChan {
			responses++
			assert.Equal(t, ResponseTypeChart, resp.Type)
		}

		assert.Equal(t, 1, responses)
	})
}

func TestStreamProcessor_ExtractChartDataFromString(t *testing.T) {
	t.Run("should extract chart data from formatted string", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())

		dataStr := "labels: [A, B, C], data: [1.0, 2.0, 3.0]"
		chartData, err := sp.extractChartDataFromString(dataStr)

		require.NoError(t, err)
		require.NotNil(t, chartData)

		assert.Equal(t, []string{"A", "B", "C"}, chartData.Labels)
		assert.Len(t, chartData.Datasets, 1)
		assert.Equal(t, []float64{1.0, 2.0, 3.0}, chartData.Datasets[0].Data)
	})

	t.Run("should return error for invalid string format", func(t *testing.T) {
		sp := NewStreamProcessor(zap.NewNop())

		dataStr := "this is not chart data"
		_, err := sp.extractChartDataFromString(dataStr)

		assert.Error(t, err)
	})
}
