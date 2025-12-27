package security

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

type CommandRule struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Enabled bool   `json:"enabled"`
}

type CommandPolicy struct {
	Enabled bool          `json:"enabled"`
	Rules   []CommandRule `json:"rules"`
}

type PolicyStore struct {
	mu       sync.RWMutex
	path     string
	policy   CommandPolicy
	compiled []*regexp.Regexp
}

func NewPolicyStore(path string, defaultPolicy CommandPolicy) *PolicyStore {
	s := &PolicyStore{path: path, policy: defaultPolicy}
	_ = s.Load()
	_ = s.compileLocked()
	return s
}

func (s *PolicyStore) Get() CommandPolicy {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.policy
}

func (s *PolicyStore) Set(p CommandPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policy = p
	if err := s.compileLocked(); err != nil {
		return err
	}
	return s.saveLocked()
}

func (s *PolicyStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
			return s.saveLocked()
		}
		return err
	}

	var p CommandPolicy
	if err := json.Unmarshal(b, &p); err != nil {
		return err
	}
	s.policy = p
	return s.compileLocked()
}

func (s *PolicyStore) compileLocked() error {
	s.compiled = s.compiled[:0]
	for _, r := range s.policy.Rules {
		if !r.Enabled {
			continue
		}
		re, err := regexp.Compile(r.Pattern)
		if err != nil {
			return err
		}
		s.compiled = append(s.compiled, re)
	}
	return nil
}

func (s *PolicyStore) saveLocked() error {
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	b, err := json.MarshalIndent(s.policy, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, b, 0o644)
}

func looksUnsafeShell(cmd string) bool {
	c := strings.ToLower(cmd)
	unsafe := []string{"`", "$(", ">", "<", "&&", "||", ";", "\n", "\r"}
	for _, tok := range unsafe {
		if strings.Contains(c, tok) {
			return true
		}
	}
	return false
}

func (s *PolicyStore) Check(cmd string) (bool, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.policy.Enabled {
		return true, "policy_disabled"
	}
	if strings.TrimSpace(cmd) == "" {
		return false, "empty_command"
	}
	if looksUnsafeShell(cmd) {
		return false, "unsafe_shell_syntax"
	}
	for _, re := range s.compiled {
		if re.MatchString(cmd) {
			return true, "matched_whitelist"
		}
	}
	return false, "not_in_whitelist"
}

func DefaultCommandPolicy() CommandPolicy {
	return CommandPolicy{
		Enabled: true,
		Rules: []CommandRule{
			{Name: "docker_stats", Pattern: `^docker\s+stats(\s+--all)?(\s+--no-stream)?$`, Enabled: true},
			{Name: "docker_ps", Pattern: `^docker\s+ps(\s+.*)?$`, Enabled: true},
			{Name: "docker_inspect", Pattern: `^docker\s+inspect\s+.+$`, Enabled: true},
			{Name: "docker_logs", Pattern: `^docker\s+logs(\s+--tail\s+\d+)?(\s+.+)?$`, Enabled: true},
			{Name: "uptime", Pattern: `^uptime$`, Enabled: true},
			{Name: "top", Pattern: `^top\s+-b\s+-n\s+\d+(\s+-d\s+\d+)?$`, Enabled: true},
			{Name: "free", Pattern: `^free(\s+-h)?$`, Enabled: true},
			{Name: "df", Pattern: `^df(\s+-h)?$`, Enabled: true},
			{Name: "ss", Pattern: `^ss(\s+.*)?$`, Enabled: true},
			{Name: "netstat", Pattern: `^netstat(\s+.*)?$`, Enabled: true},
			{Name: "ps", Pattern: `^ps\s+.*$`, Enabled: true},
			{Name: "cat", Pattern: `^cat\s+[^>]+$`, Enabled: true},
			{Name: "grep", Pattern: `^grep\s+.*$`, Enabled: true},
			{Name: "head", Pattern: `^head\s+.*$`, Enabled: true},
			{Name: "tail", Pattern: `^tail\s+.*$`, Enabled: true},
			{Name: "ls", Pattern: `^ls(\s+.*)?$`, Enabled: true},
			{Name: "journalctl", Pattern: `^journalctl(\s+--no-pager)?(\s+.*)?$`, Enabled: true},
		},
	}
}
