package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestNew(t *testing.T) {
	err := New(CodeParamError, "invalid parameter")
	if err.Code != CodeParamError {
		t.Errorf("expected code %d, got %d", CodeParamError, err.Code)
	}
	if err.Message != "invalid parameter" {
		t.Errorf("expected message 'invalid parameter', got '%s'", err.Message)
	}
	if err.HTTPStatus != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, err.HTTPStatus)
	}
}

func TestWrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := Wrap(originalErr, CodeInternalError, "wrapped error")

	if err.Code != CodeInternalError {
		t.Errorf("expected code %d, got %d", CodeInternalError, err.Code)
	}
	if err.Message != "wrapped error" {
		t.Errorf("expected message 'wrapped error', got '%s'", err.Message)
	}
	if err.Err != originalErr {
		t.Error("expected wrapped original error")
	}
}

func TestWrapf(t *testing.T) {
	originalErr := errors.New("original error")
	err := Wrapf(originalErr, CodeInternalError, "wrapped error: %s", "details")

	expectedMsg := "wrapped error: details"
	if err.Message != expectedMsg {
		t.Errorf("expected message '%s', got '%s'", expectedMsg, err.Message)
	}
}

func TestWithError(t *testing.T) {
	err := New(CodeParamError, "invalid parameter").
		WithDetail("field 'name' is required").
		WithHTTPStatus(http.StatusUnprocessableEntity)

	if err.Detail != "field 'name' is required" {
		t.Errorf("expected detail 'field 'name' is required', got '%s'", err.Detail)
	}
	if err.HTTPStatus != http.StatusUnprocessableEntity {
		t.Errorf("expected status %d, got %d", http.StatusUnprocessableEntity, err.HTTPStatus)
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		err      *AppError
		expected string
	}{
		{
			name:     "error without wrap",
			err:      New(CodeParamError, "invalid parameter"),
			expected: "[1001] invalid parameter",
		},
		{
			name:     "error with wrap",
			err:      Wrap(errors.New("original"), CodeParamError, "invalid parameter"),
			expected: "[1001] invalid parameter: original",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, got)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	originalErr := errors.New("original error")
	err := Wrap(originalErr, CodeInternalError, "wrapped error")

	if unwrapped := errors.Unwrap(err); unwrapped != originalErr {
		t.Error("expected to unwrap original error")
	}
}

func TestGetHTTPStatus(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected int
	}{
		{"success", CodeSuccess, http.StatusOK},
		{"param error", CodeParamError, http.StatusBadRequest},
		{"not found", CodeNotFound, http.StatusNotFound},
		{"unauthorized", CodeUnauthorized, http.StatusUnauthorized},
		{"forbidden", CodeForbidden, http.StatusForbidden},
		{"conflict", CodeConflict, http.StatusConflict},
		{"SSH error", CodeSSHError, http.StatusOK},
		{"LLM error", CodeLLMError, http.StatusOK},
		{"internal error", CodeInternalError, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getHTTPStatus(tt.code); got != tt.expected {
				t.Errorf("code %d: expected status %d, got %d", tt.code, tt.expected, got)
			}
		})
	}
}

func TestIsAppError(t *testing.T) {
	appErr := New(CodeParamError, "test")
	stdErr := errors.New("standard error")

	if !IsAppError(appErr) {
		t.Error("expected true for AppError")
	}
	if IsAppError(stdErr) {
		t.Error("expected false for standard error")
	}
}

func TestGetCode(t *testing.T) {
	appErr := New(CodeParamError, "test")
	stdErr := errors.New("standard error")

	if got := GetCode(appErr); got != CodeParamError {
		t.Errorf("expected code %d, got %d", CodeParamError, got)
	}
	if got := GetCode(stdErr); got != CodeUnknownError {
		t.Errorf("expected code %d, got %d", CodeUnknownError, got)
	}
}

func TestGetMessage(t *testing.T) {
	appErr := New(CodeParamError, "test message")
	stdErr := errors.New("standard error")

	if got := GetMessage(appErr); got != "test message" {
		t.Errorf("expected message 'test message', got '%s'", got)
	}
	if got := GetMessage(stdErr); got != "未知错误" {
		t.Errorf("expected message '未知错误', got '%s'", got)
	}
}
