// Package config provides configuration management functionality.
package config

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config represents the application configuration.
type Config struct {
	Server     ServerConfig      `mapstructure:"server" yaml:"server" json:"server"`
	LLM        LLMConfig         `mapstructure:"llm" yaml:"llm" json:"llm"`
	MCPServers []MCPServerConfig `mapstructure:"mcp_servers" yaml:"mcp_servers" json:"mcp_servers"`
	Database   DatabaseConfig    `mapstructure:"database" yaml:"database" json:"database"`
	Redis      RedisConfig       `mapstructure:"redis" yaml:"redis" json:"redis"`
	Security   SecurityConfig    `mapstructure:"security" yaml:"security" json:"security"`
	Logging    LoggingConfig     `mapstructure:"logging" yaml:"logging" json:"logging"`
}

// ServerConfig represents server configuration.
type ServerConfig struct {
	Host          string    `mapstructure:"host" yaml:"host" json:"host"`
	Port          int       `mapstructure:"port" yaml:"port" json:"port"`
	WebSocketPath string    `mapstructure:"websocket_path" yaml:"websocket_path" json:"websocket_path"`
	TLS           TLSConfig `mapstructure:"tls" yaml:"tls" json:"tls"`
}

// TLSConfig represents TLS configuration.
type TLSConfig struct {
	Enabled  bool   `mapstructure:"enabled" yaml:"enabled" json:"enabled"`
	CertFile string `mapstructure:"cert_file" yaml:"cert_file" json:"cert_file"`
	KeyFile  string `mapstructure:"key_file" yaml:"key_file" json:"key_file"`
}

// LLMConfig represents LLM configuration.
type LLMConfig struct {
	Provider    string  `mapstructure:"provider" yaml:"provider" json:"provider"`
	APIKey      string  `mapstructure:"api_key" yaml:"api_key" json:"api_key"`
	Model       string  `mapstructure:"model" yaml:"model" json:"model"`
	Temperature float64 `mapstructure:"temperature" yaml:"temperature" json:"temperature"`
	MaxTokens   int     `mapstructure:"max_tokens" yaml:"max_tokens" json:"max_tokens"`
}

// MCPServerConfig represents MCP server configuration.
type MCPServerConfig struct {
	ID        string            `mapstructure:"id" yaml:"id" json:"id"`
	Name      string            `mapstructure:"name" yaml:"name" json:"name"`
	Transport string            `mapstructure:"transport" yaml:"transport" json:"transport"`
	Command   string            `mapstructure:"command" yaml:"command,omitempty" json:"command,omitempty"`
	Args      []string          `mapstructure:"args" yaml:"args,omitempty" json:"args,omitempty"`
	Env       map[string]string `mapstructure:"env" yaml:"env,omitempty" json:"env,omitempty"`
	URL       string            `mapstructure:"url" yaml:"url,omitempty" json:"url,omitempty"`
	Auth      *AuthConfig       `mapstructure:"auth" yaml:"auth,omitempty" json:"auth,omitempty"`
}

// AuthConfig represents authentication configuration.
type AuthConfig struct {
	Type  string `mapstructure:"type" yaml:"type" json:"type"`
	Token string `mapstructure:"token" yaml:"token" json:"token"`
}

// DatabaseConfig represents database configuration.
type DatabaseConfig struct {
	Host           string `mapstructure:"host" yaml:"host" json:"host"`
	Port           int    `mapstructure:"port" yaml:"port" json:"port"`
	Database       string `mapstructure:"database" yaml:"database" json:"database"`
	User           string `mapstructure:"user" yaml:"user" json:"user"`
	Password       string `mapstructure:"password" yaml:"password" json:"password"`
	MaxConnections int    `mapstructure:"max_connections" yaml:"max_connections" json:"max_connections"`
}

// RedisConfig represents Redis configuration.
type RedisConfig struct {
	Host     string `mapstructure:"host" yaml:"host" json:"host"`
	Port     int    `mapstructure:"port" yaml:"port" json:"port"`
	Password string `mapstructure:"password" yaml:"password" json:"password"`
	DB       int    `mapstructure:"db" yaml:"db" json:"db"`
	PoolSize int    `mapstructure:"pool_size" yaml:"pool_size" json:"pool_size"`
}

