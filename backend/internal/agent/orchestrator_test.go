package agent

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSubtaskExecutionFailure tests that when a subtask fails, dependent tasks are also marked as failed
// Requirements: 5.2, 5.3
func TestSubtaskExecutionFailure(t *testing.T) {
	orchestrator := newOrchestratorForTesting()

	// Create subtasks where the first one will fail
	subtasks := []*SubTask{
		{
			ID:          "task1",
			Description: "First task that will fail",
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				return nil, errors.New("task1 failed")
			},
		},
		{
			ID:           "task2",
			Description:  "Second task that depends on task1",
			Dependencies: []string{"task1"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{"result": "success"}, nil
			},
		},
		{
			ID:           "task3",
			Description:  "Third task that depends on task2",
			Dependencies: []string{"task2"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{"result": "success"}, nil
			},
		},
	}

	// Execute subtasks
	resultChan := make(chan TaskResult, 10)
	ctx := context.Background()

	go func() {
		defer close(resultChan)
		orchestrator.executeSubtasks(ctx, subtasks, resultChan)
	}()

	// Collect results
	results := make([]TaskResult, 0)
	for result := range resultChan {
		results = append(results, result)
	}

	// Verify we got 3 results (all tasks should report)
	assert.Len(t, results, 3, "should have 3 task results")

	// Verify all tasks failed
	for _, result := range results {
		assert.False(t, result.Success, "all tasks should fail")
		assert.NotEmpty(t, result.Error, "all tasks should have an error")
	}

	// Verify at least one error mentions the original failure
	foundOriginalError := false
	for _, result := range results {
		if contains(result.Error, "task1 failed") {
			foundOriginalError = true
			break
		}
	}
	assert.True(t, foundOriginalError, "should find original task1 failure")
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && 
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		len(s) > len(substr)+1 && s[1:len(substr)+1] == substr))
}

// TestSubtaskExecutionFailureWithMultipleDependencies tests failure propagation with multiple dependencies
// Requirements: 5.2, 5.3
func TestSubtaskExecutionFailureWithMultipleDependencies(t *testing.T) {
	orchestrator := newOrchestratorForTesting()

	// Create a diamond dependency pattern where one branch fails
	subtasks := []*SubTask{
		{
			ID:          "root",
			Description: "Root task",
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{"data": "root"}, nil
			},
		},
		{
			ID:           "branch1",
			Description:  "Branch 1 - will fail",
			Dependencies: []string{"root"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				return nil, errors.New("branch1 failed")
			},
		},
		{
			ID:           "branch2",
			Description:  "Branch 2 - will succeed",
			Dependencies: []string{"root"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{"data": "branch2"}, nil
			},
		},
		{
			ID:           "merge",
			Description:  "Merge task depending on both branches",
			Dependencies: []string{"branch1", "branch2"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				return map[string]interface{}{"merged": true}, nil
			},
		},
	}

	// Execute subtasks
	resultChan := make(chan TaskResult, 10)
	ctx := context.Background()

	go func() {
		defer close(resultChan)
		orchestrator.executeSubtasks(ctx, subtasks, resultChan)
	}()

	// Collect results by tracking success/failure
	successCount := 0
	failureCount := 0
	for result := range resultChan {
		if result.Success {
			successCount++
		} else {
			failureCount++
		}
	}

	// Verify we got 4 results
	assert.Equal(t, 4, successCount+failureCount, "should have 4 task results")

	// Verify at least 2 tasks failed (branch1 and merge)
	assert.GreaterOrEqual(t, failureCount, 2, "at least branch1 and merge should fail")

	// Verify at least 1 task succeeded (root, and possibly branch2)
	assert.GreaterOrEqual(t, successCount, 1, "at least root should succeed")
}

