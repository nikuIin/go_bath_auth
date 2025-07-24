package v1

import (
	"encoding/json"
	"errors"
	"log/slog"

	"context" // Import context package

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/nikuIin/base_go_auth/src/core"
	"github.com/nikuIin/base_go_auth/src/internal/services"
)

// ErrorResponse represents a standard error response.
type ErrorResponse struct {
	Error string `json:"error" example:"error message"`
}

type TokenPairResponse struct {
	AccessToken  string `json:"access_token" example:"eyJhbGciOiJIUzUxMiIsInR5cCI6IkpXVCJ9..."`
	RefreshToken string `json:"refresh_token" example:"V29uZGVyZnVsIHJlZnJlc2ggdG9rZW4h"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" example:"V29uZGVyZnVsIHJlZnJlc2ggdG9rZW4h"`
}

type UserGUIDResponse struct {
	UserID string `json:"user_id" example:"a1b2c3d4-e5f6-7890-1234-567890abcdef"`
}

type SuccessResponse struct {
	Message string `json:"message" example:"operation successful"`
}

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

const (
	userAgentContextKey string = "userAgent"
	ipAddressContextKey string = "ipAddress"
)

// Declare loggerConfig and a variable for its initialization error at package level.
// This allows them to be set once when the package is initialized.
var (
	logger slog.Logger
	loggerConfig core.LoggerConfig
	initErr      error // Renamed from 'error' to 'initErr' to avoid shadowing the built-in type
)

func init() {
	loggerConfig, initErr = core.InitializeLoggerConfig()
	if initErr != nil {
		panic("Failed to initialize logger configuration: " + initErr.Error())
	}

	logger = *core.GetConfigureLogger(loggerConfig.Level)
}


// @Summary      Generate a new token pair
// @Description  Generates a new access and refresh token pair for a given user ID.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        user_id query string true "User GUID" Format(uuid)
// @Success      200 {object} TokenPairResponse
// @Failure      400 {object} ErrorResponse "Invalid request: user_id is required"
// @Failure      422 {object} ErrorResponse "Invalid request: user_id must be a valid UUID"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /api/v1/auth/token [post]
func (h *AuthHandler) GenerateTokenPair(c *fiber.Ctx) error {
	userID := c.Query("user_id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id is required"})
	}

	// Add UUID validation check
	if _, err := uuid.Parse(userID); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "user_id must be a valid UUID"})
	}

	ipAddress := h.getFirstValidIP(c)
	userAgent := string(c.Request().Header.UserAgent())

	logger.Info("Generate new tokens pair", "user_id", userID, "ip_address", ipAddress, "user_agent", userAgent)

	// Create a new context and add IP and User-Agent to it
	ctxWithData := context.WithValue(c.Context(), ipAddressContextKey, ipAddress)
	ctxWithData = context.WithValue(ctxWithData, userAgentContextKey, userAgent)

	accessToken, refreshToken, err := h.authService.GenerateTokens(ctxWithData, userID, ipAddress, userAgent)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not generate tokens"})
	}

	return c.JSON(fiber.Map{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
	})
}

// @Summary      Refresh a token pair
// @Description  Refreshes an existing token pair using a valid refresh token.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        Authorization header string true "Bearer {access_token}"
// @Security     ApiKeyAuth
// @Param        refresh_token body RefreshTokenRequest true "Refresh Token"
// @Success      200 {object} TokenPairResponse
// @Failure      400 {object} ErrorResponse "Invalid request body"
// @Failure      401 {object} ErrorResponse "Unauthorized: invalid or missing token"
// @Router       /api/v1/auth/token/refresh [post]
func (h *AuthHandler) RefreshTokenPair(c *fiber.Ctx) error {

	var req RefreshTokenRequest
	reqBytes := c.Request().Body()
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": "refresh token is invalid format"})
	}

	ipAddress := h.getFirstValidIP(c)
	userAgent := string(c.Request().Header.UserAgent())


	// Create a new context and add IP and User-Agent to it
	ctxWithData := context.WithValue(c.Context(), ipAddressContextKey, ipAddress)
	ctxWithData = context.WithValue(ctxWithData, userAgentContextKey, userAgent)

	accessToken, ok := c.Locals("access_token").(string)
	if !ok || accessToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}
	userID, ok := c.Locals("user_id").(string)
	if !ok || accessToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}

	logger.Info("Refresh tokens.", "user_id", userID, "user_agent", userAgent, "ip_address", ipAddress)

	newAccessToken, newRefreshToken, err := h.authService.RefreshTokens(ctxWithData, accessToken, req.RefreshToken)
	if err != nil {
		// Handle specific errors from the service layer
		if errors.Is(err, services.ErrInvalidToken) ||
					errors.Is(err, services.ErrTokenRevoked) ||
					errors.Is(err, services.ErrTokenNotFound) ||
					errors.Is(err, services.ErrNotPairsTokens) ||
					errors.Is(err, services.ErrUserAgentMismatch) { // User agent mismatch also leads to Unauthorized
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not refresh tokens"})
	}

	return c.JSON(fiber.Map{
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
	})
}

// @Summary      Logout user
// @Description  Revokes all refresh tokens for the current user, effectively logging them out.
// @Tags         Auth
// @Security     ApiKeyAuth
// @Produce      json
// @Param        Authorization header string true "Bearer {access_token}"
// @Success      200 {object} SuccessResponse
// @Failure      401 {object} ErrorResponse "Unauthorized: invalid or missing token"
// @Failure      500 {object} ErrorResponse "Internal server error"
// @Router       /api/v1/auth/token/logout [post]
func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}

	if err := h.authService.RevokeUsersRefreshTokens(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "could not logout"})
	}

	return c.JSON(fiber.Map{"message": "logged out successfully"})
}

// @Summary      Get current user's GUID
// @Description  Retrieves the GUID of the user associated with the provided access token.
// @Tags         User
// @Security     ApiKeyAuth
// @Produce      json
// @Param        Authorization header string true "Bearer {access_token}"
// @Success      200 {object} UserGUIDResponse
// @Failure      401 {object} ErrorResponse "Unauthorized: invalid or missing token"
// @Router       /api/v1/user/me [get]
func (h *AuthHandler) GetMyGUID(c *fiber.Ctx) error {
	userID, ok := c.Locals("user_id").(string)
	if !ok || userID == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid token"})
	}

	return c.JSON(fiber.Map{"user_id": userID})
}


func (h *AuthHandler) getFirstValidIP(c *fiber.Ctx) string {
	ipAddresses := c.IPs()
	var ipAddress string = ""
	if len(ipAddresses) > 0 {
		ipAddress = ipAddresses[0]
	}

	return ipAddress
}
