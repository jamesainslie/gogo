package logging

import (
	"log/slog"
	"os"
)

// SetupLogger configures structured logging
func SetupLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: verbose,
	}

	handler := slog.NewTextHandler(os.Stderr, opts)
	return slog.New(handler)
}

// SetupJSONLogger configures JSON structured logging
func SetupJSONLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: verbose,
	}

	handler := slog.NewJSONHandler(os.Stderr, opts)
	return slog.New(handler)
}
