// Package confirmation provides confirmation management functionality.
package confirmation

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfirmationTimeout tests that confirmation requests timeout correctly.
// Requirements: 7.2
func TestConfirmationTimeout(t *testing.T) {
	t.Run("request times out after specified duration", func(t *testing.T) {
		// Create manager with short timeout
		timeout := 100 * time.Millisecond
		manager := NewManager(timeout, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "restart",
			Target: "test-service",
			Params: map[string]interface{}{
				"force": true,
			},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)
		require.NotNil(t, request)

		// Wait for response (should timeout)
		response, err := manager.WaitForResponse(ctx, request.ID)
		require.NoError(t, err)
		require.NotNil(t, response)

		// Verify response is timeout
		assert.Equal(t, ActionTimeout, response.Action)
		assert.Equal(t, request.ID, response.RequestID)
		assert.False(t, response.Timestamp.IsZero())
	})

	t.Run("timeout does not occur if confirmed before timeout", func(t *testing.T) {
		// Create manager with longer timeout
		timeout := 500 * time.Millisecond
		manager := NewManager(timeout, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "delete",
			Target: "test-resource",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)

		// Start goroutine to wait for response
		responseChan := make(chan *Response, 1)
		errorChan := make(chan error, 1)

		go func() {
			resp, err := manager.WaitForResponse(ctx, request.ID)
			if err != nil {
				errorChan <- err
				return
			}
			responseChan <- resp
		}()

		// Confirm before timeout
		time.Sleep(50 * time.Millisecond)
		err = manager.Confirm(request.ID)
		require.NoError(t, err)

		// Wait for response
		select {
		case response := <-responseChan:
			// Should get confirmed, not timeout
			assert.Equal(t, ActionConfirmed, response.Action)
			assert.Equal(t, request.ID, response.RequestID)
		case err := <-errorChan:
			t.Fatalf("unexpected error: %v", err)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for response")
		}
	})

	t.Run("timeout does not occur if cancelled before timeout", func(t *testing.T) {
		// Create manager with longer timeout
		timeout := 500 * time.Millisecond
		manager := NewManager(timeout, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "modify_config",
			Target: "test-config",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)

		// Start goroutine to wait for response
		responseChan := make(chan *Response, 1)
		errorChan := make(chan error, 1)

		go func() {
			resp, err := manager.WaitForResponse(ctx, request.ID)
			if err != nil {
				errorChan <- err
				return
			}
			responseChan <- resp
		}()

		// Cancel before timeout
		time.Sleep(50 * time.Millisecond)
		err = manager.Cancel(request.ID)
		require.NoError(t, err)

		// Wait for response
		select {
		case response := <-responseChan:
			// Should get cancelled, not timeout
			assert.Equal(t, ActionCancelled, response.Action)
			assert.Equal(t, request.ID, response.RequestID)
		case err := <-errorChan:
			t.Fatalf("unexpected error: %v", err)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for response")
		}
	})

	t.Run("multiple requests timeout independently", func(t *testing.T) {
		// Create manager with short timeout
		timeout := 100 * time.Millisecond
		manager := NewManager(timeout, func(req *Request) error {
			return nil
		})

		// Create multiple requests
		ctx := context.Background()
		request1, err := manager.CreateRequest(ctx, Operation{
			Type:   "restart",
			Target: "service-1",
			Params: map[string]interface{}{},
		})
		require.NoError(t, err)

		request2, err := manager.CreateRequest(ctx, Operation{
			Type:   "restart",
			Target: "service-2",
			Params: map[string]interface{}{},
		})
		require.NoError(t, err)

		// Start goroutines to wait for both responses
		responseChan1 := make(chan *Response, 1)
		responseChan2 := make(chan *Response, 1)
		errorChan := make(chan error, 2)

		go func() {
			resp, err := manager.WaitForResponse(ctx, request1.ID)
			if err != nil {
				errorChan <- err
				return
			}
			responseChan1 <- resp
		}()

		go func() {
			resp, err := manager.WaitForResponse(ctx, request2.ID)
			if err != nil {
				errorChan <- err
				return
			}
			responseChan2 <- resp
		}()

		// Wait for both responses
		var response1, response2 *Response
		for i := 0; i < 2; i++ {
			select {
			case resp := <-responseChan1:
				response1 = resp
			case resp := <-responseChan2:
				response2 = resp
			case err := <-errorChan:
				t.Fatalf("unexpected error: %v", err)
			case <-time.After(500 * time.Millisecond):
				t.Fatal("timeout waiting for responses")
			}
		}

		// Verify both timed out
		require.NotNil(t, response1)
		assert.Equal(t, ActionTimeout, response1.Action)
		assert.Equal(t, request1.ID, response1.RequestID)

		require.NotNil(t, response2)
		assert.Equal(t, ActionTimeout, response2.Action)
		assert.Equal(t, request2.ID, response2.RequestID)
	})
}

