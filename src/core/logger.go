package core

import (
	"log/slog"
	"os"
)


func GetConfigureLogger(level slog.Level) (*slog.Logger) {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
		AddSource: true,
	})

	logger := slog.New(handler)
	return logger
}
