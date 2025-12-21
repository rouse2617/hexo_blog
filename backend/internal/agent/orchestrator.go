package agent

import (
	"context"
	"fmt"
	"sync"
)

// SubTask represents a single task in the orchestration
type SubTask struct {
	ID           string
	Description  string
	Dependencies []string // IDs of tasks that must complete before this one
	Execute      func(ctx context.Context, inputs map[string]interface{}) (interface{}, error)
}

// orchestrator implements the TaskOrchestrator interface
type orchestrator struct {
	mu sync.RWMutex
}

// NewTaskOrchestrator creates a new task orchestrator
func NewTaskOrchestrator() TaskOrchestrator {
	return &orchestrator{}
}

// newOrchestratorForTesting creates an orchestrator for testing purposes
func newOrchestratorForTesting() *orchestrator {
	return &orchestrator{}
}

// Execute orchestrates the execution of tasks based on the intent
func (o *orchestrator) Execute(ctx context.Context, intent *Intent) (<-chan TaskResult, error) {
	resultChan := make(chan TaskResult, 10)

	// Decompose intent into subtasks
	subtasks, err := o.decomposeIntent(intent)
	if err != nil {
		close(resultChan)
		return resultChan, fmt.Errorf("failed to decompose intent: %w", err)
	}

	// Execute subtasks in a goroutine
	go func() {
		defer close(resultChan)
		o.executeSubtasks(ctx, subtasks, resultChan)
	}()

	return resultChan, nil
}

// decomposeIntent breaks down an intent into subtasks
func (o *orchestrator) decomposeIntent(intent *Intent) ([]*SubTask, error) {
	if intent == nil {
		return nil, fmt.Errorf("intent cannot be nil")
	}

	// For now, create simple subtasks based on intent type
	var subtasks []*SubTask

	switch intent.Type {
	case IntentTypeQuery:
		subtasks = []*SubTask{
			{
				ID:          "validate",
				Description: "Validate query parameters",
				Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
					return map[string]interface{}{"validated": true}, nil
				},
			},
			{
				ID:           "fetch",
				Description:  "Fetch data",
				Dependencies: []string{"validate"},
				Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
					return map[string]interface{}{"data": "query result"}, nil
				},
			},
		}
	case IntentTypeAnalyze:
		subtasks = []*SubTask{
			{
				ID:          "collect",
				Description: "Collect metrics",
				Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
					return map[string]interface{}{"metrics": []string{"cpu", "memory"}}, nil
				},
			},
			{
				ID:           "analyze",
				Description:  "Analyze metrics",
				Dependencies: []string{"collect"},
				Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
					return map[string]interface{}{"analysis": "system healthy"}, nil
				},
			},
		}
	case IntentTypeOperate:
		subtasks = []*SubTask{
			{
				ID:          "prepare",
				Description: "Prepare operation",
				Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
					return map[string]interface{}{"prepared": true}, nil
				},
			},
			{
				ID:           "execute",
				Description:  "Execute operation",
				Dependencies: []string{"prepare"},
				Execute: func(ctx context.Context, inputs map[string]interface{}) (interface{}, error) {
					return map[string]interface{}{"executed": true}, nil
				},
			},
		}
	default:
		return nil, fmt.Errorf("unknown intent type: %v", intent.Type)
	}

	return subtasks, nil
}

// executeSubtasks executes subtasks in dependency order
func (o *orchestrator) executeSubtasks(ctx context.Context, subtasks []*SubTask, resultChan chan<- TaskResult) {
	completed := make(map[string]interface{})
	failed := make(map[string]bool)

	// Build dependency graph
	remaining := make([]*SubTask, len(subtasks))
	copy(remaining, subtasks)

	for len(remaining) > 0 {
		// Find tasks that can be executed (all dependencies met)
		var executable []*SubTask
		var stillRemaining []*SubTask

		for _, task := range remaining {
			canExecute := true
			for _, dep := range task.Dependencies {
				if failed[dep] {
					// Dependency failed, mark this task as failed too
					resultChan <- TaskResult{
						Success: false,
						Error:   fmt.Sprintf("dependency %s failed", dep),
					}
					failed[task.ID] = true
					canExecute = false
					break
				}
				if _, ok := completed[dep]; !ok {
					canExecute = false
					break
				}
			}

			if failed[task.ID] {
				continue
			}

			if canExecute {
				executable = append(executable, task)
			} else {
				stillRemaining = append(stillRemaining, task)
			}
		}

		if len(executable) == 0 && len(stillRemaining) > 0 {
			// Deadlock: no tasks can execute but some remain
			for range stillRemaining {
				resultChan <- TaskResult{
					Success: false,
					Error:   "circular dependency or missing dependency",
				}
			}
			break
		}

		// Execute all executable tasks
		for _, task := range executable {
			select {
			case <-ctx.Done():
				resultChan <- TaskResult{
					Success: false,
					Error:   ctx.Err().Error(),
				}
				return
			default:
				// Prepare inputs from completed tasks
				inputs := make(map[string]interface{})
				for _, dep := range task.Dependencies {
					if result, ok := completed[dep]; ok {
						inputs[dep] = result
					}
				}

				// Execute task
				result, err := task.Execute(ctx, inputs)
				if err != nil {
					resultChan <- TaskResult{
						Success: false,
						Error:   err.Error(),
					}
					failed[task.ID] = true
				} else {
					resultChan <- TaskResult{
						Success: true,
						Data:    result,
					}
					completed[task.ID] = result
				}
			}
		}

		remaining = stillRemaining
	}
}
