package services

import (
	"context"
	"database/sql"
	"errors"

	"github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

type PortfolioService struct {
	db *sql.DB
}

func NewPortfolioService(db *sql.DB) *PortfolioService {
	return &PortfolioService{db: db}
}

func (s *PortfolioService) Create(ctx context.Context, portfolio *models.Portfolio) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO portfolios (user_id, name, description, balance, risk, strategy)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`

	err = tx.QueryRowContext(
		ctx,
		query,
		portfolio.UserID,
		portfolio.Name,
		portfolio.Description,
		portfolio.Balance,
		portfolio.Risk,
		portfolio.Strategy,
	).Scan(&portfolio.ID, &portfolio.CreatedAt, &portfolio.UpdatedAt)

	if err != nil {
		return err
	}

	return tx.Commit()
}