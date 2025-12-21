package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validYAMLConfig is a valid configuration for testing.
const validYAMLConfig = `
server:
  host: "0.0.0.0"
  port: 8080
  websocket_path: "/ws"
  tls:
    enabled: false

llm:
  provider: "openai"
  api_key: "test-api-key"
  model: "gpt-4"
  temperature: 0.7
  max_tokens: 2000

mcp_servers:
  - id: "log-mcp"
    name: "Log MCP Server"
    transport: "stdio"
    command: "python"
    args: ["-m", "log_mcp_server"]

database:
  host: "localhost"
  port: 5432
  database: "opsgenius"
  user: "postgres"
  password: "secret"
  max_connections: 100

redis:
  host: "localhost"
  port: 6379
  password: ""
  db: 0
  pool_size: 50

security:
  auth_enabled: true
  jwt_secret: "test-secret"
  allowed_origins: ["http://localhost:3000"]

logging:
  level: "info"
  format: "json"
  output: "stdout"
`

// validJSONConfig is a valid JSON configuration for testing.
const validJSONConfig = `{
  "server": {
    "host": "0.0.0.0",
    "port": 8080,
    "websocket_path": "/ws",
    "tls": {"enabled": false}
  },
  "llm": {
    "provider": "anthropic",
    "api_key": "test-key",
    "model": "claude-3",
    "temperature": 0.5,
    "max_tokens": 1000
  },
  "mcp_servers": [
    {
      "id": "test-mcp",
      "name": "Test MCP",
      "transport": "http",
      "url": "http://localhost:9090"
    }
  ],
  "database": {
    "host": "db.example.com",
    "port": 5432,
    "database": "testdb",
    "user": "testuser",
    "password": "testpass"
  },
  "redis": {
    "host": "redis.example.com",
    "port": 6379
  },
  "logging": {
    "level": "debug",
    "format": "text"
  }
}`

func TestLoadFromReader_ValidYAML(t *testing.T) {
	cfg, err := LoadFromReader("yaml", []byte(validYAMLConfig))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	// Verify server config
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "/ws", cfg.Server.WebSocketPath)
	assert.False(t, cfg.Server.TLS.Enabled)

	// Verify LLM config
	assert.Equal(t, "openai", cfg.LLM.Provider)
	assert.Equal(t, "test-api-key", cfg.LLM.APIKey)
	assert.Equal(t, "gpt-4", cfg.LLM.Model)
	assert.Equal(t, 0.7, cfg.LLM.Temperature)
	assert.Equal(t, 2000, cfg.LLM.MaxTokens)

	// Verify MCP servers
	require.Len(t, cfg.MCPServers, 1)
	assert.Equal(t, "log-mcp", cfg.MCPServers[0].ID)
	assert.Equal(t, "Log MCP Server", cfg.MCPServers[0].Name)
	assert.Equal(t, "stdio", cfg.MCPServers[0].Transport)
	assert.Equal(t, "python", cfg.MCPServers[0].Command)

	// Verify database config
	assert.Equal(t, "localhost", cfg.Database.Host)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "opsgenius", cfg.Database.Database)
	assert.Equal(t, "postgres", cfg.Database.User)

	// Verify redis config
	assert.Equal(t, "localhost", cfg.Redis.Host)
	assert.Equal(t, 6379, cfg.Redis.Port)

	// Verify logging config
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}

func TestLoadFromReader_ValidJSON(t *testing.T) {
	cfg, err := LoadFromReader("json", []byte(validJSONConfig))
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "anthropic", cfg.LLM.Provider)
	assert.Equal(t, "claude-3", cfg.LLM.Model)
	assert.Equal(t, "http", cfg.MCPServers[0].Transport)
	assert.Equal(t, "http://localhost:9090", cfg.MCPServers[0].URL)
}

func TestLoadFromReader_InvalidYAML(t *testing.T) {
	invalidYAML := `
server:
  port: "not a number"  # This should cause unmarshal error
  host: [invalid array]
`
	_, err := LoadFromReader("yaml", []byte(invalidYAML))
	assert.Error(t, err)
}

func TestLoadFromReader_MalformedYAML(t *testing.T) {
	malformedYAML := `
server:
  host: "localhost"
    port: 8080  # Invalid indentation
`
	_, err := LoadFromReader("yaml", []byte(malformedYAML))
	assert.Error(t, err)
}

func TestValidate_MissingLLMProvider(t *testing.T) {
	configYAML := `
server:
  port: 8080
llm:
  api_key: "test"
  model: "gpt-4"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
redis:
  host: "localhost"
  port: 6379
`
	_, err := LoadFromReader("yaml", []byte(configYAML))
	assert.ErrorIs(t, err, ErrLLMProviderRequired)
}

func TestValidate_InvalidLLMProvider(t *testing.T) {
	configYAML := `
server:
  port: 8080
llm:
  provider: "invalid-provider"
  api_key: "test"
  model: "gpt-4"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
redis:
  host: "localhost"
  port: 6379
`
	_, err := LoadFromReader("yaml", []byte(configYAML))
	assert.ErrorIs(t, err, ErrLLMProviderInvalid)
}

