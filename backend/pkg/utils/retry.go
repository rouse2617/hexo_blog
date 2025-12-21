// Package utils provides utility functions for the OpsGenius Backend.
package utils

import (
	"context"
	"math"
	"time"
)

// RetryConfig represents configuration for retry operations.
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
}

// DefaultRetryConfig returns the default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   time.Second,
		MaxDelay:    30 * time.Second,
	}
}

// RetryFunc is a function that can be retried.
type RetryFunc func(ctx context.Context) error

// Retry executes a function with exponential backoff retry.
func Retry(ctx context.Context, config RetryConfig, fn RetryFunc) error {
	var lastErr error
	
	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if err := fn(ctx); err == nil {
			return nil
		} else {
			lastErr = err
		}
		
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Calculate backoff delay
		delay := time.Duration(math.Pow(2, float64(attempt))) * config.BaseDelay
		if delay > config.MaxDelay {
			delay = config.MaxDelay
		}
		
		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	
	return lastErr
}

// IsRetryable determines if an error is retryable.
func IsRetryable(err error) bool {
	// TODO: Implement logic to determine if error is retryable
	// For now, assume all errors are retryable
	return err != nil
}
