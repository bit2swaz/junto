package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	AvatarConfig string    `json:"avatar_config"` // Stored as JSONB in DB, string here for simplicity or json.RawMessage
	CreatedAt    time.Time `json:"created_at"`
	CoupleID     *int64    `json:"couple_id,omitempty"`
}

func (s *service) CreateUser(ctx context.Context, email, passwordHash string) (*User, error) {
	query := `
		INSERT INTO users (email, password_hash, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id, email, created_at
	`
	user := &User{
		Email:        email,
		PasswordHash: passwordHash,
	}
	err := s.db.QueryRow(ctx, query, email, passwordHash).Scan(&user.ID, &user.Email, &user.CreatedAt)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		SELECT id, email, password_hash, created_at, couple_id
		FROM users
		WHERE email = $1
	`
	user := &User{}
	err := s.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.CoupleID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}

func (s *service) GetUserByID(ctx context.Context, id int64) (*User, error) {
	query := `
		SELECT id, email, password_hash, created_at, couple_id
		FROM users
		WHERE id = $1
	`
	user := &User{}
	err := s.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.CreatedAt,
		&user.CoupleID,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return user, nil
}