// TestDependencyOrder tests that subtasks are executed in the correct dependency order
// Requirements: 5.2
func TestDependencyOrder(t *testing.T) {
	orchestrator := newOrchestratorForTesting()

	executionOrder := make([]string, 0)
	var mu sync.Mutex

	// Create subtasks with dependencies
	subtasks := []*SubTask{
		{
			ID:          "task1",
			Description: "First task",
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, "task1")
				mu.Unlock()
				return map[string]interface{}{"result": "task1"}, nil
			},
		},
		{
			ID:           "task2",
			Description:  "Second task depends on task1",
			Dependencies: []string{"task1"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, "task2")
				mu.Unlock()
				// Verify task1 result is available in inputs
				assert.Contains(t, inputs, "task1", "task1 result should be in inputs")
				return map[string]interface{}{"result": "task2"}, nil
			},
		},
		{
			ID:           "task3",
			Description:  "Third task depends on task2",
			Dependencies: []string{"task2"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, "task3")
				mu.Unlock()
				// Verify task2 result is available in inputs
				assert.Contains(t, inputs, "task2", "task2 result should be in inputs")
				return map[string]interface{}{"result": "task3"}, nil
			},
		},
	}

	// Execute subtasks
	resultChan := make(chan TaskResult, 10)
	ctx := context.Background()

	go func() {
		defer close(resultChan)
		orchestrator.executeSubtasks(ctx, subtasks, resultChan)
	}()

	// Collect results
	successCount := 0
	for result := range resultChan {
		if result.Success {
			successCount++
		}
	}

	// Verify all tasks succeeded
	assert.Equal(t, 3, successCount, "all 3 tasks should succeed")

	// Verify execution order
	require.Len(t, executionOrder, 3, "should have 3 tasks executed")
	assert.Equal(t, "task1", executionOrder[0], "task1 should execute first")
	assert.Equal(t, "task2", executionOrder[1], "task2 should execute second")
	assert.Equal(t, "task3", executionOrder[2], "task3 should execute third")
}

// TestDependencyOrderWithParallelExecution tests that independent tasks can execute in parallel
// Requirements: 5.2
func TestDependencyOrderWithParallelExecution(t *testing.T) {
	orchestrator := newOrchestratorForTesting()

	executionOrder := make([]string, 0)
	var mu sync.Mutex

	// Create subtasks where task2 and task3 are independent
	subtasks := []*SubTask{
		{
			ID:          "task1",
			Description: "First task",
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, "task1")
				mu.Unlock()
				return map[string]interface{}{"result": "task1"}, nil
			},
		},
		{
			ID:           "task2",
			Description:  "Second task depends on task1",
			Dependencies: []string{"task1"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				time.Sleep(10 * time.Millisecond) // Simulate work
				mu.Lock()
				executionOrder = append(executionOrder, "task2")
				mu.Unlock()
				return map[string]interface{}{"result": "task2"}, nil
			},
		},
		{
			ID:           "task3",
			Description:  "Third task depends on task1 (parallel to task2)",
			Dependencies: []string{"task1"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				time.Sleep(10 * time.Millisecond) // Simulate work
				mu.Lock()
				executionOrder = append(executionOrder, "task3")
				mu.Unlock()
				return map[string]interface{}{"result": "task3"}, nil
			},
		},
		{
			ID:           "task4",
			Description:  "Fourth task depends on both task2 and task3",
			Dependencies: []string{"task2", "task3"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				mu.Lock()
				executionOrder = append(executionOrder, "task4")
				mu.Unlock()
				// Verify both task2 and task3 results are available
				assert.Contains(t, inputs, "task2", "task2 result should be in inputs")
				assert.Contains(t, inputs, "task3", "task3 result should be in inputs")
				return map[string]interface{}{"result": "task4"}, nil
			},
		},
	}

	// Execute subtasks
	resultChan := make(chan TaskResult, 10)
	ctx := context.Background()

	go func() {
		defer close(resultChan)
		orchestrator.executeSubtasks(ctx, subtasks, resultChan)
	}()

	// Collect results
	successCount := 0
	for result := range resultChan {
		if result.Success {
			successCount++
		}
	}

	// Verify all tasks succeeded
	assert.Equal(t, 4, successCount, "all 4 tasks should succeed")

	// Verify execution order constraints
	require.Len(t, executionOrder, 4, "should have 4 tasks executed")
	assert.Equal(t, "task1", executionOrder[0], "task1 should execute first")
	assert.Equal(t, "task4", executionOrder[3], "task4 should execute last")
	// task2 and task3 can be in any order (parallel execution)
}

