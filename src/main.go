package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	_ "github.com/nikuIin/base_go_auth/docs" // Import generated docs
	"github.com/nikuIin/base_go_auth/src/core"
	"github.com/nikuIin/base_go_auth/src/db"
	v1 "github.com/nikuIin/base_go_auth/src/internal/api/v1"
	"github.com/nikuIin/base_go_auth/src/internal/repository"
	"github.com/nikuIin/base_go_auth/src/internal/services"
)

// @title           Go Base Auth API
// @version         1.0
// @description     This is a sample authentication service.
func main() {
	// Setup logger first with a default level
	logger := core.GetConfigureLogger(slog.LevelDebug)

	// Connect to database
	databaseConfig, err := core.InitializeDatabaseConfig()
	logger.Info("Database driver", "driver", databaseConfig.DBDriver, "host", databaseConfig.Host)
	database, err := db.ConnectToDatabase(databaseConfig, logger)
	if err != nil {
		logger.Error("Could not connect to the database", "error", err)
		os.Exit(1)
	}

	// Create repository
	tokenRepo := repository.NewTokenRepository(database, logger)


	jwtConfig, err := core.InitializeJWTConfig()
	if err != nil {
		logger.Error("Could not initialize server config", "error", err)
		os.Exit(1)
	}
	notificationWebhookConfig, err := core.InitializeLoginAttemptWebhookConfig()
	if err != nil {
		logger.Error("Could not initialize notification webhook config", "error", err)
		os.Exit(1)
	}
	// Create service
	authService := services.NewAuthService(
		*tokenRepo, // Dereference tokenRepo to match expected type
		logger,
		jwtConfig.Secret,
		time.Minute*time.Duration(jwtConfig.ExpiresAccessMinutes),    // accessExpireTime
		time.Minute*time.Duration(jwtConfig.ExpiresRefreshMinutes),  // refreshExpireTime
		notificationWebhookConfig.URL,
	)

	// Create handler
	authHandler := v1.NewAuthHandler(authService)
	serverConfig, err := core.InitializeServerConfig()
	if err != nil {
		logger.Error("Could not initialize server config", "error", err)
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{
		CaseSensitive: true,
		StrictRouting: true,
		EnableTrustedProxyCheck: true,
		AppName:       serverConfig.Title,
		DisableStartupMessage: true,
	})

	// Setup V1 Routes
	v1.SetupRoutes(app, authHandler, authService)

	logger.Info("Starting server", "port", serverConfig.Port)
	err = app.Listen(":" + serverConfig.Port)
	if err != nil {
		logger.Error("Could not start server", "error", err)
		os.Exit(1)
	}
}
