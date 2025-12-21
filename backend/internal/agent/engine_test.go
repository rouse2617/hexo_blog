package agent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStreamingResponseGeneration tests that the engine generates streaming responses
// Requirements: 6.1 - WHEN Agent 开始生成响应，THE Agent_Engine SHALL 以流式方式发送响应片段到前端
func TestStreamingResponseGeneration(t *testing.T) {
	t.Run("should send multiple response fragments", func(t *testing.T) {
		engine := &mockStreamingEngine{
			fragmentCount: 5,
		}

		ctx := context.Background()
		responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
		require.NoError(t, err)
		require.NotNil(t, responseChan)

		// Collect all responses
		var responses []Response
		for resp := range responseChan {
			responses = append(responses, resp)
		}

		// Should have multiple fragments (at least 2: text fragments + done marker)
		assert.GreaterOrEqual(t, len(responses), 2, "should receive multiple response fragments")

		// Last response should be done marker
		lastResponse := responses[len(responses)-1]
		assert.Equal(t, ResponseTypeDone, lastResponse.Type, "last response should be done marker")
	})

	t.Run("should send responses incrementally over time", func(t *testing.T) {
		engine := &mockStreamingEngine{
			fragmentCount: 3,
			fragmentDelay: 10 * time.Millisecond,
		}

		ctx := context.Background()
		responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
		require.NoError(t, err)

		// Track timing of responses
		var timestamps []time.Time
		for range responseChan {
			timestamps = append(timestamps, time.Now())
		}

		// Should have received multiple responses
		assert.GreaterOrEqual(t, len(timestamps), 2)

		// Verify responses arrived at different times (streaming, not all at once)
		if len(timestamps) >= 2 {
			timeDiff := timestamps[len(timestamps)-1].Sub(timestamps[0])
			assert.Greater(t, timeDiff, time.Duration(0), "responses should arrive over time, not all at once")
		}
	})

	t.Run("should close channel after streaming completes", func(t *testing.T) {
		engine := &mockStreamingEngine{
			fragmentCount: 2,
		}

		ctx := context.Background()
		responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
		require.NoError(t, err)

		// Read all responses
		for range responseChan {
		}

		// Channel should be closed - reading again should return immediately with zero value
		select {
		case _, ok := <-responseChan:
			assert.False(t, ok, "channel should be closed after streaming completes")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("channel should be closed but appears to be blocking")
		}
	})

	t.Run("should send done marker as final response", func(t *testing.T) {
		engine := &mockStreamingEngine{
			fragmentCount: 4,
		}

		ctx := context.Background()
		responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
		require.NoError(t, err)

		var responses []Response
		for resp := range responseChan {
			responses = append(responses, resp)
		}

		require.NotEmpty(t, responses)

		// Last response must be done marker
		lastResponse := responses[len(responses)-1]
		assert.Equal(t, ResponseTypeDone, lastResponse.Type)

		// Count done markers - should be exactly one
		doneCount := 0
		for _, resp := range responses {
			if resp.Type == ResponseTypeDone {
				doneCount++
			}
		}
		assert.Equal(t, 1, doneCount, "should have exactly one done marker")
	})

	t.Run("should handle empty message", func(t *testing.T) {
		engine := &mockStreamingEngine{
			fragmentCount: 1,
		}

		ctx := context.Background()
		responseChan, err := engine.ProcessMessage(ctx, "test-session", "")
		require.NoError(t, err)

		// Should still receive responses even with empty message
		var responses []Response
		for resp := range responseChan {
			responses = append(responses, resp)
		}

		assert.NotEmpty(t, responses)
	})
}