// TestConfirmationResponse tests confirmation and cancellation responses.
// Requirements: 7.5
func TestConfirmationResponse(t *testing.T) {
	t.Run("confirm returns confirmed action", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "restart",
			Target: "test-service",
			Params: map[string]interface{}{
				"graceful": true,
			},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)

		// Start goroutine to wait for response
		responseChan := make(chan *Response, 1)
		errorChan := make(chan error, 1)

		go func() {
			resp, err := manager.WaitForResponse(ctx, request.ID)
			if err != nil {
				errorChan <- err
				return
			}
			responseChan <- resp
		}()

		// Give goroutine time to start waiting
		time.Sleep(10 * time.Millisecond)

		// Confirm the request
		err = manager.Confirm(request.ID)
		require.NoError(t, err)

		// Wait for response
		select {
		case response := <-responseChan:
			assert.Equal(t, ActionConfirmed, response.Action)
			assert.Equal(t, request.ID, response.RequestID)
			assert.False(t, response.Timestamp.IsZero())
			assert.WithinDuration(t, time.Now(), response.Timestamp, 1*time.Second)
		case err := <-errorChan:
			t.Fatalf("unexpected error: %v", err)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for response")
		}
	})

	t.Run("cancel returns cancelled action", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "delete",
			Target: "test-resource",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)

		// Start goroutine to wait for response
		responseChan := make(chan *Response, 1)
		errorChan := make(chan error, 1)

		go func() {
			resp, err := manager.WaitForResponse(ctx, request.ID)
			if err != nil {
				errorChan <- err
				return
			}
			responseChan <- resp
		}()

		// Give goroutine time to start waiting
		time.Sleep(10 * time.Millisecond)

		// Cancel the request
		err = manager.Cancel(request.ID)
		require.NoError(t, err)

		// Wait for response
		select {
		case response := <-responseChan:
			assert.Equal(t, ActionCancelled, response.Action)
			assert.Equal(t, request.ID, response.RequestID)
			assert.False(t, response.Timestamp.IsZero())
			assert.WithinDuration(t, time.Now(), response.Timestamp, 1*time.Second)
		case err := <-errorChan:
			t.Fatalf("unexpected error: %v", err)
		case <-time.After(1 * time.Second):
			t.Fatal("timeout waiting for response")
		}
	})

	t.Run("confirm on non-existent request returns error", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Try to confirm non-existent request
		err := manager.Confirm("non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request not found")
	})

	t.Run("cancel on non-existent request returns error", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Try to cancel non-existent request
		err := manager.Cancel("non-existent-id")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request not found")
	})

	t.Run("cannot confirm request twice", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "restart",
			Target: "test-service",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)

		// Confirm the request
		err = manager.Confirm(request.ID)
		require.NoError(t, err)

		// Try to confirm again
		err = manager.Confirm(request.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request not found")
	})

	t.Run("cannot cancel request twice", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "delete",
			Target: "test-resource",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)

		// Cancel the request
		err = manager.Cancel(request.ID)
		require.NoError(t, err)

		// Try to cancel again
		err = manager.Cancel(request.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request not found")
	})

	t.Run("cannot confirm after cancel", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "modify_config",
			Target: "test-config",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)

		// Cancel the request
		err = manager.Cancel(request.ID)
		require.NoError(t, err)

		// Try to confirm
		err = manager.Confirm(request.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request not found")
	})

	t.Run("cannot cancel after confirm", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "update",
			Target: "test-resource",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)

		// Confirm the request
		err = manager.Confirm(request.ID)
		require.NoError(t, err)

		// Try to cancel
		err = manager.Cancel(request.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "request not found")
	})

	t.Run("multiple requests can be confirmed independently", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create multiple requests
		ctx := context.Background()
		request1, err := manager.CreateRequest(ctx, Operation{
			Type:   "restart",
			Target: "service-1",
			Params: map[string]interface{}{},
		})
		require.NoError(t, err)

		request2, err := manager.CreateRequest(ctx, Operation{
			Type:   "restart",
			Target: "service-2",
			Params: map[string]interface{}{},
		})
		require.NoError(t, err)

		// Confirm first request
		err = manager.Confirm(request1.ID)
		require.NoError(t, err)

		// Cancel second request
		err = manager.Cancel(request2.ID)
		require.NoError(t, err)

		// Verify first request cannot be confirmed again
		err = manager.Confirm(request1.ID)
		assert.Error(t, err)

		// Verify second request cannot be cancelled again
		err = manager.Cancel(request2.ID)
		assert.Error(t, err)
	})
}

