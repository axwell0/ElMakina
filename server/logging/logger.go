package logging

import (
	"log/slog"
	"os"
)

// NewServerLogger returns the default structured logger used for operational
func NewServerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
}
