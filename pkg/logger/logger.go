package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// New creates a new zerolog logger with structured JSON output.
func New(serviceName string) zerolog.Logger {
	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger().
		Level(zerolog.InfoLevel)
}

// NewWithLevel creates a logger with a specific log level.
func NewWithLevel(serviceName string, level string) zerolog.Logger {
	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Str("service", serviceName).
		Logger().
		Level(lvl)
}

func init() {
	zerolog.TimeFieldFormat = time.RFC3339
}
