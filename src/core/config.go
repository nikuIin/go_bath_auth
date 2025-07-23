package core

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/joho/godotenv"
)


type DatabaseConfig struct {
	DBDriver string
	Host     string
	Port     string
	DBName   string
	Username string
	Password string
}


type serverConfig struct {
	host string
	port string
}

type loginAttemptWebhookConfig struct {
	url string
}

type LoggerConfig struct {
	Level slog.Level
}


func InitializeLoggerConfig() (LoggerConfig, error) {
	err := godotenv.Load()
	if err != nil {
		return LoggerConfig{}, err
	}

	var levelStr string = os.Getenv("LOGGER_LEVEL")
	if levelStr == "" {
		return LoggerConfig{
			Level: slog.LevelInfo,
		}, nil
	}

	levelStr = strings.ToUpper(levelStr)
	var level slog.Level
	switch levelStr {

	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	default:
		return LoggerConfig{},
	 		   fmt.Errorf(
					"Invalid LOGGER_LEVEL: %s expected DEBUG, INFO, WARNING or ERROR",
			 		levelStr,
			    )
	}

	return LoggerConfig{
		Level: level,
	}, nil
}

func InitializeDatabaseConfig() (DatabaseConfig, error) {

	// TODO: add regex mapping for settings

	return DatabaseConfig{
		DBDriver:   os.Getenv("DB_DRIVER"),
		Host:       os.Getenv("DB_HOST"),
		Port:       os.Getenv("DB_PORT"),
		DBName:     os.Getenv("DB_NAME"),
		Username:   os.Getenv("DB_USERNAME"),
		Password:   os.Getenv("DB_PASSWORD"),
	},  nil

}