// TestCreateRequest tests request creation edge cases.
func TestCreateRequest(t *testing.T) {
	t.Run("create request with valid operation", func(t *testing.T) {
		// Create manager
		requestSent := false
		manager := NewManager(30*time.Second, func(req *Request) error {
			requestSent = true
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "restart",
			Target: "test-service",
			Params: map[string]interface{}{
				"graceful": true,
			},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)
		require.NotNil(t, request)

		// Verify request fields
		assert.NotEmpty(t, request.ID)
		assert.Equal(t, "restart", request.Operation.Type)
		assert.Equal(t, "test-service", request.Operation.Target)
		assert.NotEmpty(t, request.Description)
		assert.Equal(t, RiskLevelMedium, request.RiskLevel)
		assert.Equal(t, 30*time.Second, request.Timeout)
		assert.False(t, request.CreatedAt.IsZero())

		// Verify request was sent
		assert.True(t, requestSent)
	})

	t.Run("create request without operation type returns error", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create operation without type
		operation := Operation{
			Type:   "",
			Target: "test-service",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Contains(t, err.Error(), "operation type is required")
	})

	t.Run("create request without operation target returns error", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create operation without target
		operation := Operation{
			Type:   "restart",
			Target: "",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Contains(t, err.Error(), "operation target is required")
	})

	t.Run("create request with send failure returns error", func(t *testing.T) {
		// Create manager with failing send function
		sendError := errors.New("send failed")
		manager := NewManager(30*time.Second, func(req *Request) error {
			return sendError
		})

		// Create operation
		operation := Operation{
			Type:   "restart",
			Target: "test-service",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		assert.Error(t, err)
		assert.Nil(t, request)
		assert.Contains(t, err.Error(), "failed to send request")
	})

	t.Run("create request with nil send function succeeds", func(t *testing.T) {
		// Create manager without send function
		manager := NewManager(30*time.Second, nil)

		// Create operation
		operation := Operation{
			Type:   "restart",
			Target: "test-service",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)
		require.NotNil(t, request)
	})
}

// TestWaitForResponse tests waiting for responses.
func TestWaitForResponse(t *testing.T) {
	t.Run("wait for response on non-existent request returns error", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Try to wait for non-existent request
		ctx := context.Background()
		response, err := manager.WaitForResponse(ctx, "non-existent-id")
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Contains(t, err.Error(), "request not found")
	})

	t.Run("wait for response respects context cancellation", func(t *testing.T) {
		// Create manager
		manager := NewManager(30*time.Second, func(req *Request) error {
			return nil
		})

		// Create operation
		operation := Operation{
			Type:   "restart",
			Target: "test-service",
			Params: map[string]interface{}{},
		}

		// Create request
		ctx := context.Background()
		request, err := manager.CreateRequest(ctx, operation)
		require.NoError(t, err)

		// Create context with timeout
		waitCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()

		// Wait for response (should be cancelled by context)
		response, err := manager.WaitForResponse(waitCtx, request.ID)
		assert.Error(t, err)
		assert.Nil(t, response)
		assert.Equal(t, context.DeadlineExceeded, err)
	})
}

// TestRiskLevelDetermination tests risk level determination.
func TestRiskLevelDetermination(t *testing.T) {
	tests := []struct {
		name          string
		operationType string
		expectedRisk  RiskLevel
	}{
		{"delete is high risk", "delete", RiskLevelHigh},
		{"destroy is high risk", "destroy", RiskLevelHigh},
		{"terminate is high risk", "terminate", RiskLevelHigh},
		{"restart is medium risk", "restart", RiskLevelMedium},
		{"modify_config is medium risk", "modify_config", RiskLevelMedium},
		{"update is medium risk", "update", RiskLevelMedium},
		{"query is low risk", "query", RiskLevelLow},
		{"read is low risk", "read", RiskLevelLow},
		{"unknown is low risk", "unknown", RiskLevelLow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			riskLevel := determineRiskLevel(tt.operationType)
			assert.Equal(t, tt.expectedRisk, riskLevel)
		})
	}
}

// TestDefaultTimeout tests default timeout behavior.
func TestDefaultTimeout(t *testing.T) {
	t.Run("uses provided timeout", func(t *testing.T) {
		timeout := 45 * time.Second
		manager := NewManager(timeout, nil)
		assert.NotNil(t, manager)
		assert.Equal(t, timeout, manager.defaultTimeout)
	})

	t.Run("uses default timeout when zero provided", func(t *testing.T) {
		manager := NewManager(0, nil)
		assert.NotNil(t, manager)
		assert.Equal(t, 30*time.Second, manager.defaultTimeout)
	})
}
