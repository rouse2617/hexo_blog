package config

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/leanovate/gopter"
	"github.com/leanovate/gopter/gen"
	"github.com/leanovate/gopter/prop"
	"gopkg.in/yaml.v3"
)

// Feature: ops-genius-backend, Property 31: 配置文件解析正确性
// *对于任何*有效的配置文件（YAML/JSON），应该被正确解析为配置对象，包含所有必需字段。
// **Validates: Requirements 11.1**

func TestProperty_ConfigParsingCorrectness(t *testing.T) {
	parameters := gopter.DefaultTestParameters()
	parameters.MinSuccessfulTests = 100

	properties := gopter.NewProperties(parameters)

	// Property: Valid config should round-trip through YAML serialization
	properties.Property("valid config round-trips through YAML", prop.ForAll(
		func(cfg Config) bool {
			// Serialize to YAML
			yamlData, err := yaml.Marshal(cfg)
			if err != nil {
				return false
			}

			// Parse back
			parsed, err := LoadFromReader("yaml", yamlData)
			if err != nil {
				return false
			}

			// Verify key fields are preserved
			return configsEqual(&cfg, parsed)
		},
		genValidConfig(),
	))

	// Property: Valid config should round-trip through JSON serialization
	properties.Property("valid config round-trips through JSON", prop.ForAll(
		func(cfg Config) bool {
			// Serialize to JSON
			jsonData, err := json.Marshal(cfg)
			if err != nil {
				return false
			}

			// Parse back
			parsed, err := LoadFromReader("json", jsonData)
			if err != nil {
				return false
			}

			// Verify key fields are preserved
			return configsEqual(&cfg, parsed)
		},
		genValidConfig(),
	))

	// Property: All required fields are present after parsing
	properties.Property("parsed config contains all required fields", prop.ForAll(
		func(cfg Config) bool {
			// Serialize to YAML
			yamlData, err := yaml.Marshal(cfg)
			if err != nil {
				return false
			}

			// Parse back
			parsed, err := LoadFromReader("yaml", yamlData)
			if err != nil {
				return false
			}

			// Verify required fields are present
			return parsed.LLM.Provider != "" &&
				parsed.LLM.APIKey != "" &&
				parsed.LLM.Model != "" &&
				parsed.Database.Host != "" &&
				parsed.Database.Database != "" &&
				parsed.Database.User != "" &&
				parsed.Redis.Host != ""
		},
		genValidConfig(),
	))

	properties.TestingRun(t)
}

// genAlphaNumString generates an alphanumeric string of given length range.
func genAlphaNumString(minLen, maxLen int) gopter.Gen {
	return gen.IntRange(minLen, maxLen).FlatMap(func(length interface{}) gopter.Gen {
		return gen.SliceOfN(length.(int), gen.AlphaNumChar()).Map(func(chars []rune) string {
			return string(chars)
		})
	}, reflect.TypeOf(""))
}

// genValidConfig generates a valid Config for property testing.
func genValidConfig() gopter.Gen {
	return gopter.CombineGens(
		genServerConfig(),
		genLLMConfig(),
		genMCPServers(),
		genDatabaseConfig(),
		genRedisConfig(),
		genSecurityConfig(),
		genLoggingConfig(),
	).Map(func(values []interface{}) Config {
		return Config{
			Server:     values[0].(ServerConfig),
			LLM:        values[1].(LLMConfig),
			MCPServers: values[2].([]MCPServerConfig),
			Database:   values[3].(DatabaseConfig),
			Redis:      values[4].(RedisConfig),
			Security:   values[5].(SecurityConfig),
			Logging:    values[6].(LoggingConfig),
		}
	})
}

func genServerConfig() gopter.Gen {
	return gopter.CombineGens(
		genAlphaNumString(1, 20),
		gen.IntRange(1, 65535),
		genAlphaNumString(1, 20),
	).Map(func(values []interface{}) ServerConfig {
		return ServerConfig{
			Host:          values[0].(string),
			Port:          values[1].(int),
			WebSocketPath: "/" + values[2].(string),
			TLS:           TLSConfig{Enabled: false},
		}
	})
}

func genLLMConfig() gopter.Gen {
	providers := []string{"openai", "anthropic", "azure"}
	return gopter.CombineGens(
		gen.OneConstOf(providers[0], providers[1], providers[2]),
		genAlphaNumString(10, 50),
		genAlphaNumString(3, 20),
		gen.Float64Range(0.0, 1.0),
		gen.IntRange(100, 4000),
	).Map(func(values []interface{}) LLMConfig {
		return LLMConfig{
			Provider:    values[0].(string),
			APIKey:      values[1].(string),
			Model:       values[2].(string),
			Temperature: values[3].(float64),
			MaxTokens:   values[4].(int),
		}
	})
}

