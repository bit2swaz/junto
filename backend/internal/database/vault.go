package database

import (
	"context"
	"time"
)

type VaultItem struct {
	ID          int64     `json:"id"`
	CoupleID    int64     `json:"couple_id"`
	CreatedBy   int64     `json:"created_by"`
	ContentText string    `json:"content_text,omitempty"`
	UnlockAt    time.Time `json:"unlock_at"`
	CreatedAt   time.Time `json:"created_at"`
	Locked      bool      `json:"locked,omitempty"`
}

func (s *service) CreateVaultItem(ctx context.Context, coupleID, userID int64, content string, unlockAt time.Time) (*VaultItem, error) {
	query := `
		INSERT INTO vault_items (couple_id, created_by, content_text, unlock_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, couple_id, created_by, content_text, unlock_at, created_at
	`
	var item VaultItem
	err := s.db.QueryRow(ctx, query, coupleID, userID, content, unlockAt).Scan(
		&item.ID, &item.CoupleID, &item.CreatedBy, &item.ContentText, &item.UnlockAt, &item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *service) GetVaultItems(ctx context.Context, coupleID, userID int64) ([]VaultItem, error) {
	query := `
		SELECT id, couple_id, created_by, content_text, unlock_at, created_at
		FROM vault_items
		WHERE couple_id = $1
		ORDER BY created_at DESC
	`
	rows, err := s.db.Query(ctx, query, coupleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []VaultItem
	now := time.Now()

	for rows.Next() {
		var item VaultItem
		err := rows.Scan(
			&item.ID, &item.CoupleID, &item.CreatedBy, &item.ContentText, &item.UnlockAt, &item.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Mask content if locked AND not the owner
		if item.UnlockAt.After(now) {
			item.Locked = true
			if item.CreatedBy != userID {
				item.ContentText = "" // Hide content for non-owners
			}
		} else {
			item.Locked = false
		}

		items = append(items, item)
	}
	return items, nil
}
