package repository

import (
	"context"
	"database/sql"
	"log/slog"
	"time"
)

type TokenData struct {
	JTI       string
	TokenHash string
	UserID    string
	IPAddress string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

type TokenRepository struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewTokenRepository(db *sql.DB, logger *slog.Logger) *TokenRepository {
	return &TokenRepository{db: db, logger: logger}
}

func (r *TokenRepository) AddUser(ctx context.Context, userID string) error {
	query := `INSERT INTO "user" (user_id) VALUES ($1) ON CONFLICT (user_id) DO NOTHING;`

	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to add user to db", "error", err, "userID", userID)
		return err
	}

	r.logger.Debug("Successfully executed add user query", "userID", userID)
	return nil
}

func (r *TokenRepository) StoreRefreshToken(
	ctx context.Context,
	tokenHash, jti, userID, ipAddress, userAgent string,
	createdAt, expiresAt time.Time,
) error {
	query := `
		INSERT INTO refresh_token (refresh_token_id, user_id, token_hash, ip_address, user_agent, created_at, expires_at)
		VALUES ($1::UUID, $2::UUID, $3, $4, $5, $6, $7);
	`
	_, err := r.db.ExecContext(ctx, query, jti, userID, tokenHash, ipAddress, userAgent, createdAt, expiresAt)
	if err != nil {
		r.logger.Error("Failed to store refresh token in db", "error", err, "jti", jti)
		return err
	}

	r.logger.Debug("Successfully stored refresh token", "jti", jti, "userID", userID)
	return nil
}

func (r *TokenRepository) RevokeToken(ctx context.Context, token_hash string) error {
	query := `DELETE FROM refresh_token WHERE token_hash=$1;`

	_, err := r.db.ExecContext(ctx, query, token_hash)
	if err != nil {
		r.logger.Error("Failed to revoke token from db", "error", err, "token_hash", token_hash)
		return err
	}

	r.logger.Debug("Successfully revoked token", "token_hash", token_hash)
	return nil
}

func (r *TokenRepository) GetRefreshUserTokens(ctx context.Context, userID string) ([]TokenData, error) {
	query := `
		SELECT user_id, refresh_token_id, token_hash, ip_address, user_agent, created_at, expires_at
		FROM refresh_token
			WHERE user_id=$1 and expires_at > current_timestamp;
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to get refresh tokens from db", "error", err, "userID", userID)
		return nil, err
	}
	defer rows.Close()

	var tokens []TokenData
	for rows.Next() {
		var tokenData TokenData
		err := rows.Scan(
			&tokenData.UserID,
			&tokenData.JTI,
			&tokenData.TokenHash,
			&tokenData.IPAddress,
			&tokenData.UserAgent,
			&tokenData.CreatedAt,
			&tokenData.ExpiresAt,
		)
		if err != nil {
			r.logger.Error("Failed to scan refresh token row", "error", err, "userID", userID)
			return nil, err
		}
		tokens = append(tokens, tokenData)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error during rows iteration for refresh tokens", "error", err, "userID", userID)
		return nil, err
	}

	r.logger.Debug("Successfully retrieved refresh tokens", "count", len(tokens), "userID", userID)
	return tokens, nil
}

func (r *TokenRepository) RevokeTokensByUserID(ctx context.Context, userID string) error {
	query := `DELETE FROM refresh_token WHERE user_id=$1;`

	result, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to revoke all tokens for user", "error", err, "userID", userID)
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.Debug("Successfully revoked all tokens for user", "userID", userID, "revoked_count", rowsAffected)
	return nil
}