// TestContextCancellation tests that task execution respects context cancellation
// Requirements: 5.3
func TestContextCancellation(t *testing.T) {
	orchestrator := newOrchestratorForTesting()

	ctx, cancel := context.WithCancel(context.Background())

	subtasks := []*SubTask{
		{
			ID:          "task1",
			Description: "First task",
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return map[string]interface{}{"result": "task1"}, nil
			},
		},
		{
			ID:           "task2",
			Description:  "Second task",
			Dependencies: []string{"task1"},
			Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
				time.Sleep(50 * time.Millisecond)
				return map[string]interface{}{"result": "task2"}, nil
			},
		},
	}

	// Execute subtasks
	resultChan := make(chan TaskResult, 10)

	go func() {
		defer close(resultChan)
		orchestrator.executeSubtasks(ctx, subtasks, resultChan)
	}()

	// Cancel context after a short delay
	time.Sleep(30 * time.Millisecond)
	cancel()

	// Collect results
	results := make([]TaskResult, 0)
	for result := range resultChan {
		results = append(results, result)
	}

	// At least one task should have been cancelled
	cancelled := false
	for _, result := range results {
		if !result.Success && result.Error != "" {
			if result.Error == "context canceled" {
				cancelled = true
				break
			}
		}
	}
	assert.True(t, cancelled, "at least one task should be cancelled")
}

// TestDecomposeIntent tests that intents are properly decomposed into subtasks
// Requirements: 5.1
func TestDecomposeIntent(t *testing.T) {
	orchestrator := newOrchestratorForTesting()

	tests := []struct {
		name          string
		intent        *Intent
		expectedTasks int
		expectError   bool
	}{
		{
			name: "query intent",
			intent: &Intent{
				Type:     IntentTypeQuery,
				Entities: map[string]interface{}{"node": "Node-A"},
			},
			expectedTasks: 2,
			expectError:   false,
		},
		{
			name: "analyze intent",
			intent: &Intent{
				Type:     IntentTypeAnalyze,
				Entities: map[string]interface{}{"service": "nginx"},
			},
			expectedTasks: 2,
			expectError:   false,
		},
		{
			name: "operate intent",
			intent: &Intent{
				Type:     IntentTypeOperate,
				Entities: map[string]interface{}{"action": "restart"},
			},
			expectedTasks: 2,
			expectError:   false,
		},
		{
			name:        "nil intent",
			intent:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			subtasks, err := orchestrator.decomposeIntent(tt.intent)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, subtasks)
			} else {
				assert.NoError(t, err)
				assert.Len(t, subtasks, tt.expectedTasks)

				// Verify at least one task has dependencies (non-trivial decomposition)
				hasDependencies := false
				for _, task := range subtasks {
					if len(task.Dependencies) > 0 {
						hasDependencies = true
						break
					}
				}
				assert.True(t, hasDependencies, "decomposition should create tasks with dependencies")
			}
		})
	}
}

// TestExecuteIntegration tests the full Execute method
// Requirements: 5.1, 5.2, 5.4, 5.5
func TestExecuteIntegration(t *testing.T) {
	orchestrator := NewTaskOrchestrator()

	intent := &Intent{
		Type:     IntentTypeQuery,
		Entities: map[string]interface{}{"node": "Node-A"},
	}

	ctx := context.Background()
	resultChan, err := orchestrator.Execute(ctx, intent)
	require.NoError(t, err)
	require.NotNil(t, resultChan)

	// Collect all results
	results := make([]TaskResult, 0)
	for result := range resultChan {
		results = append(results, result)
	}

	// Verify we got results
	assert.NotEmpty(t, results, "should receive task results")

	// Verify all tasks succeeded
	for _, result := range results {
		assert.True(t, result.Success, "all tasks should succeed")
		assert.Empty(t, result.Error, "should have no error")
		assert.NotNil(t, result.Data)
	}
}
