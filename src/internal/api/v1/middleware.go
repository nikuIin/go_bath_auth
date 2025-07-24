package v1

import (
	"errors"
	"strings"

	"github.com/nikuIin/base_go_auth/src/internal/services"

	"github.com/gofiber/fiber/v2"
)

func AuthMiddleware(authService *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header format. Should be: Bearer {access_token}"})
		}

		accessToken := parts[1]
		userID, _, _, err := authService.VerifyAccessToken(c.Context(), accessToken)
		if errors.Is(err, services.ErrTokenBlocked) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "token is blocked"})
		} else if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}


		c.Locals("access_token", accessToken)
		c.Locals("user_id", userID)
		return c.Next()
	}
}
