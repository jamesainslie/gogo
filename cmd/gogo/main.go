package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/user/gogo/internal/cli"
)

var version = "dev"

func main() {
	// Set up structured logging
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	ctx := context.Background()

	if err := cli.Execute(ctx, version); err != nil {
		slog.Error("command failed", "error", err)
		os.Exit(1)
	}
}
