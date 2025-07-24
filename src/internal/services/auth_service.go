package services

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/nikuIin/base_go_auth/src/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenRevoked      = errors.New("token revoked")
	ErrTokenNotFound     = errors.New("token not found")
	ErrUserAgentMismatch = errors.New("user agent mismatch")
	ErrNotPairsTokens    = errors.New("token not from one pair")
)

type AuthService struct {
	repo                     repository.TokenRepository
	logger                   *slog.Logger
	jwtSecret                string
	accessExpireTime         time.Duration
	refreshExpireTime        time.Duration
	notifyNewLoginWebhookUrl string
	// TODO: думаю хорошей идеей сделать максимальное количество refresh токенов для юзера
}

func NewAuthService(
	repo repository.TokenRepository,
	logger *slog.Logger,
	jwtSecret string,
	accessExpireTime time.Duration,
	refreshExpireTime time.Duration,
	notifyNewLoginWebhookUrl string,
) *AuthService {
	return &AuthService{
		repo:                     repo,
		logger:                   logger,
		jwtSecret:                jwtSecret,
		accessExpireTime:         accessExpireTime,
		refreshExpireTime:        refreshExpireTime,
		notifyNewLoginWebhookUrl: notifyNewLoginWebhookUrl,
	}
}


func (s *AuthService) hashRefreshToken(token []byte) (string, error) {
	hash, err := bcrypt.GenerateFromPassword(token, bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash refresh token", "error", err)
		return "", err
	}
	return string(hash), nil
}

func (s *AuthService) GenerateTokens(ctx context.Context, userID, ipAddress, userAgent string) (accessToken, refreshToken string, err error) {
	var jti string = uuid.New().String()

	// TODO: можно поменять UserAgent на fingerprint браузера
	// Add userID to the database (this function will not add userID if user already exists)
	// Пометка: в более общем сервисе аутентификации я бы вынес добавление пользователя в методы регистрации.
	// Также же сейчас таблица пользователей хранит только его id, но я все равно решил выделить отдельную
	// таблицу с расчетом на то, что система аутентификации могла бы расшириться и добавились
	// бы новые поля для пользователей.
	err = s.repo.AddUser(ctx, userID)
	if err != nil {
		return "", "", err
	}

	// generate access token
	accessPayload := jwt.MapClaims{
		"sub": userID,
		"jti": jti,
		"exp": time.Now().Add(s.accessExpireTime).Unix(),
		"iat": time.Now().Unix(),
	}
	accessJWT := jwt.NewWithClaims(jwt.SigningMethodHS512, accessPayload)
	accessToken, err = accessJWT.SignedString([]byte(s.jwtSecret))
	if err != nil {
		s.logger.Error("Failed to sign access token", "error", err)
		return "", "", err
	}

	// generate refresh token
	refreshBytes := make([]byte, 32)
	_, err = rand.Read(refreshBytes)
	if err != nil {
		s.logger.Error("Failed to generate random bytes for refresh token", "error", err)
		return "", "", err
	}

	refreshToken = base64.RawStdEncoding.Strict().EncodeToString(refreshBytes)

	tokenHash, err := s.hashRefreshToken(refreshBytes)
	if err != nil {
		return "", "", err
	}

	createdAt := time.Now()
	expiresAt := createdAt.Add(s.refreshExpireTime)
	err = s.repo.StoreRefreshToken(ctx, tokenHash, jti, userID, ipAddress, userAgent, createdAt, expiresAt)
	if err != nil {
		return "", "", err
	}



	return accessToken, refreshToken, err
}

func (s *AuthService) RefreshTokens(
	ctx context.Context, accessToken, refreshToken string,
) (newAccessToken, newRefreshToken string, err error) {
	// Verify accessToken.
	userID, accessJTI, err := s.VerifyAccessToken(ctx, accessToken)
	if err != nil {
		return "", "", err
	}

	// Verify refreshToken.
	refreshJTI, oldTokenHash, err := s.VerifyRefreshToken(ctx, refreshToken, userID)
	if err != nil {
		return "", "", err
	}

	if accessJTI != refreshJTI {
		return "", "", ErrNotPairsTokens
	}

	// Delete old refresh token from database
	err = s.repo.RevokeToken(ctx, oldTokenHash)
	if err != nil {
		return "", "", err
	}

	// Generate new tokens.
	newAccessToken, newRefreshToken, err = s.GenerateTokens(ctx, userID, ctx.Value("ipAddress").(string), ctx.Value("userAgent").(string))
	if err != nil {
		return "", "", err
	}

	return newAccessToken, newRefreshToken, nil
}

func (s *AuthService) VerifyAccessToken(ctx context.Context, accessToken string) (userID, jti string, err error) {
	token, err := jwt.Parse(accessToken, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		s.logger.Info("Access token verification failed", "error", err)
		return "", "", ErrInvalidToken
	}

	payload, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		s.logger.Info("Invalid access token payload, not a MapClaims")
		return "", "", ErrInvalidToken
	}

	userID, ok = payload["sub"].(string)
	if !ok {
		s.logger.Info("Invalid 'sub' claim in access token")
		return "", "", ErrInvalidToken
	}

	jti, ok = payload["jti"].(string)
	if !ok {
		s.logger.Info("Invalid 'jti' claim in access token")
		return "", "", ErrInvalidToken
	}

	return userID, jti, nil
}