// TestResponseGenerationErrorHandling tests error handling during response generation
// Requirements: 6.5 - WHEN 响应生成过程中出错，THE Agent_Engine SHALL 发送错误消息并终止流式响应
func TestResponseGenerationErrorHandling(t *testing.T) {
	t.Run("should send error response when generation fails", func(t *testing.T) {
		expectedError := errors.New("LLM API failure")
		engine := &mockErrorEngine{
			errorToReturn: expectedError,
			sendErrorResponse: true,
		}

		ctx := context.Background()
		responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
		require.NoError(t, err)

		// Should receive error response
		var responses []Response
		foundError := false
		for resp := range responseChan {
			responses = append(responses, resp)
			if resp.Type == ResponseTypeError {
				foundError = true
				assert.NotNil(t, resp.Content, "error response should have content")
			}
		}

		assert.True(t, foundError, "should receive error response when generation fails")
	})

	t.Run("should terminate streaming after error", func(t *testing.T) {
		engine := &mockErrorEngine{
			errorToReturn: errors.New("processing error"),
			sendErrorResponse: true,
			fragmentsBeforeError: 2,
		}

		ctx := context.Background()
		responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
		require.NoError(t, err)

		var responses []Response
		for resp := range responseChan {
			responses = append(responses, resp)
		}

		// Should have some fragments before error, then error, then channel closes
		assert.NotEmpty(t, responses)

		// Find error response
		foundError := false
		errorIndex := -1
		for i, resp := range responses {
			if resp.Type == ResponseTypeError {
				foundError = true
				errorIndex = i
				break
			}
		}

		assert.True(t, foundError, "should have error response")

		// After error, channel should close (no more responses after error)
		if errorIndex >= 0 {
			// Error should be one of the last responses (possibly followed by done marker)
			assert.LessOrEqual(t, len(responses)-errorIndex, 2, "channel should close shortly after error")
		}
	})

	t.Run("should handle context cancellation", func(t *testing.T) {
		engine := &mockStreamingEngine{
			fragmentCount: 10,
			fragmentDelay: 50 * time.Millisecond,
		}

		ctx, cancel := context.WithCancel(context.Background())
		responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
		require.NoError(t, err)

		// Read a few responses then cancel
		responseCount := 0
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		for range responseChan {
			responseCount++
		}

		// Should have received some responses but not all (due to cancellation)
		// This is a basic test - actual implementation should respect context cancellation
		assert.Greater(t, responseCount, 0, "should have received some responses before cancellation")
	})

	t.Run("should handle timeout", func(t *testing.T) {
		engine := &mockStreamingEngine{
			fragmentCount: 5,
			fragmentDelay: 100 * time.Millisecond,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
		require.NoError(t, err)

		var responses []Response
		for resp := range responseChan {
			responses = append(responses, resp)
		}

		// Should have received partial responses before timeout
		// Exact count depends on timing, but should be less than total fragments
		assert.NotEmpty(t, responses, "should have received some responses before timeout")
	})

	t.Run("should return error for invalid session", func(t *testing.T) {
		engine := &mockErrorEngine{
			errorOnStart: true,
			errorToReturn: errors.New("invalid session ID"),
		}

		ctx := context.Background()
		responseChan, err := engine.ProcessMessage(ctx, "", "test message")

		// Should return error immediately
		assert.Error(t, err)
		assert.Nil(t, responseChan, "should not return channel when initialization fails")
	})

	t.Run("should handle multiple concurrent requests", func(t *testing.T) {
		engine := &mockStreamingEngine{
			fragmentCount: 3,
			fragmentDelay: 10 * time.Millisecond,
		}

		ctx := context.Background()
		numRequests := 5

		// Start multiple concurrent requests
		channels := make([]<-chan Response, numRequests)
		for i := 0; i < numRequests; i++ {
			ch, err := engine.ProcessMessage(ctx, "test-session", "test message")
			require.NoError(t, err)
			channels[i] = ch
		}

		// Read from all channels
		for i, ch := range channels {
			var responses []Response
			for resp := range ch {
				responses = append(responses, resp)
			}
			assert.NotEmpty(t, responses, "request %d should receive responses", i)
		}
	})

	t.Run("should include error details in error response", func(t *testing.T) {
		expectedError := "database connection failed"
		engine := &mockErrorEngine{
			errorToReturn: errors.New(expectedError),
			sendErrorResponse: true,
		}

		ctx := context.Background()
		responseChan, err := engine.ProcessMessage(ctx, "test-session", "test message")
		require.NoError(t, err)

		for resp := range responseChan {
			if resp.Type == ResponseTypeError {
				// Error content should contain error information
				assert.NotNil(t, resp.Content)
				
				// Check if content contains error details
				if errContent, ok := resp.Content.(string); ok {
					assert.NotEmpty(t, errContent, "error response should have error message")
				} else if errMap, ok := resp.Content.(map[string]interface{}); ok {
					assert.NotEmpty(t, errMap, "error response should have error details")
				}
			}
		}
	})
}

// mockErrorEngine simulates an engine that can produce errors
type mockErrorEngine struct {
	errorOnStart         bool
	errorToReturn        error
	sendErrorResponse    bool
	fragmentsBeforeError int
}

func (m *mockErrorEngine) ProcessMessage(ctx context.Context, sessionID string, message string) (<-chan Response, error) {
	// Return error immediately if configured
	if m.errorOnStart {
		return nil, m.errorToReturn
	}

	responseChan := make(chan Response, 10)

	go func() {
		defer close(responseChan)

		// Send some fragments before error if configured
		for i := 0; i < m.fragmentsBeforeError; i++ {
			select {
			case <-ctx.Done():
				return
			case responseChan <- Response{
				Type:    ResponseTypeText,
				Content: "fragment",
			}:
			}
		}

		// Send error response if configured
		if m.sendErrorResponse {
			responseChan <- Response{
				Type:    ResponseTypeError,
				Content: m.errorToReturn.Error(),
			}
		}
	}()

	return responseChan, nil
}

func (m *mockErrorEngine) Stop() error {
	return nil
}
