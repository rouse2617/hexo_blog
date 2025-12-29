package config

import (
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	LLM      LLMConfig      `yaml:"llm"`
	SSH      SSHConfig      `yaml:"ssh"`
	Hosts    []HostConfig   `yaml:"hosts"`      // 预定义主机列表
	Database DatabaseConfig `yaml:"database"`
	Scripts  ScriptsConfig  `yaml:"scripts"`
	Agent    AgentConfig    `yaml:"agent"`
	Log      LogConfig      `yaml:"log"`
	MCP      []MCPConfig    `yaml:"mcp"`       // MCP 配置
	Auth     AuthConfig     `yaml:"auth"`      // 认证配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Addr string `yaml:"addr"` // 监听地址，如 :8080
	Mode string `yaml:"mode"` // 运行模式: debug / release
}

// LLMConfig LLM 配置
type LLMConfig struct {
	Endpoint  string        `yaml:"endpoint"`   // API 端点
	Model     string        `yaml:"model"`      // 模型名称
	Timeout   time.Duration `yaml:"timeout"`    // 请求超时
	MaxTokens int           `yaml:"max_tokens"` // 最大 token 数
	APIKey    string        `yaml:"api_key"`    // API Key (可选)
}

// SSHConfig SSH 配置
type SSHConfig struct {
	DefaultTimeout    time.Duration `yaml:"default_timeout"`    // 默认执行超时
	ConnectTimeout    time.Duration `yaml:"connect_timeout"`    // 连接超时
	MaxConnections    int           `yaml:"max_connections"`    // 最大连接数
	MaxConcurrent     int           `yaml:"max_concurrent"`     // 最大并发执行数
	KeepaliveInterval time.Duration `yaml:"keepalive_interval"` // 心跳间隔
	DefaultUser       string        `yaml:"default_user"`       // 默认用户
	DefaultKeyPath    string        `yaml:"default_key_path"`   // 默认密钥路径
}

// HostConfig 主机配置
type HostConfig struct {
	Name     string   `yaml:"name"`                // 主机名称（唯一标识）
	Host     string   `yaml:"host"`                // IP 地址或域名
	Port     int      `yaml:"port,omitempty"`      // SSH 端口，默认 22
	User     string   `yaml:"user,omitempty"`      // 用户名，默认使用 ssh.default_user
	Group    string   `yaml:"group,omitempty"`     // 分组
	Tags     []string `yaml:"tags,omitempty"`      // 标签
	AuthType string   `yaml:"auth_type,omitempty"` // 认证方式: password / key，默认 key
	Password string   `yaml:"password,omitempty"`  // 密码（auth_type=password 时使用）
	KeyPath  string   `yaml:"key_path,omitempty"`  // 私钥路径，默认使用 ssh.default_key_path
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver string `yaml:"driver"` // 数据库驱动: sqlite / postgres
	DSN    string `yaml:"dsn"`    // 数据源名称
}

// ScriptsConfig 脚本配置
type ScriptsConfig struct {
	Dir string `yaml:"dir"` // 脚本目录
}

// AgentConfig Agent 配置
type AgentConfig struct {
	MaxLoops       int    `yaml:"max_loops"`        // 最大循环次数
	Timeout        int    `yaml:"timeout"`          // 单次请求超时（秒）
	PromptVersion  string `yaml:"prompt_version"`   // 提示词版本: standard / enhanced
	EnableThinking bool   `yaml:"enable_thinking"`  // 启用思考过程输出
}

// LogConfig 日志配置
type LogConfig struct {
	Level  string `yaml:"level"`  // 日志级别: debug / info / warn / error
	Format string `yaml:"format"` // 日志格式: json / text
	Output string `yaml:"output"` // 输出位置: stdout / 文件路径
}

// MCPConfig MCP 配置
type MCPConfig struct {
	Name    string `yaml:"name"`    // MCP server 名称
	URL     string `yaml:"url"`     // MCP server 地址
	Timeout int    `yaml:"timeout"` // 超时时间（秒）
	Enabled bool   `yaml:"enabled"` // 是否启用
}

// AuthConfig 认证配置
type AuthConfig struct {
	Enabled       bool          `yaml:"enabled"`        // 是否启用认证
	Secret        string        `yaml:"secret"`         // JWT 密钥
	TokenDuration time.Duration `yaml:"token_duration"` // Token 有效期
	Issuer        string        `yaml:"issuer"`         // Token 签发者
}

