// Package confirmation provides confirmation management functionality.
package confirmation

import (
	"context"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: ops-genius-backend, Property 24: 确认请求创建和发送
// Validates: Requirements 7.1
func TestProperty_ConfirmationRequestCreationAndSending(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any high-risk operation, a confirmation request should be created and sent", prop.ForAll(
		func(operationType string, target string) bool {
			// Skip empty values
			if operationType == "" || target == "" {
				return true
			}

			requestSent := false
			var sentRequest *Request

			// Create manager with send function that captures the request
			manager := NewManager(30*time.Second, func(req *Request) error {
				requestSent = true
				sentRequest = req
				return nil
			})

			// Create operation
			operation := Operation{
				Type:   operationType,
				Target: target,
				Params: map[string]interface{}{
					"test": "value",
				},
			}

			// Create request
			ctx := context.Background()
			request, err := manager.CreateRequest(ctx, operation)
			if err != nil {
				return false
			}

			// Verify request was created
			if request == nil {
				return false
			}

			// Verify request has required fields
			if request.ID == "" {
				return false
			}

			if request.Operation.Type != operationType {
				return false
			}

			if request.Operation.Target != target {
				return false
			}

			if request.Description == "" {
				return false
			}

			if request.RiskLevel == "" {
				return false
			}

			if request.Timeout == 0 {
				return false
			}

			if request.CreatedAt.IsZero() {
				return false
			}

			// Verify request was sent
			if !requestSent {
				return false
			}

			// Verify sent request matches created request
			if sentRequest == nil {
				return false
			}

			if sentRequest.ID != request.ID {
				return false
			}

			if sentRequest.Operation.Type != request.Operation.Type {
				return false
			}

			if sentRequest.Operation.Target != request.Operation.Target {
				return false
			}

			return true
		},
		gen.OneConstOf("restart", "modify_config", "delete", "update", "query"),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 25: 确认响应处理正确性
// Validates: Requirements 7.3, 7.4
func TestProperty_ConfirmationResponseHandling(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any confirmation request, confirm should execute and cancel should terminate", prop.ForAll(
		func(operationType string, target string, shouldConfirm bool) bool {
			// Skip empty values
			if operationType == "" || target == "" {
				return true
			}

			// Create manager
			manager := NewManager(30*time.Second, func(req *Request) error {
				return nil
			})

			// Create operation
			operation := Operation{
				Type:   operationType,
				Target: target,
				Params: map[string]interface{}{
					"test": "value",
				},
			}

			// Create request
			ctx := context.Background()
			request, err := manager.CreateRequest(ctx, operation)
			if err != nil {
				return false
			}

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

			// Respond to request
			var respondErr error
			if shouldConfirm {
				respondErr = manager.Confirm(request.ID)
			} else {
				respondErr = manager.Cancel(request.ID)
			}

			if respondErr != nil {
				return false
			}

			// Wait for response
			select {
			case response := <-responseChan:
				// Verify response
				if response == nil {
					return false
				}

				if response.RequestID != request.ID {
					return false
				}

				// Verify action matches what we did
				expectedAction := ActionCancelled
				if shouldConfirm {
					expectedAction = ActionConfirmed
				}

				if response.Action != expectedAction {
					return false
				}

				if response.Timestamp.IsZero() {
					return false
				}

				// Verify timestamp is recent
				if time.Since(response.Timestamp) > 1*time.Second {
					return false
				}

				return true

			case err := <-errorChan:
				// Should not get error
				_ = err
				return false

			case <-time.After(1 * time.Second):
				// Timeout - should not happen
				return false
			}
		},
		gen.OneConstOf("restart", "modify_config", "delete", "update", "query"),
		gen.Identifier(),
		gen.Bool(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property: Request timeout handling
func TestProperty_RequestTimeoutHandling(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("for any request, timeout should trigger automatic cancellation", prop.ForAll(
		func(operationType string, target string) bool {
			// Skip empty values
			if operationType == "" || target == "" {
				return true
			}

			// Create manager with short timeout
			manager := NewManager(100*time.Millisecond, func(req *Request) error {
				return nil
			})

			// Create operation
			operation := Operation{
				Type:   operationType,
				Target: target,
				Params: map[string]interface{}{
					"test": "value",
				},
			}

			// Create request
			ctx := context.Background()
			request, err := manager.CreateRequest(ctx, operation)
			if err != nil {
				return false
			}

			// Wait for response (should timeout)
			response, err := manager.WaitForResponse(ctx, request.ID)
			if err != nil {
				return false
			}

			// Verify response is timeout
			if response == nil {
				return false
			}

			if response.Action != ActionTimeout {
				return false
			}

			if response.RequestID != request.ID {
				return false
			}

			return true
		},
		gen.OneConstOf("restart", "modify_config", "delete"),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property: Multiple requests can be handled concurrently
func TestProperty_ConcurrentRequestHandling(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("multiple requests should be handled independently", prop.ForAll(
		func(requestCount uint8) bool {
			// Limit request count
			if requestCount == 0 || requestCount > 10 {
				return true
			}

			// Create manager
			manager := NewManager(30*time.Second, func(req *Request) error {
				return nil
			})

			// Create multiple requests
			requests := make([]*Request, requestCount)
			ctx := context.Background()

			for i := uint8(0); i < requestCount; i++ {
				operation := Operation{
					Type:   "restart",
					Target: "service-" + string(rune('A'+i)),
					Params: map[string]interface{}{},
				}

				req, err := manager.CreateRequest(ctx, operation)
				if err != nil {
					return false
				}
				requests[i] = req
			}

			// Start goroutines to wait for responses
			responseChan := make(chan *Response, requestCount)
			errorChan := make(chan error, requestCount)

			for _, req := range requests {
				go func(requestID string) {
					resp, err := manager.WaitForResponse(ctx, requestID)
					if err != nil {
						errorChan <- err
						return
					}
					responseChan <- resp
				}(req.ID)
			}

			// Give goroutines time to start waiting
			time.Sleep(10 * time.Millisecond)

			// Confirm all requests
			for _, req := range requests {
				if err := manager.Confirm(req.ID); err != nil {
					return false
				}
			}

			// Collect all responses
			responses := make(map[string]*Response)
			for i := uint8(0); i < requestCount; i++ {
				select {
				case response := <-responseChan:
					responses[response.RequestID] = response
				case <-errorChan:
					return false
				case <-time.After(1 * time.Second):
					return false
				}
			}

			// Verify all responses
			for _, req := range requests {
				response, exists := responses[req.ID]
				if !exists {
					return false
				}

				if response.Action != ActionConfirmed {
					return false
				}

				if response.RequestID != req.ID {
					return false
				}
			}

			return true
		},
		gen.UInt8Range(1, 10),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property: Risk level is determined correctly
func TestProperty_RiskLevelDetermination(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("risk level should be determined based on operation type", prop.ForAll(
		func(operationType string, target string) bool {
			// Skip empty values
			if operationType == "" || target == "" {
				return true
			}

			// Create manager
			manager := NewManager(30*time.Second, func(req *Request) error {
				return nil
			})

			// Create operation
			operation := Operation{
				Type:   operationType,
				Target: target,
				Params: map[string]interface{}{},
			}

			// Create request
			ctx := context.Background()
			request, err := manager.CreateRequest(ctx, operation)
			if err != nil {
				return false
			}

			// Verify risk level
			expectedRiskLevel := determineRiskLevel(operationType)
			if request.RiskLevel != expectedRiskLevel {
				return false
			}

			// Verify high-risk operations
			if operationType == "delete" || operationType == "destroy" || operationType == "terminate" {
				if request.RiskLevel != RiskLevelHigh {
					return false
				}
			}

			// Verify medium-risk operations
			if operationType == "restart" || operationType == "modify_config" || operationType == "update" {
				if request.RiskLevel != RiskLevelMedium {
					return false
				}
			}

			return true
		},
		gen.OneConstOf("delete", "destroy", "terminate", "restart", "modify_config", "update", "query", "read"),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Additional property: Request cannot be responded to twice
func TestProperty_RequestSingleResponse(t *testing.T) {
	properties := gopter.NewProperties(nil)

	properties.Property("a request should only accept one response", prop.ForAll(
		func(operationType string, target string) bool {
			// Skip empty values
			if operationType == "" || target == "" {
				return true
			}

			// Create manager
			manager := NewManager(30*time.Second, func(req *Request) error {
				return nil
			})

			// Create operation
			operation := Operation{
				Type:   operationType,
				Target: target,
				Params: map[string]interface{}{},
			}

			// Create request
			ctx := context.Background()
			request, err := manager.CreateRequest(ctx, operation)
			if err != nil {
				return false
			}

			// Confirm the request
			if err := manager.Confirm(request.ID); err != nil {
				return false
			}

			// Try to confirm again (should fail)
			err = manager.Confirm(request.ID)
			if err == nil {
				return false // Should return error
			}

			// Try to cancel (should also fail)
			err = manager.Cancel(request.ID)
			if err == nil {
				return false // Should return error
			}

			return true
		},
		gen.OneConstOf("restart", "modify_config", "delete"),
		gen.Identifier(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}
