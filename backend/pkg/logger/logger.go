package logger

import (
	"log/slog"
	"os"
)

// New returns a structured JSON logger writing to stdout.
func New() *slog.Logger {
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	return slog.New(h)
}
