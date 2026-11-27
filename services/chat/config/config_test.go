package config

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type validationTestCase struct {
	name    string
	input   string
	wantErr bool
}

func runValidationTests(
	t *testing.T,
	tests []validationTestCase,
	validate func(string) error,
) {
	t.Helper()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.input)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidatePort(t *testing.T) {
	tests := []validationTestCase{
		{
			name: "valid port", input: "8080", wantErr: false,
		},
		{
			name: "minimum port", input: "1", wantErr: false,
		},
		{
			name: "maximum port", input: "65535", wantErr: false,
		},
		{
			name: "zero port", input: "0", wantErr: true,
		},
		{
			name: "negative port", input: "-1", wantErr: true,
		},
		{
			name: "above maximum", input: "65536", wantErr: true,
		},
		{
			name: "empty port", input: "", wantErr: true,
		},
		{
			name: "whitespace only", input: " ", wantErr: true,
		},
		{
			name: "non-numeric port", input: "abc", wantErr: true,
		},
		{
			name: "mixed characters", input: "abc80", wantErr: true,
		},
		{
			name: "decimal port", input: "80.5", wantErr: true,
		},
		{
			name: "leading whitespace", input: " 8080", wantErr: true,
		},
		{
			name: "trailing whitespace", input: "8080 ", wantErr: true,
		},
		{
			name: "integer overflow", input: "99999999999999999999999999999999999", wantErr: true,
		},
		{
			name: "port with plus sign", input: "+8080", wantErr: true,
		},
	}

	runValidationTests(t, tests, validatePort)
}

func TestValidateAddress(t *testing.T) {
	tests := []validationTestCase{
		{
			name: "valid localhost", input: "localhost:6379", wantErr: false,
		},
		{
			name: "valid IPv4", input: "127.0.0.1:6379", wantErr: false,
		},
		{
			name: "valid IPv6", input: "[::1]:6379", wantErr: false,
		},
		{
			name: "valid hostname", input: "redis:6379", wantErr: false,
		},
		{
			name: "valid domain", input: "redis.example.com:6379", wantErr: false,
		},
		{
			name: "empty address", input: "", wantErr: true,
		},
		{
			name: "missing port", input: "localhost", wantErr: true,
		},
		{
			name: "empty port", input: "localhost:", wantErr: true,
		},
		{
			name: "empty host", input: ":6379", wantErr: true,
		},
		{
			name: "whitespace host", input: " :6379", wantErr: true,
		},
		{
			name: "host with whitespace", input: "local host:6379", wantErr: true,
		},
		{
			name: "invalid port", input: "localhost:abc", wantErr: true,
		},
		{
			name: "zero port", input: "localhost:0", wantErr: true,
		},
		{
			name: "port above maximum", input: "localhost:65536", wantErr: true,
		},
		{
			name: "IPv6 without brackets", input: "::1:6379", wantErr: true,
		},
	}

	runValidationTests(t, tests, validateAddress)
}

func TestValidateTopic(t *testing.T) {
	tests := []validationTestCase{
		{
			name: "valid topic", input: "websocket-events", wantErr: false,
		},
		{
			name: "uppercase letters", input: "WebSocketEvents", wantErr: false,
		},
		{
			name: "topic with digits", input: "events123", wantErr: false,
		},
		{
			name: "topic with underscore", input: "websocket_events", wantErr: false,
		},
		{
			name: "topic with dot", input: "websocket.events", wantErr: false,
		},
		{
			name: "all allowed characters", input: "Events_123.test-v1", wantErr: false,
		},
		{
			name: "minimum length", input: "a", wantErr: false,
		},
		{
			name: "maximum length", input: strings.Repeat("a", 249), wantErr: false,
		},
		{
			name: "above maximum length", input: strings.Repeat("a", 250), wantErr: true,
		},
		{
			name: "empty topic", input: "", wantErr: true,
		},
		{
			name: "single dot", input: ".", wantErr: true,
		},
		{
			name: "double dot", input: "..", wantErr: true,
		},
		{
			name: "whitespace only", input: " ", wantErr: true,
		},
		{
			name: "topic with whitespace", input: "websocket events", wantErr: true,
		},
		{
			name: "topic with slash", input: "websocket/events", wantErr: true,
		},
		{
			name: "topic with special characters", input: "events@123", wantErr: true,
		},
		{
			name: "non-ASCII characters", input: "события", wantErr: true,
		},
		{
			name: "topic with newline", input: "events\n", wantErr: true,
		},
	}

	runValidationTests(t, tests, validateTopic)
}

