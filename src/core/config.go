package core

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
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


type ServerConfig struct {
	Title string
	Port string
}

type LoginAttemptWebhookConfig struct {
	URL string
}

type JWTConfig struct {
	Secret string
	ExpiresAccessMinutes int
	ExpiresRefreshMinutes int
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
	case "WARNING":
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
	err := godotenv.Load()
	if err != nil {
		return DatabaseConfig{}, err
	}

	return DatabaseConfig{
		DBDriver:   os.Getenv("DB_DRIVER"),
		Host:       os.Getenv("DB_HOST"),
		Port:       os.Getenv("DB_PORT"),
		DBName:     os.Getenv("DB_NAME"),
		Username:   os.Getenv("DB_USERNAME"),
		Password:   os.Getenv("DB_PASSWORD"),
	},  nil

}

func InitializeLoginAttemptWebhookConfig() (LoginAttemptWebhookConfig, error) {
	err := godotenv.Load()
	if err != nil {
		return LoginAttemptWebhookConfig{}, err
	}

	return LoginAttemptWebhookConfig{
		URL:   os.Getenv("NOTIFICATION_WEBHOOK_URL"),
	},  nil
}

func InitializeJWTConfig() (JWTConfig, error) {
	err := godotenv.Load()
	if err != nil {
		return JWTConfig{}, err
	}

	expiresAccessMinutes, err := strconv.Atoi(os.Getenv("EXPIRES_ACCESS_MINUTES"))
	if err != nil {
		return JWTConfig{}, err
	}

	expiresRefreshMinutes, err := strconv.Atoi(os.Getenv("EXPIRES_REFRESH_MINUTES"))
	if err != nil {
		return JWTConfig{}, err
	}

	return JWTConfig{
		Secret:   os.Getenv("APPLICATION_HOST"),
		ExpiresAccessMinutes:  expiresAccessMinutes,
		ExpiresRefreshMinutes: expiresRefreshMinutes,
	},  nil
}


func InitializeServerConfig() (ServerConfig, error) {
	err := godotenv.Load()
	if err != nil {
		return ServerConfig{}, err
	}

	return ServerConfig{
		Title: os.Getenv("APP_NAME"),
		Port:  os.Getenv("APPLICATION_PORT"),
	}, nil
}