func (s *AuthService) VerifyRefreshToken(
	ctx context.Context, refreshToken, userID string,
) (string, string, error) {
	refreshBytes, err := base64.RawStdEncoding.Strict().DecodeString(refreshToken)
	if err != nil {
		s.logger.Info("Failed to decode refresh token", "error", err)
		return "", "", ErrInvalidToken
	}

	var jti string
	var tokenHash string
	var refreshTokenData repository.TokenData
	var refreshTokenDataArray []repository.TokenData

	refreshTokenDataArray, repoErr := s.repo.GetRefreshUserTokens(ctx, userID)
	if repoErr != nil {
		if repoErr == sql.ErrNoRows {
			s.logger.Info("Refresh token not found in db", "user", userID)
			return "", "", ErrInvalidToken
		}
		s.logger.Info("Failed to get refresh token from db", "error", repoErr, "user", userID)
		return "", "", repoErr
	}

	foundMatch := false
	for _, token := range refreshTokenDataArray {
		// Compare bcrypt hash from the provided refreshBytes with the stored hash for this token
		compareErr := bcrypt.CompareHashAndPassword([]byte(token.TokenHash), refreshBytes)
		if compareErr == nil {
			// If match found, then break
			refreshTokenData = token
			jti = token.JTI
			tokenHash = token.TokenHash
			foundMatch = true
			break
		} else if compareErr == bcrypt.ErrMismatchedHashAndPassword {
			s.logger.Debug(
				"Candidate refresh token hash mismatch",
				"token_hash", refreshTokenData.TokenHash,
				"userID", token.UserID,
			)
		} else {
			s.logger.Warn(
				"Bcrypt comparison failed for a candidate token due to unexpected error",
				"error", compareErr,
				"token_hash", refreshTokenData.TokenHash,
				"userID", token.UserID,
			)
		}
	}

	if !foundMatch {
		s.logger.Info(
			"No matching refresh token found for user in database after iterating all records",
			"userID", userID,
		)
		return "", "", ErrInvalidToken
	}

	if time.Now().After(refreshTokenData.ExpiresAt) {
		s.logger.Info(
			"Refresh token expired",
			"token_hash", refreshTokenData.TokenHash,
			"userID", userID,
		)
		if repoErr = s.repo.RevokeToken(ctx, refreshTokenData.TokenHash); repoErr != nil {
			s.logger.Error(
				"Failed to revoke expired token",
				"error", repoErr,
				"token_hash", refreshTokenData.TokenHash,
			)
		}
		return "", "", ErrInvalidToken
	}

	err = bcrypt.CompareHashAndPassword([]byte(refreshTokenData.TokenHash), refreshBytes)
	if err != nil {
		s.logger.Info(
			"Refresh token hash mismatch",
			"token_hash", refreshTokenData.TokenHash,
			"userID", userID)
		if repoErr = s.repo.RevokeToken(ctx, refreshTokenData.TokenHash); repoErr != nil {
			s.logger.Error(
				"Failed to revoke token after hash mismatch",
				"error", repoErr,
				"token_hash", refreshTokenData.TokenHash,
			)
		}
		return "", "", ErrInvalidToken
	}

	userAgent, ok := ctx.Value("userAgent").(string)
	if !ok || refreshTokenData.UserAgent != userAgent {
		s.logger.Info(
			"User agent mismatch",
			"token_hash", refreshTokenData.TokenHash,
			"userAgent", refreshTokenData.UserAgent,
			"userAgentIn", userAgent,
			"user_id", userID,
		)
		if repoErr = s.RevokeUsersRefreshTokens(ctx, userID); repoErr != nil {
			s.logger.Error(
				"Failed to revoke user tokens after user agent mismatch",
				"error", repoErr,
				"userID", userID,
			)
		}
		return "", "", ErrUserAgentMismatch
	}

	ipAddress, ok := ctx.Value("ipAddress").(string)
	if !ok || refreshTokenData.IPAddress != ipAddress {
		s.NotifyNewLoginWebhook(
			refreshTokenData.UserID,
		  	ipAddress,
			refreshTokenData.IPAddress,
		 	time.Now(),
		)
	}

	return jti, tokenHash, nil
}

func (s *AuthService) RevokeUsersRefreshTokens(ctx context.Context, userID string) error {
	err := s.repo.RevokeTokensByUserID(ctx, userID)
	if err != nil {
		s.logger.Error("failed to revoke user's refresh tokens", "error", err, "userID", userID)
		return err
	}
	return nil
}

func (s *AuthService) NotifyNewLoginWebhook(userID, newIPAddress, oldIPAddress string, timestamp time.Time) {
	s.logger.Info("Check")
	go func() {
		payload := map[string]any{
			"user_id":    userID,
			"old_ip_address": oldIPAddress,
			"new_ip_address": newIPAddress,
			"timestamp":  timestamp.Format(time.RFC3339),
		}
		body, err := json.Marshal(payload)
		if err != nil {
			s.logger.Error("Failed to marshal webhook payload", "error", err, "userID", userID)
			return
		}

		req, err := http.NewRequest("POST", s.notifyNewLoginWebhookUrl, strings.NewReader(string(body)))
		if err != nil {
			s.logger.Error("Failed to create webhook request", "error", err, "userID", userID)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			s.logger.Error("Failed to send webhook", "error", err, "userID", userID)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			s.logger.Warn(
				"webhook returned non-200 status",
				"status_code", resp.StatusCode,
				"userID", userID,
			)
		} else {
			s.logger.Info("Webhook sent successfully", "userID", userID)
		}
	}()
}