func TestParseBrokers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name: "single broker", input: "localhost:9092", want: []string{"localhost:9092"},
		},
		{
			name: "multiple brokers", input: "kafka1:9092,kafka2:9092", want: []string{"kafka1:9092", "kafka2:9092"},
		},
		{
			name: "whitespace around brokers", input: " kafka1:9092 , kafka2:9092 ", want: []string{"kafka1:9092", "kafka2:9092"},
		},
		{
			name: "empty", input: "", want: []string{""},
		},
		{
			name: "whitespace only", input: " ", want: []string{""},
		},
		{
			name: "trailing comma", input: "kafka1:9092,", want: []string{"kafka1:9092", ""},
		},
		{
			name: "leading comma", input: ",kafka2:9092", want: []string{"", "kafka2:9092"},
		},
		{
			name: "double comma", input: "kafka1:9092,,kafka2:9092", want: []string{"kafka1:9092", "", "kafka2:9092"},
		},
		{
			name: "only comma", input: ",", want: []string{"", ""},
		},
		{
			name: "IPv6 broker", input: "[::1]:9092", want: []string{"[::1]:9092"},
		},
		{
			name: "duplicate broker", input: "kafka:9092,kafka:9092", want: []string{"kafka:9092", "kafka:9092"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBrokers(tt.input)
			require.Equal(t, tt.want, result)
		})
	}
}

