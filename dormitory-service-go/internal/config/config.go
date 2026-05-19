package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all application configuration.
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Kafka    KafkaConfig    `mapstructure:"kafka"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Log      LogConfig      `mapstructure:"log"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port int `mapstructure:"port"`
}

// DatabaseConfig holds MariaDB connection settings.
type DatabaseConfig struct {
	DSN         string `mapstructure:"dsn"`
	Driver      string `mapstructure:"driver"`
	MaxOpenConn int    `mapstructure:"max_open_conns"`
	MaxIdleConn int    `mapstructure:"max_idle_conns"`
}

// RedisConfig holds Redis connection settings.
type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	DB       int    `mapstructure:"db"`
	Password string `mapstructure:"password"`
}

// KafkaConfig holds Kafka connection settings.
type KafkaConfig struct {
	Brokers       []string `mapstructure:"brokers"`
	EventTopic    string   `mapstructure:"event_topic"`
	AlertTopic    string   `mapstructure:"alert_topic"`
	GroupID       string   `mapstructure:"group_id"`
	MaxPollRecord int      `mapstructure:"max_poll_records"`
}

// JWTConfig holds JWT settings.
type JWTConfig struct {
	Secret           string `mapstructure:"secret"`
	ExpirationHours  int    `mapstructure:"expiration_hours"`
}

// LogConfig holds logging settings.
type LogConfig struct {
	Level string `mapstructure:"level"`
}

// Address returns the Redis address string (host:port).
func (r RedisConfig) Address() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// Load reads configuration from config.yaml and environment variables.
// It follows the precedence: defaults < config.yaml < env vars.
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// --- Defaults ---
	v.SetDefault("server.port", 8081)
	v.SetDefault("database.driver", "mysql")
	v.SetDefault("database.max_open_conns", 25)
	v.SetDefault("database.max_idle_conns", 10)
	v.SetDefault("redis.host", "127.0.0.1")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)
	v.SetDefault("redis.password", "")
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.event_topic", "t_dorm_event")
	v.SetDefault("kafka.alert_topic", "t_dorm_alert")
	v.SetDefault("kafka.group_id", "dormitory-service-group")
	v.SetDefault("kafka.max_poll_records", 500)
	v.SetDefault("jwt.secret", "your-256-bit-secret")
	v.SetDefault("jwt.expiration_hours", 24)
	v.SetDefault("log.level", "info")

	// --- Config file ---
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("read config: %w", err)
		}
		// config file is optional; fall through to defaults + env
	}

	// --- Environment variable mapping ---
	// Support both our own prefixed vars AND Spring Boot compatible vars
	v.SetEnvPrefix("")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Map Spring Boot compatible env vars to our config keys
	envOverrides := map[string]string{
		"SPRING_DATASOURCE_URL":      "database.dsn",
		"SPRING_DATASOURCE_USERNAME": "", // embedded in dsn, not separate
		"SPRING_DATASOURCE_PASSWORD": "", // embedded in dsn, not separate
		"KAFKA_BOOTSTRAP_SERVERS":    "kafka.brokers",
		"SPRING_DATA_REDIS_HOST":     "redis.host",
		"SPRING_DATA_REDIS_PORT":     "redis.port",
		"SPRING_DATA_REDIS_PASSWORD": "redis.password",
	}

	for envKey, configKey := range envOverrides {
		if configKey == "" {
			continue
		}
		if val, ok := os.LookupEnv(envKey); ok {
			v.Set(configKey, val)
		}
	}

	// Handle SPRING_DATASOURCE_URL -> DSN override specially
	if dsn, ok := os.LookupEnv("SPRING_DATASOURCE_URL"); ok {
		// If the DSN contains user/password from Spring-style JDBC URL,
		// we need to convert it. Spring JDBC URLs look like:
		// jdbc:mariadb://host:port/db?params
		// Our DSN looks like: user:pass@tcp(host:port)/db?params
		dsn = convertSpringDSN(dsn)
		user := os.Getenv("SPRING_DATASOURCE_USERNAME")
		pass := os.Getenv("SPRING_DATASOURCE_PASSWORD")
		if user != "" || pass != "" {
			dsn = embedCredentials(dsn, user, pass)
		}
		v.Set("database.dsn", dsn)
	}

	// Override kafka.brokers from comma-separated env var
	if brokers, ok := os.LookupEnv("KAFKA_BOOTSTRAP_SERVERS"); ok {
		v.Set("kafka.brokers", strings.Split(brokers, ","))
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

// convertSpringDSN converts a Spring Boot JDBC URL to a Go sqlx DSN.
// Example:
//
//	jdbc:mariadb://mariadb:3306/dormitory
//	→ root:password@tcp(mariadb:3306)/dormitory?charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai
func convertSpringDSN(jdbcURL string) string {
	// Strip jdbc:mariadb:// prefix
	trimmed := strings.TrimPrefix(jdbcURL, "jdbc:mariadb://")
	trimmed = strings.TrimPrefix(trimmed, "jdbc:mysql://")

	// Add tcp() wrapper and extra params
	if !strings.Contains(trimmed, "?") {
		trimmed += "?"
	} else {
		trimmed += "&"
	}
	trimmed += "charset=utf8mb4&parseTime=True&loc=Asia%2FShanghai"

	return trimmed
}

// embedCredentials inserts user:password@ prefix into a DSN.
func embedCredentials(dsn, username, password string) string {
	// DSN format: [user[:pass]@tcp(host:port)/db][?params]
	if strings.HasPrefix(dsn, "tcp(") {
		creds := ""
		if username != "" {
			creds = username
		}
		if password != "" {
			creds += ":" + password
		}
		if creds != "" {
			return creds + "@" + dsn
		}
	}
	return dsn
}