// Load 从文件加载配置
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// 设置默认值
	cfg.setDefaults()

	// 从环境变量覆盖
	cfg.loadFromEnv()

	return cfg, nil
}

// setDefaults 设置默认值
func (c *Config) setDefaults() {
	// Server 默认值
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}
	if c.Server.Mode == "" {
		c.Server.Mode = "debug"
	}

	// LLM 默认值
	if c.LLM.Endpoint == "" {
		c.LLM.Endpoint = "http://localhost:11434/v1"
	}
	if c.LLM.Model == "" {
		c.LLM.Model = "qwen2.5:14b"
	}
	if c.LLM.Timeout == 0 {
		c.LLM.Timeout = 60 * time.Second
	}
	if c.LLM.MaxTokens == 0 {
		c.LLM.MaxTokens = 4096
	}

	// SSH 默认值
	if c.SSH.DefaultTimeout == 0 {
		c.SSH.DefaultTimeout = 30 * time.Second
	}
	if c.SSH.ConnectTimeout == 0 {
		c.SSH.ConnectTimeout = 10 * time.Second
	}
	if c.SSH.MaxConnections == 0 {
		c.SSH.MaxConnections = 100
	}
	if c.SSH.MaxConcurrent == 0 {
		c.SSH.MaxConcurrent = 20
	}
	if c.SSH.KeepaliveInterval == 0 {
		c.SSH.KeepaliveInterval = 30 * time.Second
	}
	if c.SSH.DefaultUser == "" {
		c.SSH.DefaultUser = "root"
	}
	if c.SSH.DefaultKeyPath == "" {
		c.SSH.DefaultKeyPath = "~/.ssh/id_rsa"
	}

	// Database 默认值
	if c.Database.Driver == "" {
		c.Database.Driver = "sqlite"
	}
	if c.Database.DSN == "" {
		c.Database.DSN = "./data/aiops.db"
	}

	// Scripts 默认值
	if c.Scripts.Dir == "" {
		c.Scripts.Dir = "./scripts"
	}

	// Agent 默认值
	if c.Agent.MaxLoops == 0 {
		c.Agent.MaxLoops = 10
	}
	if c.Agent.Timeout == 0 {
		c.Agent.Timeout = 300 // 5 分钟
	}
	if c.Agent.PromptVersion == "" {
		c.Agent.PromptVersion = "enhanced" // 默认使用增强版
	}
	// EnableThinking 默认为 false，等 Phase 2 实现后启用
	// if c.Agent.EnableThinking == false {
	// 	c.Agent.EnableThinking = true
	// }

	// Log 默认值
	if c.Log.Level == "" {
		c.Log.Level = "info"
	}
	if c.Log.Format == "" {
		c.Log.Format = "json"
	}
	if c.Log.Output == "" {
		c.Log.Output = "stdout"
	}

	// Auth 默认值
	if c.Auth.Secret == "" {
		c.Auth.Secret = "change-this-secret-in-production"
	}
	if c.Auth.TokenDuration == 0 {
		c.Auth.TokenDuration = 24 * time.Hour
	}
	if c.Auth.Issuer == "" {
		c.Auth.Issuer = "ai-ops"
	}
}

// loadFromEnv 从环境变量加载配置
func (c *Config) loadFromEnv() {
	// LLM API Key 优先从环境变量读取（支持多种环境变量名称）
	if apiKey := os.Getenv("LLM_API_KEY"); apiKey != "" {
		c.LLM.APIKey = apiKey
	} else if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		c.LLM.APIKey = apiKey
	}

	// LLM Endpoint 可从环境变量覆盖
	if endpoint := os.Getenv("LLM_ENDPOINT"); endpoint != "" {
		c.LLM.Endpoint = endpoint
	}

	// LLM Model 可从环境变量覆盖
	if model := os.Getenv("LLM_MODEL"); model != "" {
		c.LLM.Model = model
	}

	// 数据库 DSN 可从环境变量覆盖
	if dsn := os.Getenv("DATABASE_DSN"); dsn != "" {
		c.Database.DSN = dsn
	}

	// 服务器地址可从环境变量覆盖
	if addr := os.Getenv("SERVER_ADDR"); addr != "" {
		c.Server.Addr = addr
	}

	// 服务器模式可从环境变量覆盖
	if mode := os.Getenv("SERVER_MODE"); mode != "" {
		c.Server.Mode = mode
	}

	// JWT Secret 可从环境变量覆盖（安全优先）
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		c.Auth.Secret = secret
	}
}
