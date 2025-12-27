package security

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CommandAuditEvent struct {
	Time      time.Time `json:"time"`
	SessionID string    `json:"session_id,omitempty"`
	Tool      string    `json:"tool,omitempty"`
	Host      string    `json:"host"`
	Command   string    `json:"command"`
	Allowed   bool      `json:"allowed"`
	Reason    string    `json:"reason,omitempty"`
	ElapsedMs int64     `json:"elapsed_ms,omitempty"`
	Error     string    `json:"error,omitempty"`
}

type AuditLogger struct {
	mu   sync.Mutex
	path string
}

func NewAuditLogger(path string) *AuditLogger {
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	return &AuditLogger{path: path}
}

func (l *AuditLogger) Append(e CommandAuditEvent) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_ = os.MkdirAll(filepath.Dir(l.path), 0o755)
	f, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	_, err = f.Write(append(b, '\n'))
	return err
}

func (l *AuditLogger) ReadLastN(n int) ([]CommandAuditEvent, error) {
	if n <= 0 {
		return []CommandAuditEvent{}, nil
	}
	f, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []CommandAuditEvent{}, nil
		}
		return nil, err
	}
	defer f.Close()

	// simple approach: read all lines then slice last N
	scanner := bufio.NewScanner(f)
	lines := make([]string, 0)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	start := 0
	if len(lines) > n {
		start = len(lines) - n
	}

	out := make([]CommandAuditEvent, 0, len(lines)-start)
	for _, line := range lines[start:] {
		var e CommandAuditEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}
