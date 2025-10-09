// SAGE - Secure Agent Guarantee Engine
// Copyright (C) 2025 SAGE-X-project
//
// This file is part of SAGE.
//
// SAGE is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// SAGE is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with SAGE. If not, see <https://www.gnu.org/licenses/>.

package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogLevels(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{DebugLevel, "DEBUG"},
		{InfoLevel, "INFO"},
		{WarnLevel, "WARN"},
		{ErrorLevel, "ERROR"},
		{FatalLevel, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.level.String())
		})
	}
}

func TestStructuredLogger(t *testing.T) {
	t.Run("LogLevelFiltering", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(&buf, WarnLevel)

		logger.Debug("debug message")
		assert.Empty(t, buf.String(), "Debug message should be filtered")

		logger.Info("info message")
		assert.Empty(t, buf.String(), "Info message should be filtered")

		logger.Warn("warn message")
		assert.NotEmpty(t, buf.String(), "Warn message should be logged")

		buf.Reset()
		logger.Error("error message")
		assert.NotEmpty(t, buf.String(), "Error message should be logged")
	})

	t.Run("StructuredFields", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(&buf, InfoLevel)

		logger.Info("test message",
			String("key1", "value1"),
			Int("key2", 42),
			Bool("key3", true),
			Error(errors.New("test error")),
			Duration("duration", 1000000000), // 1 second
		)

		var entry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &entry)
		require.NoError(t, err)

		assert.Equal(t, "INFO", entry["level"])
		assert.Equal(t, "test message", entry["message"])
		assert.Equal(t, "value1", entry["key1"])
		assert.Equal(t, float64(42), entry["key2"])
		assert.Equal(t, true, entry["key3"])
		assert.Equal(t, "test error", entry["error"])
		assert.Equal(t, "1s", entry["duration"])
		assert.NotNil(t, entry["timestamp"])
		assert.NotNil(t, entry["caller"])
	})

	t.Run("WithFields", func(t *testing.T) {
		var buf bytes.Buffer
		baseLogger := NewLogger(&buf, InfoLevel)

		logger := baseLogger.WithFields(
			String("service", "sage"),
			String("version", "1.0.0"),
		)

		logger.Info("test message")

		var entry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &entry)
		require.NoError(t, err)

		assert.Equal(t, "sage", entry["service"])
		assert.Equal(t, "1.0.0", entry["version"])
	})

	t.Run("WithContext", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(&buf, InfoLevel)

		ctx := context.WithValue(context.Background(), "request_id", "req-123")
		ctx = context.WithValue(ctx, "trace_id", "trace-456")

		contextLogger := logger.WithContext(ctx)
		contextLogger.Info("test message")

		var entry map[string]interface{}
		err := json.Unmarshal(buf.Bytes(), &entry)
		require.NoError(t, err)

		assert.Equal(t, "req-123", entry["request_id"])
		assert.Equal(t, "trace-456", entry["trace_id"])
	})

	t.Run("SetLevel", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(&buf, InfoLevel)

		logger.Debug("debug 1")
		assert.Empty(t, buf.String(), "Debug should be filtered at info level")

		logger.SetLevel(DebugLevel)
		logger.Debug("debug 2")
		assert.NotEmpty(t, buf.String(), "Debug should be logged at debug level")
	})

	t.Run("GetLevel", func(t *testing.T) {
		logger := NewLogger(&bytes.Buffer{}, InfoLevel)
		assert.Equal(t, InfoLevel, logger.GetLevel())

		logger.SetLevel(ErrorLevel)
		assert.Equal(t, ErrorLevel, logger.GetLevel())
	})

	t.Run("PrettyPrint", func(t *testing.T) {
		var buf bytes.Buffer
		logger := NewLogger(&buf, InfoLevel)
		logger.SetPrettyPrint(true)

		logger.Info("test message", String("key", "value"))

		output := buf.String()
		assert.Contains(t, output, "{\n")
		assert.Contains(t, output, "  \"")
		assert.Contains(t, output, "\n}")
	})
}

