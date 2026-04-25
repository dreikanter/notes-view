package logging

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDefaults(t *testing.T) {
	ctx := context.Background()
	logger, closer, err := New(Config{})
	require.NoError(t, err, "New")
	assert.Nil(t, closer)
	require.NotNil(t, logger)
	assert.True(t, logger.Enabled(ctx, slog.LevelInfo), "info level should be enabled by default")
	assert.False(t, logger.Enabled(ctx, slog.LevelDebug), "debug level should not be enabled by default")
}

func TestNewLevels(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"DEBUG", slog.LevelDebug},
	}
	for _, tt := range cases {
		t.Run(tt.in, func(t *testing.T) {
			logger, _, err := New(Config{Level: tt.in})
			require.NoError(t, err, "New(%q)", tt.in)
			assert.True(t, logger.Enabled(ctx, tt.want), "level %q: want %v enabled", tt.in, tt.want)
		})
	}
}

func TestNewInvalidLevel(t *testing.T) {
	_, _, err := New(Config{Level: "loud"})
	require.Error(t, err)
}

func TestNewInvalidFormat(t *testing.T) {
	_, _, err := New(Config{Format: "yaml"})
	require.Error(t, err)
}

func TestNewWithFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "nview.log")

	logger, closer, err := New(Config{File: path, Format: "json"})
	require.NoError(t, err, "New")
	require.NotNil(t, closer)
	defer func() { _ = closer.Close() }()

	logger.Info("hello", "who", "world")

	// The file should have been created alongside any missing parent dirs
	// and should contain the JSON-encoded record.
	data, err := os.ReadFile(path)
	require.NoError(t, err, "read log file")
	s := string(data)
	assert.Contains(t, s, `"msg":"hello"`)
	assert.Contains(t, s, `"who":"world"`)
}

func TestDiscardLogger(t *testing.T) {
	logger := Discard()
	require.NotNil(t, logger)
	// Error is the lowest level we allow through, but output goes to io.Discard,
	// so this should not panic or produce visible output.
	logger.Error("dropped")
}