// SecurityConfig represents security configuration.
type SecurityConfig struct {
	AuthEnabled    bool     `mapstructure:"auth_enabled" yaml:"auth_enabled" json:"auth_enabled"`
	JWTSecret      string   `mapstructure:"jwt_secret" yaml:"jwt_secret" json:"jwt_secret"`
	AllowedOrigins []string `mapstructure:"allowed_origins" yaml:"allowed_origins" json:"allowed_origins"`
}

// LoggingConfig represents logging configuration.
type LoggingConfig struct {
	Level  string `mapstructure:"level" yaml:"level" json:"level"`
	Format string `mapstructure:"format" yaml:"format" json:"format"`
	Output string `mapstructure:"output" yaml:"output" json:"output"`
}

// Validation errors
var (
	ErrServerPortInvalid      = errors.New("server port must be between 1 and 65535")
	ErrLLMProviderRequired    = errors.New("LLM provider is required")
	ErrLLMProviderInvalid     = errors.New("LLM provider must be one of: openai, anthropic, azure")
	ErrLLMAPIKeyRequired      = errors.New("LLM API key is required")
	ErrLLMModelRequired       = errors.New("LLM model is required")
	ErrDatabaseHostRequired   = errors.New("database host is required")
	ErrDatabasePortInvalid    = errors.New("database port must be between 1 and 65535")
	ErrDatabaseNameRequired   = errors.New("database name is required")
	ErrDatabaseUserRequired   = errors.New("database user is required")
	ErrRedisHostRequired      = errors.New("redis host is required")
	ErrRedisPortInvalid       = errors.New("redis port must be between 1 and 65535")
	ErrMCPServerIDRequired    = errors.New("MCP server ID is required")
	ErrMCPServerNameRequired  = errors.New("MCP server name is required")
	ErrMCPTransportInvalid    = errors.New("MCP transport must be one of: stdio, http")
	ErrMCPStdioCommandRequired = errors.New("MCP stdio transport requires command")
	ErrMCPHTTPURLRequired     = errors.New("MCP http transport requires URL")
	ErrTLSCertFileRequired    = errors.New("TLS cert file is required when TLS is enabled")
	ErrTLSKeyFileRequired     = errors.New("TLS key file is required when TLS is enabled")
	ErrLoggingLevelInvalid    = errors.New("logging level must be one of: debug, info, warn, error")
	ErrLoggingFormatInvalid   = errors.New("logging format must be one of: json, text")
)

// validLLMProviders contains the list of valid LLM providers.
var validLLMProviders = map[string]bool{
	"openai":    true,
	"anthropic": true,
	"azure":     true,
}

// validTransports contains the list of valid MCP transports.
var validTransports = map[string]bool{
	"stdio": true,
	"http":  true,
}

// validLogLevels contains the list of valid log levels.
var validLogLevels = map[string]bool{
	"debug": true,
	"info":  true,
	"warn":  true,
	"error": true,
}

// validLogFormats contains the list of valid log formats.
var validLogFormats = map[string]bool{
	"json": true,
	"text": true,
}

// Load loads configuration from file and environment variables.
// It searches for config file in the following order:
// 1. Path specified by configPath parameter (if not empty)
// 2. Current directory
// 3. /etc/opsgenius/
// Environment variables override file configuration.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Configure viper
	v.SetConfigName("config")
	v.SetConfigType("yaml")

	// Add config paths
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.AddConfigPath(".")
		v.AddConfigPath("/etc/opsgenius/")
	}

	// Enable environment variable override
	v.SetEnvPrefix("OPSGENIUS")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found is acceptable if env vars are set
	}

	// Unmarshal config
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Expand environment variables in sensitive fields
	cfg.expandEnvVars()

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// LoadFromReader loads configuration from an io.Reader (useful for testing).
func LoadFromReader(configType string, data []byte) (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	v.SetConfigType(configType)

	if err := v.ReadConfig(strings.NewReader(string(data))); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Expand environment variables in sensitive fields
	cfg.expandEnvVars()

	// Validate config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values.
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.websocket_path", "/ws")
	v.SetDefault("server.tls.enabled", false)

	// LLM defaults
	v.SetDefault("llm.temperature", 0.7)
	v.SetDefault("llm.max_tokens", 2000)

	// Database defaults
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.max_connections", 100)

	// Redis defaults
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.pool_size", 50)

	// Security defaults
	v.SetDefault("security.auth_enabled", true)

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")
}

