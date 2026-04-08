package config

import (
	"fmt"
	"os"
	"strconv"
)

// DatabaseConfig holds PostgreSQL connection parameters.
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
	SSLMode  string
}

// KafkaConfig holds Kafka connection parameters.
type KafkaConfig struct {
	Brokers []string
}

// TracingConfig holds OpenTelemetry configuration.
type TracingConfig struct {
	Endpoint    string
	ServiceName string
}

// ServiceConfig holds common service configuration.
type ServiceConfig struct {
	Name     string
	GRPCPort int
	HTTPPort int
}

// LoadDatabaseConfig reads database config from environment variables.
func LoadDatabaseConfig(dbName string) DatabaseConfig {
	return DatabaseConfig{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnvAsInt("DB_PORT", 5432),
		User:     getEnv("DB_USER", "orderflow"),
		Password: getEnv("DB_PASSWORD", "orderflow"),
		Name:     getEnv("DB_NAME", dbName),
		SSLMode:  getEnv("DB_SSL_MODE", "disable"),
	}
}

// DSN returns the PostgreSQL connection string.
func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode,
	)
}

// LoadKafkaConfig reads Kafka config from environment variables.
func LoadKafkaConfig() KafkaConfig {
	brokers := getEnv("KAFKA_BROKERS", "localhost:9092")
	return KafkaConfig{
		Brokers: splitAndTrim(brokers),
	}
}

// LoadTracingConfig reads tracing config from environment variables.
func LoadTracingConfig(serviceName string) TracingConfig {
	return TracingConfig{
		Endpoint:    getEnv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://localhost:4318"),
		ServiceName: serviceName,
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func splitAndTrim(s string) []string {
	parts := []string{}
	for _, part := range splitString(s, ",") {
		trimmed := trimSpace(part)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func splitString(s, sep string) []string {
	result := []string{}
	current := ""
	for _, c := range s {
		if string(c) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}

func trimSpace(s string) string {
	start, end := 0, len(s)-1
	for start <= end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end >= start && (s[end] == ' ' || s[end] == '\t') {
		end--
	}
	return s[start : end+1]
}
