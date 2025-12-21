package agent

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
)

// Feature: ops-genius-backend, Property 16: 任务分解非平凡性
// *对于任何*复杂任务，Task Orchestrator 应该将其分解为至少 2 个子任务。
// **Validates: Requirements 5.1**
func TestProperty_TaskDecompositionNonTrivial(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("complex tasks should be decomposed into at least 2 subtasks", prop.ForAll(
		func(intentType IntentType) bool {
			// Only test valid intent types
			if intentType != IntentTypeQuery && intentType != IntentTypeAnalyze && intentType != IntentTypeOperate {
				return true // Skip invalid types
			}

			orchestrator := newOrchestratorForTesting()

			intent := &Intent{
				Type:     intentType,
				Entities: map[string]interface{}{"test": "data"},
			}

			subtasks, err := orchestrator.decomposeIntent(intent)
			if err != nil {
				return false
			}

			// Verify at least 2 subtasks (non-trivial decomposition)
			if len(subtasks) < 2 {
				return false
			}

			// Verify all subtasks have IDs and descriptions
			for _, task := range subtasks {
				if task.ID == "" || task.Description == "" {
					return false
				}
				if task.Execute == nil {
					return false
				}
			}

			return true
		},
		genIntentType(),
	))

	properties.Property("decomposed tasks should have at least one dependency relationship", prop.ForAll(
		func(intentType IntentType) bool {
			// Only test valid intent types
			if intentType != IntentTypeQuery && intentType != IntentTypeAnalyze && intentType != IntentTypeOperate {
				return true
			}

			orchestrator := newOrchestratorForTesting()

			intent := &Intent{
				Type:     intentType,
				Entities: map[string]interface{}{"test": "data"},
			}

			subtasks, err := orchestrator.decomposeIntent(intent)
			if err != nil {
				return false
			}

			// At least one task should have dependencies (non-trivial)
			hasDependencies := false
			for _, task := range subtasks {
				if len(task.Dependencies) > 0 {
					hasDependencies = true
					break
				}
			}

			return hasDependencies
		},
		genIntentType(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 17: 子任务执行顺序正确性
// *对于任何*有依赖关系的子任务序列，执行顺序应该满足依赖约束（依赖的子任务先执行）。
// **Validates: Requirements 5.2**
func TestProperty_SubtaskExecutionOrderCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("subtasks should execute in dependency order", prop.ForAll(
		func(numTasks uint8) bool {
			// Limit to reasonable number of tasks
			if numTasks < 2 {
				numTasks = 2
			}
			if numTasks > 10 {
				numTasks = 10
			}

			orchestrator := newOrchestratorForTesting()

			// Create a chain of dependent tasks
			executionOrder := make([]string, 0)
			var mu sync.Mutex

			subtasks := make([]*SubTask, numTasks)
			for i := uint8(0); i < numTasks; i++ {
				taskID := fmt.Sprintf("task%d", i)
				var deps []string
				if i > 0 {
					// Each task depends on the previous one
					deps = []string{fmt.Sprintf("task%d", i-1)}
				}

				subtasks[i] = &SubTask{
					ID:           taskID,
					Description:  fmt.Sprintf("Task %d", i),
					Dependencies: deps,
					Execute: func(taskID string) func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						return func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
							mu.Lock()
							executionOrder = append(executionOrder, taskID)
							mu.Unlock()
							return map[string]interface{}{"result": taskID}, nil
						}
					}(taskID),
				}
			}

			// Execute subtasks
			resultChan := make(chan TaskResult, int(numTasks)*2)
			ctx := context.Background()

			go func() {
				defer close(resultChan)
				orchestrator.executeSubtasks(ctx, subtasks, resultChan)
			}()

			// Wait for completion
			successCount := 0
			for result := range resultChan {
				if result.Success {
					successCount++
				}
			}

			// Verify all tasks succeeded
			if successCount != int(numTasks) {
				return false
			}

			// Verify execution order matches dependency order
			mu.Lock()
			defer mu.Unlock()

			if len(executionOrder) != int(numTasks) {
				return false
			}

			for i := 0; i < int(numTasks); i++ {
				expectedID := fmt.Sprintf("task%d", i)
				if executionOrder[i] != expectedID {
					return false
				}
			}

			return true
		},
		gen.UInt8Range(2, 10),
	))

	properties.Property("tasks with satisfied dependencies should execute before dependent tasks", prop.ForAll(
		func(seed int64) bool {
			orchestrator := newOrchestratorForTesting()

			// Create a diamond dependency pattern
			// task1 -> task2, task3 -> task4
			executionOrder := make([]string, 0)
			var mu sync.Mutex

			subtasks := []*SubTask{
				{
					ID:          "task1",
					Description: "Root task",
					Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						mu.Lock()
						executionOrder = append(executionOrder, "task1")
						mu.Unlock()
						time.Sleep(time.Millisecond) // Small delay
						return map[string]interface{}{"result": "task1"}, nil
					},
				},
				{
					ID:           "task2",
					Description:  "Branch 1",
					Dependencies: []string{"task1"},
					Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						mu.Lock()
						executionOrder = append(executionOrder, "task2")
						mu.Unlock()
						return map[string]interface{}{"result": "task2"}, nil
					},
				},
				{
					ID:           "task3",
					Description:  "Branch 2",
					Dependencies: []string{"task1"},
					Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						mu.Lock()
						executionOrder = append(executionOrder, "task3")
						mu.Unlock()
						return map[string]interface{}{"result": "task3"}, nil
					},
				},
				{
					ID:           "task4",
					Description:  "Merge task",
					Dependencies: []string{"task2", "task3"},
					Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						mu.Lock()
						executionOrder = append(executionOrder, "task4")
						mu.Unlock()
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

			// Wait for completion
			successCount := 0
			for result := range resultChan {
				if result.Success {
					successCount++
				}
			}

			// Verify all tasks succeeded
			if successCount != 4 {
				return false
			}

			// Verify dependency constraints
			mu.Lock()
			defer mu.Unlock()

			if len(executionOrder) != 4 {
				return false
			}

			// task1 must be first
			if executionOrder[0] != "task1" {
				return false
			}

			// task4 must be last
			if executionOrder[3] != "task4" {
				return false
			}

			// task2 and task3 must come after task1 and before task4
			task2Index := -1
			task3Index := -1
			for i, id := range executionOrder {
				if id == "task2" {
					task2Index = i
				}
				if id == "task3" {
					task3Index = i
				}
			}

			if task2Index <= 0 || task2Index >= 3 {
				return false
			}
			if task3Index <= 0 || task3Index >= 3 {
				return false
			}

			return true
		},
		gen.Int64(),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 18: 子任务结果传递
// *对于任何*子任务链，前一个子任务的输出应该作为下一个子任务的输入。
// **Validates: Requirements 5.4**
func TestProperty_SubtaskResultPassing(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("subtask results should be passed to dependent tasks as inputs", prop.ForAll(
		func(initialValue int, numTasks uint8) bool {
			// Limit to reasonable number of tasks
			if numTasks < 2 {
				numTasks = 2
			}
			if numTasks > 8 {
				numTasks = 8
			}

			orchestrator := newOrchestratorForTesting()

			// Create a chain where each task increments the value
			subtasks := make([]*SubTask, numTasks)
			for i := uint8(0); i < numTasks; i++ {
				taskID := fmt.Sprintf("task%d", i)
				var deps []string
				if i > 0 {
					deps = []string{fmt.Sprintf("task%d", i-1)}
				}

				subtasks[i] = &SubTask{
					ID:           taskID,
					Description:  fmt.Sprintf("Task %d", i),
					Dependencies: deps,
					Execute: func(i uint8, taskID string) func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						return func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
							var value int
							if i == 0 {
								// First task uses initial value
								value = initialValue
							} else {
								// Subsequent tasks should receive previous task's result
								prevTaskID := fmt.Sprintf("task%d", i-1)
								prevResult, ok := inputs[prevTaskID]
								if !ok {
									return nil, fmt.Errorf("missing input from %s", prevTaskID)
								}

								prevMap, ok := prevResult.(map[string]interface{})
								if !ok {
									return nil, fmt.Errorf("invalid input type from %s", prevTaskID)
								}

								prevValue, ok := prevMap["value"].(int)
								if !ok {
									return nil, fmt.Errorf("missing value from %s", prevTaskID)
								}

								value = prevValue
							}

							// Increment and return
							return map[string]interface{}{"value": value + 1}, nil
						}
					}(i, taskID),
				}
			}

			// Execute subtasks
			resultChan := make(chan TaskResult, int(numTasks)*2)
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

			// Verify all tasks succeeded
			if len(results) != int(numTasks) {
				return false
			}

			for _, result := range results {
				if !result.Success {
					return false
				}
			}

			// Verify the final result has the correct value (initial + numTasks)
			lastResult := results[len(results)-1]
			if lastResult.Data == nil {
				return false
			}

			dataMap, ok := lastResult.Data.(map[string]interface{})
			if !ok {
				return false
			}

			finalValue, ok := dataMap["value"].(int)
			if !ok {
				return false
			}

			expectedValue := initialValue + int(numTasks)
			return finalValue == expectedValue
		},
		gen.IntRange(0, 100),
		gen.UInt8Range(2, 8),
	))

	properties.Property("dependent tasks should receive all dependency results", prop.ForAll(
		func(value1, value2 int) bool {
			orchestrator := newOrchestratorForTesting()

			// Create tasks where task3 depends on both task1 and task2
			subtasks := []*SubTask{
				{
					ID:          "task1",
					Description: "Task 1",
					Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						return map[string]interface{}{"value": value1}, nil
					},
				},
				{
					ID:          "task2",
					Description: "Task 2",
					Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						return map[string]interface{}{"value": value2}, nil
					},
				},
				{
					ID:           "task3",
					Description:  "Task 3 - merge",
					Dependencies: []string{"task1", "task2"},
					Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						// Verify both inputs are present
						result1, ok1 := inputs["task1"]
						result2, ok2 := inputs["task2"]

						if !ok1 || !ok2 {
							return nil, fmt.Errorf("missing dependencies")
						}

						map1, ok := result1.(map[string]interface{})
						if !ok {
							return nil, fmt.Errorf("invalid task1 result")
						}

						map2, ok := result2.(map[string]interface{})
						if !ok {
							return nil, fmt.Errorf("invalid task2 result")
						}

						val1, ok := map1["value"].(int)
						if !ok {
							return nil, fmt.Errorf("invalid task1 value")
						}

						val2, ok := map2["value"].(int)
						if !ok {
							return nil, fmt.Errorf("invalid task2 value")
						}

						// Sum the values
						return map[string]interface{}{"sum": val1 + val2}, nil
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

			// Verify all tasks succeeded
			if len(results) != 3 {
				return false
			}

			for _, result := range results {
				if !result.Success {
					return false
				}
			}

			// Verify the final result has the correct sum
			lastResult := results[len(results)-1]
			dataMap, ok := lastResult.Data.(map[string]interface{})
			if !ok {
				return false
			}

			sum, ok := dataMap["sum"].(int)
			if !ok {
				return false
			}

			return sum == value1+value2
		},
		gen.IntRange(0, 100),
		gen.IntRange(0, 100),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// Feature: ops-genius-backend, Property 19: 任务结果汇总完整性
// *对于任何*完成的任务，最终响应应该包含所有子任务的结果。
// **Validates: Requirements 5.5**
func TestProperty_TaskResultAggregationCompleteness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100
	properties := gopter.NewProperties(parameters)

	properties.Property("all subtask results should be reported", prop.ForAll(
		func(numTasks uint8) bool {
			// Limit to reasonable number of tasks
			if numTasks < 2 {
				numTasks = 2
			}
			if numTasks > 10 {
				numTasks = 10
			}

			orchestrator := newOrchestratorForTesting()

			// Create independent tasks (no dependencies)
			subtasks := make([]*SubTask, numTasks)
			for i := uint8(0); i < numTasks; i++ {
				taskID := fmt.Sprintf("task%d", i)
				subtasks[i] = &SubTask{
					ID:          taskID,
					Description: fmt.Sprintf("Task %d", i),
					Execute: func(i uint8) func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						return func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
							return map[string]interface{}{"taskNum": i}, nil
						}
					}(i),
				}
			}

			// Execute subtasks
			resultChan := make(chan TaskResult, int(numTasks)*2)
			ctx := context.Background()

			go func() {
				defer close(resultChan)
				orchestrator.executeSubtasks(ctx, subtasks, resultChan)
			}()

			// Collect all results
			results := make([]TaskResult, 0)
			for result := range resultChan {
				results = append(results, result)
			}

			// Verify we received exactly numTasks results
			if len(results) != int(numTasks) {
				return false
			}

			// Verify all results are successful
			for _, result := range results {
				if !result.Success {
					return false
				}
				if result.Data == nil {
					return false
				}
			}

			return true
		},
		gen.UInt8Range(2, 10),
	))

	properties.Property("task results should include data from all completed subtasks", prop.ForAll(
		func(intentType IntentType) bool {
			// Only test valid intent types
			if intentType != IntentTypeQuery && intentType != IntentTypeAnalyze && intentType != IntentTypeOperate {
				return true
			}

			orchestrator := NewTaskOrchestrator()

			intent := &Intent{
				Type:     intentType,
				Entities: map[string]interface{}{"test": "data"},
			}

			ctx := context.Background()
			resultChan, err := orchestrator.Execute(ctx, intent)
			if err != nil {
				return false
			}

			// Collect all results
			results := make([]TaskResult, 0)
			for result := range resultChan {
				results = append(results, result)
			}

			// Should have at least 2 results (non-trivial decomposition)
			if len(results) < 2 {
				return false
			}

			// All results should be successful
			successCount := 0
			for _, result := range results {
				if result.Success {
					successCount++
					// Each successful result should have data
					if result.Data == nil {
						return false
					}
				}
			}

			// All tasks should succeed for valid intents
			return successCount == len(results)
		},
		genIntentType(),
	))

	properties.Property("failed subtasks should also be reported in results", prop.ForAll(
		func(failAtIndex uint8, numTasks uint8) bool {
			// Limit to reasonable number of tasks
			if numTasks < 2 {
				numTasks = 2
			}
			if numTasks > 8 {
				numTasks = 8
			}

			// Ensure failAtIndex is within range
			if failAtIndex >= numTasks {
				failAtIndex = numTasks - 1
			}

			orchestrator := newOrchestratorForTesting()

			// Create tasks where one will fail
			subtasks := make([]*SubTask, numTasks)
			for i := uint8(0); i < numTasks; i++ {
				taskID := fmt.Sprintf("task%d", i)
				var deps []string
				if i > 0 {
					deps = []string{fmt.Sprintf("task%d", i-1)}
				}

				subtasks[i] = &SubTask{
					ID:           taskID,
					Description:  fmt.Sprintf("Task %d", i),
					Dependencies: deps,
					Execute: func(i uint8) func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
						return func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
							if i == failAtIndex {
								return nil, fmt.Errorf("task%d failed", i)
							}
							return map[string]interface{}{"taskNum": i}, nil
						}
					}(i),
				}
			}

			// Execute subtasks
			resultChan := make(chan TaskResult, int(numTasks)*2)
			ctx := context.Background()

			go func() {
				defer close(resultChan)
				orchestrator.executeSubtasks(ctx, subtasks, resultChan)
			}()

			// Collect all results
			results := make([]TaskResult, 0)
			for result := range resultChan {
				results = append(results, result)
			}

			// Should receive results for all tasks
			if len(results) != int(numTasks) {
				return false
			}

			// At least one result should be a failure
			hasFailure := false
			for _, result := range results {
				if !result.Success {
					hasFailure = true
					// Failed results should have error messages
					if result.Error == "" {
						return false
					}
				}
			}

			return hasFailure
		},
		gen.UInt8Range(0, 7),
		gen.UInt8Range(2, 8),
	))

	properties.TestingRun(t, gopter.ConsoleReporter(false))
}

// genIntentType generates random IntentType values
func genIntentType() gopter.Gen {
	return gen.OneConstOf(IntentTypeQuery, IntentTypeAnalyze, IntentTypeOperate)
}
