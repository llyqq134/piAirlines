package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrRefreshInvalid = errors.New("refresh token invalid")

func NewRefreshToken() (plain string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	plain = base64.RawURLEncoding.EncodeToString(b)
	sum := sha256.Sum256([]byte(plain))
	hash = sum[:]
	return plain, hash, nil
}

func HashRefreshToken(plain string) []byte {
	sum := sha256.Sum256([]byte(plain))
	return sum[:]
}

func StoreRefreshToken(ctx context.Context, db *pgxpool.Pool, userID int64, tokenHash []byte, expiresAt time.Time) error {
	_, err := db.Exec(ctx, `
		INSERT INTO refresh_tokens(user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

func ConsumeRefreshToken(ctx context.Context, db *pgxpool.Pool, tokenHash []byte) (userID int64, err error) {
	// one-time use: revoke immediately (rotation)
	row := db.QueryRow(ctx, `
		UPDATE refresh_tokens
		SET revoked_at=now()
		WHERE token_hash=$1 AND revoked_at IS NULL AND expires_at > now()
		RETURNING user_id
	`, tokenHash)
	if err := row.Scan(&userID); err != nil {
		return 0, ErrRefreshInvalid
	}
	return userID, nil
}

func RevokeAllRefreshTokens(ctx context.Context, db *pgxpool.Pool, userID int64) error {
	_, err := db.Exec(ctx, `UPDATE refresh_tokens SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, userID)
	return err
}
