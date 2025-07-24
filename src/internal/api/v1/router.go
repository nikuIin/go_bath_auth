package v1

import (
	"github.com/nikuIin/base_go_auth/src/internal/services"

	"github.com/gofiber/fiber/v2"
	swagger "github.com/gofiber/swagger"
)

// SetupRoutes sets up all the v1 routes.
func SetupRoutes(app *fiber.App, handler *AuthHandler, authService *services.AuthService) {
	// Swagger documentation route
	app.Get("/swagger/*", swagger.HandlerDefault)

	api := app.Group("/api/v1")

	authMiddleware := AuthMiddleware(authService)

	// Auth routes
	api.Post("/auth/token", handler.GenerateTokenPair)
	api.Post("/auth/token/refresh", authMiddleware, handler.RefreshTokenPair)
	api.Post("/auth/token/logout", authMiddleware, handler.Logout)

	// User routes
	api.Get("/user/me", authMiddleware, handler.GetMyGUID)
}