func TestValidateBrokers(t *testing.T) {
	tests := []struct {
		name    string
		brokers []string
		wantErr bool
	}{
		{
			name: "single broker", brokers: []string{"localhost:9092"}, wantErr: false,
		},
		{
			name: "multiple brokers", brokers: []string{"kafka1:9092", "kafka2:9092"}, wantErr: false,
		},
		{
			name: "IPv6 broker", brokers: []string{"[::1]:9092"}, wantErr: false,
		},
		{
			name: "nil brokers", brokers: nil, wantErr: true,
		},
		{
			name: "empty slice", brokers: []string{}, wantErr: true,
		},
		{
			name: "empty broker", brokers: []string{""}, wantErr: true,
		},
		{
			name: "invalid single broker", brokers: []string{"localhost"}, wantErr: true,
		},
		{
			name: "invalid first broker", brokers: []string{"invalid", "kafka2:9092"}, wantErr: true,
		},
		{
			name: "invalid last broker", brokers: []string{"kafka1:9092", "invalid"}, wantErr: true,
		},
		{
			name: "empty broker between valid", brokers: []string{"kafka1:9092", "", "kafka2:9092"}, wantErr: true,
		},
		{
			name: "duplicate brokers", brokers: []string{"kafka1:9092", "kafka1:9092"}, wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBrokers(tt.brokers)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateDatabaseURL(t *testing.T) {
	tests := []validationTestCase{
		{
			name: "valid postgres URL", input: "postgres://user:pass@localhost:5432/chat_db", wantErr: false,
		},
		{
			name: "valid postgresql URL", input: "postgresql://user:pass@localhost:5432/chat_db", wantErr: false,
		},
		{
			name: "without password", input: "postgres://user@localhost:5432/chat_db", wantErr: false,
		},
		{
			name: "without explicit port", input: "postgres://user:pass@localhost/chat_db", wantErr: false,
		},
		{
			name: "with SSL mode", input: "postgres://user:pass@localhost:5432/chat_db?sslmode=disable", wantErr: false,
		},
		{
			name: "valid IPv6", input: "postgres://user:pass@[::1]:5432/chat_db", wantErr: false,
		},
		{
			name: "empty URL", input: "", wantErr: true,
		},
		{
			name: "whitespace only", input: " ", wantErr: true,
		},
		{
			name: "malformed URL", input: "postgres://%zz", wantErr: true,
		},
		{
			name: "invalid scheme", input: "mysql://user:pass@localhost:3306/chat_db", wantErr: true,
		},
		{
			name: "missing scheme", input: "localhost:5432/chat_db", wantErr: true,
		},
		{
			name: "missing user", input: "postgres://localhost:5432/chat_db", wantErr: true,
		},
		{
			name: "empty username", input: "postgres://:pass@localhost:5432/chat_db", wantErr: true,
		},
		{
			name: "whitespace username", input: "postgres://%20@localhost:5432/chat_db", wantErr: true,
		},
		{
			name: "missing host", input: "postgres://user:pass@:5432/chat_db", wantErr: true,
		},
		{
			name: "missing database", input: "postgres://user:pass@localhost:5432", wantErr: true,
		},
		{
			name: "empty database path", input: "postgres://user:pass@localhost:5432/", wantErr: true,
		},
		{
			name: "whitespace database", input: "postgres://user:pass@localhost:5432/%20", wantErr: true,
		},
		{
			name: "invalid port", input: "postgres://user:pass@localhost:65536/chat_db", wantErr: true,
		},
		{
			name: "invalid SSL mode", input: "postgres://user:pass@localhost:5432/chat_db?sslmode=invalid", wantErr: true,
		},
	}

	runValidationTests(t, tests, validateDatabaseURL)
}

func validConfig() Config {
	return Config{
		HTTPPort:     "8080",
		DatabaseURL:  "postgres://user:pass@localhost:5432/chat_db?sslmode=disable",
		RedisAddr:    "localhost:6379",
		KafkaBrokers: []string{"localhost:9092"},
		KafkaWSTopic: "websocket-events",
	}
}

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name        string
		modify      func(*Config)
		wantErr     bool
		errContains string
	}{
		{
			name:        "valid config",
			modify:      nil,
			wantErr:     false,
			errContains: "",
		},
		{
			name: "invalid HTTP port",
			modify: func(cfg *Config) {
				cfg.HTTPPort = "0"
			},
			wantErr:     true,
			errContains: "HTTP_PORT:",
		},
		{
			name: "invalid database URL",
			modify: func(cfg *Config) {
				cfg.DatabaseURL = ""
			},
			wantErr:     true,
			errContains: "DATABASE_URL:",
		},
		{
			name: "invalid Redis address",
			modify: func(cfg *Config) {
				cfg.RedisAddr = ""
			},
			wantErr:     true,
			errContains: "REDIS_ADDR:",
		},
		{
			name: "invalid Kafka brokers",
			modify: func(cfg *Config) {
				cfg.KafkaBrokers = nil
			},
			wantErr:     true,
			errContains: "KAFKA_BROKERS:",
		},
		{
			name: "invalid Kafka topic",
			modify: func(cfg *Config) {
				cfg.KafkaWSTopic = "."
			},
			wantErr:     true,
			errContains: "KAFKA_WS_TOPIC:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()

			if tt.modify != nil {
				tt.modify(&cfg)
			}

			err := cfg.Validate()
			if tt.wantErr {
				require.ErrorContains(t, err, tt.errContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	const key = "TEST_CONFIG_PORT"

	tests := []struct {
		name     string
		value    string
		exists   bool
		fallback string
		want     string
	}{
		{
			name: "existing variable", value: "9090", exists: true, fallback: "8080", want: "9090",
		},
		{
			name: "missing variable", value: "", exists: false, fallback: "8080", want: "8080",
		},
		{
			name: "empty variable", value: "", exists: true, fallback: "8080", want: "",
		},
		{
			name: "whitespace variable", value: " ", exists: true, fallback: "8080", want: " ",
		},
		{
			name: "missing variable with empty fallback", value: "", exists: false, fallback: "", want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.exists {
				t.Setenv(key, tt.value)
			} else {
				t.Setenv(key, "")
				require.NoError(t, os.Unsetenv(key))
			}

			result := getEnv(key, tt.fallback)
			require.Equal(t, tt.want, result)
		})
	}
}

func setValidEnv(t *testing.T) {
	t.Helper()

	t.Setenv("HTTP_PORT", "9090")
	t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/chat_db?sslmode=disable")
	t.Setenv("REDIS_ADDR", "localhost:6379")
	t.Setenv("KAFKA_BROKERS", "kafka1:9092, kafka2:9092")
	t.Setenv("KAFKA_WS_TOPIC", "test-events")
}

func TestLoad_Success(t *testing.T) {
	setValidEnv(t)

	cfg, err := Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "9090", cfg.HTTPPort)
	require.Equal(t, "postgres://user:pass@localhost:5432/chat_db?sslmode=disable", cfg.DatabaseURL)
	require.Equal(t, "localhost:6379", cfg.RedisAddr)
	require.Equal(t, []string{"kafka1:9092", "kafka2:9092"}, cfg.KafkaBrokers)
	require.Equal(t, "test-events", cfg.KafkaWSTopic)
}

func TestLoad_Defaults(t *testing.T) {
	setValidEnv(t)

	require.NoError(t, os.Unsetenv("HTTP_PORT"))
	require.NoError(t, os.Unsetenv("KAFKA_WS_TOPIC"))

	cfg, err := Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)

	require.Equal(t, "8080", cfg.HTTPPort)
	require.Equal(t, "websocket-events", cfg.KafkaWSTopic)
}

func TestLoad_InvalidConfig(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{
			name: "invalid HTTP port", key: "HTTP_PORT", value: "0",
		},
		{
			name: "invalid database URL", key: "DATABASE_URL", value: "",
		},
		{
			name: "invalid Redis address", key: "REDIS_ADDR", value: "",
		},
		{
			name: "invalid Kafka brokers", key: "KAFKA_BROKERS", value: "",
		},
		{
			name: "invalid Kafka topic", key: "KAFKA_WS_TOPIC", value: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnv(t)

			t.Setenv(tt.key, tt.value)

			cfg, err := Load()

			require.ErrorContains(t, err, tt.key+":")
			require.Nil(t, cfg)
		})
	}
}
