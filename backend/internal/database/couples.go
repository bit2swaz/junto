package database

import (
	"context"
	"time"
)

type Couple struct {
	ID        int64     `json:"id"`
	User1ID   int64     `json:"user1_id"`
	User2ID   int64     `json:"user2_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *service) CreateCouple(ctx context.Context, user1ID, user2ID int64) (*Couple, error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	// Create couple
	query := `
		INSERT INTO couples (user1_id, user2_id, created_at)
		VALUES ($1, $2, NOW())
		RETURNING id, created_at
	`
	couple := &Couple{
		User1ID: user1ID,
		User2ID: user2ID,
	}
	err = tx.QueryRow(ctx, query, user1ID, user2ID).Scan(&couple.ID, &couple.CreatedAt)
	if err != nil {
		return nil, err
	}

	// Update users
	updateQuery := `
		UPDATE users
		SET couple_id = $1
		WHERE id = $2 OR id = $3
	`
	_, err = tx.Exec(ctx, updateQuery, couple.ID, user1ID, user2ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return couple, nil
}
