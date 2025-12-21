// Package utils provides utility functions for the OpsGenius Backend.
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"regexp"
	"strings"
)

// GenerateID generates a random unique ID.
func GenerateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple timestamp-based approach if random fails
		// This should rarely happen in practice
		return "fallback-id"
	}
	return hex.EncodeToString(bytes)
}

// SanitizeForLog sanitizes sensitive data for logging.
func SanitizeForLog(data string) string {
	// Patterns for sensitive data
	patterns := []struct {
		regex       *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`), "$1=***"},
		{regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*[:=]\s*\S+`), "$1=***"},
		{regexp.MustCompile(`(?i)(token|bearer)\s*[:=]\s*\S+`), "$1=***"},
		{regexp.MustCompile(`(?i)(secret)\s*[:=]\s*\S+`), "$1=***"},
		{regexp.MustCompile(`(?i)(authorization)\s*[:=]\s*\S+`), "$1=***"},
	}
	
	result := data
	for _, p := range patterns {
		result = p.regex.ReplaceAllString(result, p.replacement)
	}
	
	return result
}

// MaskString masks a string, showing only the first and last few characters.
func MaskString(s string, visibleChars int) string {
	if len(s) <= visibleChars*2 {
		return strings.Repeat("*", len(s))
	}
	
	return s[:visibleChars] + strings.Repeat("*", len(s)-visibleChars*2) + s[len(s)-visibleChars:]
}