func TestValidate_MissingLLMAPIKey(t *testing.T) {
	configYAML := `
server:
  port: 8080
llm:
  provider: "openai"
  model: "gpt-4"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
redis:
  host: "localhost"
  port: 6379
`
	_, err := LoadFromReader("yaml", []byte(configYAML))
	assert.ErrorIs(t, err, ErrLLMAPIKeyRequired)
}

func TestValidate_InvalidServerPort(t *testing.T) {
	configYAML := `
server:
  port: 70000
llm:
  provider: "openai"
  api_key: "test"
  model: "gpt-4"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
redis:
  host: "localhost"
  port: 6379
`
	_, err := LoadFromReader("yaml", []byte(configYAML))
	assert.ErrorIs(t, err, ErrServerPortInvalid)
}

func TestValidate_TLSEnabledWithoutCert(t *testing.T) {
	configYAML := `
server:
  port: 8080
  tls:
    enabled: true
llm:
  provider: "openai"
  api_key: "test"
  model: "gpt-4"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
redis:
  host: "localhost"
  port: 6379
`
	_, err := LoadFromReader("yaml", []byte(configYAML))
	assert.ErrorIs(t, err, ErrTLSCertFileRequired)
}

func TestValidate_MCPServerInvalidTransport(t *testing.T) {
	configYAML := `
server:
  port: 8080
llm:
  provider: "openai"
  api_key: "test"
  model: "gpt-4"
mcp_servers:
  - id: "test"
    name: "Test"
    transport: "invalid"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
redis:
  host: "localhost"
  port: 6379
`
	_, err := LoadFromReader("yaml", []byte(configYAML))
	assert.ErrorIs(t, err, ErrMCPTransportInvalid)
}

func TestValidate_MCPStdioWithoutCommand(t *testing.T) {
	configYAML := `
server:
  port: 8080
llm:
  provider: "openai"
  api_key: "test"
  model: "gpt-4"
mcp_servers:
  - id: "test"
    name: "Test"
    transport: "stdio"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
redis:
  host: "localhost"
  port: 6379
`
	_, err := LoadFromReader("yaml", []byte(configYAML))
	assert.ErrorIs(t, err, ErrMCPStdioCommandRequired)
}

func TestValidate_MCPHTTPWithoutURL(t *testing.T) {
	configYAML := `
server:
  port: 8080
llm:
  provider: "openai"
  api_key: "test"
  model: "gpt-4"
mcp_servers:
  - id: "test"
    name: "Test"
    transport: "http"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
redis:
  host: "localhost"
  port: 6379
`
	_, err := LoadFromReader("yaml", []byte(configYAML))
	assert.ErrorIs(t, err, ErrMCPHTTPURLRequired)
}

func TestValidate_InvalidLoggingLevel(t *testing.T) {
	configYAML := `
server:
  port: 8080
llm:
  provider: "openai"
  api_key: "test"
  model: "gpt-4"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
redis:
  host: "localhost"
  port: 6379
logging:
  level: "invalid"
`
	_, err := LoadFromReader("yaml", []byte(configYAML))
	assert.ErrorIs(t, err, ErrLoggingLevelInvalid)
}

func TestExpandEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("TEST_API_KEY", "env-api-key")
	os.Setenv("TEST_DB_PASSWORD", "env-db-password")
	defer os.Unsetenv("TEST_API_KEY")
	defer os.Unsetenv("TEST_DB_PASSWORD")

	configYAML := `
server:
  port: 8080
llm:
  provider: "openai"
  api_key: "${TEST_API_KEY}"
  model: "gpt-4"
database:
  host: "localhost"
  port: 5432
  database: "test"
  user: "test"
  password: "${TEST_DB_PASSWORD}"
redis:
  host: "localhost"
  port: 6379
`
	cfg, err := LoadFromReader("yaml", []byte(configYAML))
	require.NoError(t, err)

	assert.Equal(t, "env-api-key", cfg.LLM.APIKey)
	assert.Equal(t, "env-db-password", cfg.Database.Password)
}

func TestDefaults(t *testing.T) {
	// Minimal config that relies on defaults
	configYAML := `
llm:
  provider: "openai"
  api_key: "test"
  model: "gpt-4"
database:
  host: "localhost"
  database: "test"
  user: "test"
redis:
  host: "localhost"
`
	cfg, err := LoadFromReader("yaml", []byte(configYAML))
	require.NoError(t, err)

	// Check defaults are applied
	assert.Equal(t, "0.0.0.0", cfg.Server.Host)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.Equal(t, "/ws", cfg.Server.WebSocketPath)
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, 100, cfg.Database.MaxConnections)
	assert.Equal(t, 6379, cfg.Redis.Port)
	assert.Equal(t, 50, cfg.Redis.PoolSize)
	assert.Equal(t, "info", cfg.Logging.Level)
	assert.Equal(t, "json", cfg.Logging.Format)
}