func TestSageError(t *testing.T) {
	t.Run("BasicError", func(t *testing.T) {
		err := NewSageError(ErrCodeInternal, "Something went wrong", nil)

		assert.Equal(t, ErrCodeInternal, err.Code)
		assert.Equal(t, "Something went wrong", err.Message)
		assert.Equal(t, "INTERNAL_ERROR: Something went wrong", err.Error())
		assert.Nil(t, err.Unwrap())
	})

	t.Run("ErrorWithCause", func(t *testing.T) {
		cause := errors.New("underlying error")
		err := NewSageError(ErrCodeNetworkError, "Network failure", cause)

		assert.Equal(t, cause, err.Unwrap())
		assert.Contains(t, err.Error(), "caused by: underlying error")
	})

	t.Run("ErrorWithDetails", func(t *testing.T) {
		err := NewSageError(ErrCodeValidationError, "Validation failed", nil)
		err.WithDetails("field", "email").
			WithDetails("reason", "invalid format")

		assert.Equal(t, "email", err.Details["field"])
		assert.Equal(t, "invalid format", err.Details["reason"])
	})

	t.Run("CommonErrorCodes", func(t *testing.T) {
		// Test all error codes are defined and not empty
		assert.NotEmpty(t, ErrCodeInternal)
		assert.NotEmpty(t, ErrCodeInvalidInput)
		assert.NotEmpty(t, ErrCodeNotFound)
		assert.NotEmpty(t, ErrCodeUnauthorized)
		assert.NotEmpty(t, ErrCodeForbidden)
		assert.NotEmpty(t, ErrCodeConflict)
		assert.NotEmpty(t, ErrCodeTimeout)
		assert.NotEmpty(t, ErrCodeNetworkError)
		assert.NotEmpty(t, ErrCodeBlockchainError)
		assert.NotEmpty(t, ErrCodeCryptoError)
		assert.NotEmpty(t, ErrCodeValidationError)
		assert.NotEmpty(t, ErrCodeConfigurationError)

		// Test specific values
		assert.Equal(t, "INTERNAL_ERROR", ErrCodeInternal)
		assert.Equal(t, "INVALID_INPUT", ErrCodeInvalidInput)
		assert.Equal(t, "NOT_FOUND", ErrCodeNotFound)
		assert.Equal(t, "UNAUTHORIZED", ErrCodeUnauthorized)
		assert.Equal(t, "FORBIDDEN", ErrCodeForbidden)
		assert.Equal(t, "CONFLICT", ErrCodeConflict)
		assert.Equal(t, "TIMEOUT", ErrCodeTimeout)
		assert.Equal(t, "NETWORK_ERROR", ErrCodeNetworkError)
		assert.Equal(t, "BLOCKCHAIN_ERROR", ErrCodeBlockchainError)
		assert.Equal(t, "CRYPTO_ERROR", ErrCodeCryptoError)
		assert.Equal(t, "VALIDATION_ERROR", ErrCodeValidationError)
		assert.Equal(t, "CONFIGURATION_ERROR", ErrCodeConfigurationError)
	})
}

func TestDefaultLogger(t *testing.T) {
	t.Run("DefaultLoggerExists", func(t *testing.T) {
		logger := GetDefaultLogger()
		assert.NotNil(t, logger)
	})

	t.Run("SetDefaultLogger", func(t *testing.T) {
		var buf bytes.Buffer
		newLogger := NewLogger(&buf, DebugLevel)
		SetDefaultLogger(newLogger)

		Debug("test debug")
		assert.NotEmpty(t, buf.String())

		buf.Reset()
		Info("test info")
		assert.NotEmpty(t, buf.String())

		buf.Reset()
		Warn("test warn")
		assert.NotEmpty(t, buf.String())

		buf.Reset()
		ErrorMsg("test error")
		assert.NotEmpty(t, buf.String())
	})
}

func TestFieldConstructors(t *testing.T) {
	t.Run("StringField", func(t *testing.T) {
		field := String("key", "value")
		assert.Equal(t, "key", field.Key)
		assert.Equal(t, "value", field.Value)
	})

	t.Run("IntField", func(t *testing.T) {
		field := Int("count", 42)
		assert.Equal(t, "count", field.Key)
		assert.Equal(t, 42, field.Value)
	})

	t.Run("BoolField", func(t *testing.T) {
		field := Bool("enabled", true)
		assert.Equal(t, "enabled", field.Key)
		assert.Equal(t, true, field.Value)
	})

	t.Run("ErrorField", func(t *testing.T) {
		err := errors.New("test error")
		field := Error(err)
		assert.Equal(t, "error", field.Key)
		assert.Equal(t, "test error", field.Value)

		// Test nil error
		field = Error(nil)
		assert.Equal(t, "error", field.Key)
		assert.Nil(t, field.Value)
	})

	t.Run("AnyField", func(t *testing.T) {
		type testStruct struct {
			Name string
		}
		value := testStruct{Name: "test"}
		field := Any("data", value)
		assert.Equal(t, "data", field.Key)
		assert.Equal(t, value, field.Value)
	})
}

func BenchmarkLogger(b *testing.B) {
	logger := NewLogger(&bytes.Buffer{}, InfoLevel)

	b.Run("SimpleLog", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("benchmark message")
		}
	})

	b.Run("LogWithFields", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			logger.Info("benchmark message",
				String("key1", "value1"),
				Int("key2", 42),
				Bool("key3", true),
			)
		}
	})

	b.Run("FilteredLog", func(b *testing.B) {
		logger.SetLevel(ErrorLevel)
		for i := 0; i < b.N; i++ {
			logger.Debug("filtered message")
		}
	})
}