// expandEnvVars expands environment variables in sensitive configuration fields.
func (c *Config) expandEnvVars() {
	c.LLM.APIKey = expandEnvVar(c.LLM.APIKey)
	c.Database.Password = expandEnvVar(c.Database.Password)
	c.Redis.Password = expandEnvVar(c.Redis.Password)
	c.Security.JWTSecret = expandEnvVar(c.Security.JWTSecret)

	for i := range c.MCPServers {
		if c.MCPServers[i].Auth != nil {
			c.MCPServers[i].Auth.Token = expandEnvVar(c.MCPServers[i].Auth.Token)
		}
	}
}

// expandEnvVar expands ${VAR} or $VAR patterns in a string.
func expandEnvVar(s string) string {
	return os.ExpandEnv(s)
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if err := c.validateServer(); err != nil {
		return err
	}
	if err := c.validateLLM(); err != nil {
		return err
	}
	if err := c.validateMCPServers(); err != nil {
		return err
	}
	if err := c.validateDatabase(); err != nil {
		return err
	}
	if err := c.validateRedis(); err != nil {
		return err
	}
	if err := c.validateLogging(); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateServer() error {
	if c.Server.Port < 1 || c.Server.Port > 65535 {
		return ErrServerPortInvalid
	}
	if c.Server.TLS.Enabled {
		if c.Server.TLS.CertFile == "" {
			return ErrTLSCertFileRequired
		}
		if c.Server.TLS.KeyFile == "" {
			return ErrTLSKeyFileRequired
		}
	}
	return nil
}

func (c *Config) validateLLM() error {
	if c.LLM.Provider == "" {
		return ErrLLMProviderRequired
	}
	if !validLLMProviders[c.LLM.Provider] {
		return ErrLLMProviderInvalid
	}
	if c.LLM.APIKey == "" {
		return ErrLLMAPIKeyRequired
	}
	if c.LLM.Model == "" {
		return ErrLLMModelRequired
	}
	return nil
}

func (c *Config) validateMCPServers() error {
	for i, server := range c.MCPServers {
		if server.ID == "" {
			return fmt.Errorf("MCP server %d: %w", i, ErrMCPServerIDRequired)
		}
		if server.Name == "" {
			return fmt.Errorf("MCP server %s: %w", server.ID, ErrMCPServerNameRequired)
		}
		if !validTransports[server.Transport] {
			return fmt.Errorf("MCP server %s: %w", server.ID, ErrMCPTransportInvalid)
		}
		if server.Transport == "stdio" && server.Command == "" {
			return fmt.Errorf("MCP server %s: %w", server.ID, ErrMCPStdioCommandRequired)
		}
		if server.Transport == "http" && server.URL == "" {
			return fmt.Errorf("MCP server %s: %w", server.ID, ErrMCPHTTPURLRequired)
		}
	}
	return nil
}

func (c *Config) validateDatabase() error {
	if c.Database.Host == "" {
		return ErrDatabaseHostRequired
	}
	if c.Database.Port < 1 || c.Database.Port > 65535 {
		return ErrDatabasePortInvalid
	}
	if c.Database.Database == "" {
		return ErrDatabaseNameRequired
	}
	if c.Database.User == "" {
		return ErrDatabaseUserRequired
	}
	return nil
}

func (c *Config) validateRedis() error {
	if c.Redis.Host == "" {
		return ErrRedisHostRequired
	}
	if c.Redis.Port < 1 || c.Redis.Port > 65535 {
		return ErrRedisPortInvalid
	}
	return nil
}

func (c *Config) validateLogging() error {
	if c.Logging.Level != "" && !validLogLevels[c.Logging.Level] {
		return ErrLoggingLevelInvalid
	}
	if c.Logging.Format != "" && !validLogFormats[c.Logging.Format] {
		return ErrLoggingFormatInvalid
	}
	return nil
}