func genMCPServers() gopter.Gen {
	// Generate 0-3 MCP servers directly without filtering
	return gen.IntRange(0, 3).FlatMap(func(count interface{}) gopter.Gen {
		n := count.(int)
		if n == 0 {
			return gen.Const([]MCPServerConfig{})
		}
		gens := make([]gopter.Gen, n)
		for i := 0; i < n; i++ {
			gens[i] = genMCPServerConfig()
		}
		return gopter.CombineGens(gens...).Map(func(values []interface{}) []MCPServerConfig {
			result := make([]MCPServerConfig, len(values))
			for i, v := range values {
				result[i] = v.(MCPServerConfig)
			}
			return result
		})
	}, reflect.TypeOf([]MCPServerConfig{}))
}

func genMCPServerConfig() gopter.Gen {
	return gopter.CombineGens(
		gen.Identifier(),
		genAlphaNumString(3, 30),
		gen.Bool(),
	).Map(func(values []interface{}) MCPServerConfig {
		id := values[0].(string)
		name := values[1].(string)
		useStdio := values[2].(bool)

		if useStdio {
			return MCPServerConfig{
				ID:        id,
				Name:      name,
				Transport: "stdio",
				Command:   "python",
				Args:      []string{"-m", "test_server"},
			}
		}
		return MCPServerConfig{
			ID:        id,
			Name:      name,
			Transport: "http",
			URL:       "http://localhost:9090",
		}
	})
}

func genDatabaseConfig() gopter.Gen {
	return gopter.CombineGens(
		genAlphaNumString(3, 30),
		gen.IntRange(1, 65535),
		gen.Identifier(),
		gen.Identifier(),
		genAlphaNumString(0, 30),
		gen.IntRange(1, 200),
	).Map(func(values []interface{}) DatabaseConfig {
		return DatabaseConfig{
			Host:           values[0].(string),
			Port:           values[1].(int),
			Database:       values[2].(string),
			User:           values[3].(string),
			Password:       values[4].(string),
			MaxConnections: values[5].(int),
		}
	})
}

func genRedisConfig() gopter.Gen {
	return gopter.CombineGens(
		genAlphaNumString(3, 30),
		gen.IntRange(1, 65535),
		genAlphaNumString(0, 30),
		gen.IntRange(0, 15),
		gen.IntRange(1, 100),
	).Map(func(values []interface{}) RedisConfig {
		return RedisConfig{
			Host:     values[0].(string),
			Port:     values[1].(int),
			Password: values[2].(string),
			DB:       values[3].(int),
			PoolSize: values[4].(int),
		}
	})
}

func genSecurityConfig() gopter.Gen {
	return gopter.CombineGens(
		gen.Bool(),
		genAlphaNumString(0, 50),
		gen.IntRange(0, 3).FlatMap(func(count interface{}) gopter.Gen {
			n := count.(int)
			if n == 0 {
				return gen.Const([]string{})
			}
			gens := make([]gopter.Gen, n)
			for i := 0; i < n; i++ {
				gens[i] = genAlphaNumString(3, 30)
			}
			return gopter.CombineGens(gens...).Map(func(values []interface{}) []string {
				result := make([]string, len(values))
				for i, v := range values {
					result[i] = v.(string)
				}
				return result
			})
		}, reflect.TypeOf([]string{})),
	).Map(func(values []interface{}) SecurityConfig {
		return SecurityConfig{
			AuthEnabled:    values[0].(bool),
			JWTSecret:      values[1].(string),
			AllowedOrigins: values[2].([]string),
		}
	})
}

func genLoggingConfig() gopter.Gen {
	levels := []string{"debug", "info", "warn", "error"}
	formats := []string{"json", "text"}
	return gopter.CombineGens(
		gen.OneConstOf(levels[0], levels[1], levels[2], levels[3]),
		gen.OneConstOf(formats[0], formats[1]),
		gen.OneConstOf("stdout", "stderr", "/var/log/app.log"),
	).Map(func(values []interface{}) LoggingConfig {
		return LoggingConfig{
			Level:  values[0].(string),
			Format: values[1].(string),
			Output: values[2].(string),
		}
	})
}

// configsEqual compares two configs for equality of key fields.
func configsEqual(a, b *Config) bool {
	// Compare server config
	if a.Server.Host != b.Server.Host ||
		a.Server.Port != b.Server.Port ||
		a.Server.WebSocketPath != b.Server.WebSocketPath {
		return false
	}

	// Compare LLM config
	if a.LLM.Provider != b.LLM.Provider ||
		a.LLM.APIKey != b.LLM.APIKey ||
		a.LLM.Model != b.LLM.Model {
		return false
	}

	// Compare database config
	if a.Database.Host != b.Database.Host ||
		a.Database.Port != b.Database.Port ||
		a.Database.Database != b.Database.Database ||
		a.Database.User != b.Database.User {
		return false
	}

	// Compare redis config
	if a.Redis.Host != b.Redis.Host ||
		a.Redis.Port != b.Redis.Port {
		return false
	}

	// Compare MCP servers count
	if len(a.MCPServers) != len(b.MCPServers) {
		return false
	}

	return true
}
