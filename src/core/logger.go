package core

import (
	"io"
	"log/slog"
	"os"
)


func GetConfigureLogger(level slog.Level) (*slog.Logger) {
	logFilePath := "logs.json"

	file, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// If opening the log file fails, panic as logging to file is a critical requirement
		panic("Failed to open log file: " + err.Error())
	}

	multiWriter := io.MultiWriter(os.Stdout, file)

	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: level,
		AddSource: true,
	})

	logger := slog.New(handler)
	return logger
}
