package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// 创建临时配置文件
	content := `
server:
  addr: ":9090"
  mode: release

llm:
  endpoint: "http://localhost:11434/v1"
  model: "qwen2.5:7b"
  timeout: 30s
  max_tokens: 2048

ssh:
  default_timeout: 60s
  max_concurrent: 10

database:
  driver: sqlite
  dsn: ./test.db

agent:
  max_loops: 5

log:
  level: debug
  format: text
`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	// 加载配置
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证配置值
	if cfg.Server.Addr != ":9090" {
		t.Errorf("Server.Addr 期望 :9090, 实际 %s", cfg.Server.Addr)
	}
	if cfg.Server.Mode != "release" {
		t.Errorf("Server.Mode 期望 release, 实际 %s", cfg.Server.Mode)
	}
	if cfg.LLM.Model != "qwen2.5:7b" {
		t.Errorf("LLM.Model 期望 qwen2.5:7b, 实际 %s", cfg.LLM.Model)
	}
	if cfg.LLM.Timeout != 30*time.Second {
		t.Errorf("LLM.Timeout 期望 30s, 实际 %v", cfg.LLM.Timeout)
	}
	if cfg.SSH.DefaultTimeout != 60*time.Second {
		t.Errorf("SSH.DefaultTimeout 期望 60s, 实际 %v", cfg.SSH.DefaultTimeout)
	}
	if cfg.SSH.MaxConcurrent != 10 {
		t.Errorf("SSH.MaxConcurrent 期望 10, 实际 %d", cfg.SSH.MaxConcurrent)
	}
	if cfg.Agent.MaxLoops != 5 {
		t.Errorf("Agent.MaxLoops 期望 5, 实际 %d", cfg.Agent.MaxLoops)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level 期望 debug, 实际 %s", cfg.Log.Level)
	}
}

func TestLoadDefaults(t *testing.T) {
	// 创建空配置文件
	content := `# 空配置，测试默认值`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	// 加载配置
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证默认值
	if cfg.Server.Addr != ":8080" {
		t.Errorf("Server.Addr 默认值期望 :8080, 实际 %s", cfg.Server.Addr)
	}
	if cfg.Server.Mode != "debug" {
		t.Errorf("Server.Mode 默认值期望 debug, 实际 %s", cfg.Server.Mode)
	}
	if cfg.LLM.Endpoint != "http://localhost:11434/v1" {
		t.Errorf("LLM.Endpoint 默认值错误: %s", cfg.LLM.Endpoint)
	}
	if cfg.SSH.DefaultTimeout != 30*time.Second {
		t.Errorf("SSH.DefaultTimeout 默认值期望 30s, 实际 %v", cfg.SSH.DefaultTimeout)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Errorf("Database.Driver 默认值期望 sqlite, 实际 %s", cfg.Database.Driver)
	}
	if cfg.Agent.MaxLoops != 10 {
		t.Errorf("Agent.MaxLoops 默认值期望 10, 实际 %d", cfg.Agent.MaxLoops)
	}
}

func TestLoadFromEnv(t *testing.T) {
	// 设置环境变量
	os.Setenv("LLM_API_KEY", "test-api-key")
	os.Setenv("DATABASE_DSN", "postgres://localhost/test")
	os.Setenv("SERVER_ADDR", ":3000")
	defer func() {
		os.Unsetenv("LLM_API_KEY")
		os.Unsetenv("DATABASE_DSN")
		os.Unsetenv("SERVER_ADDR")
	}()

	// 创建空配置文件
	content := `# 测试环境变量覆盖`
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("写入临时文件失败: %v", err)
	}
	tmpFile.Close()

	// 加载配置
	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("加载配置失败: %v", err)
	}

	// 验证环境变量覆盖
	if cfg.LLM.APIKey != "test-api-key" {
		t.Errorf("LLM.APIKey 期望 test-api-key, 实际 %s", cfg.LLM.APIKey)
	}
	if cfg.Database.DSN != "postgres://localhost/test" {
		t.Errorf("Database.DSN 期望 postgres://localhost/test, 实际 %s", cfg.Database.DSN)
	}
	if cfg.Server.Addr != ":3000" {
		t.Errorf("Server.Addr 期望 :3000, 实际 %s", cfg.Server.Addr)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := Load("nonexistent.yaml")
	if err == nil {
		t.Error("期望文件不存在错误，但没有返回错误")
	}
}
