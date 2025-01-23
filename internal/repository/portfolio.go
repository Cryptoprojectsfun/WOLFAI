package repository

import (
    "context"
    "fmt"
    "github.com/QUOTRIX/WOLFAI/internal/database"
    "github.com/QUOTRIX/WOLFAI/internal/models"
)

type PortfolioRepository struct {
    db *database.DB
}

func NewPortfolioRepository(db *database.DB) *PortfolioRepository {
    return &PortfolioRepository{db: db}
}

func (r *PortfolioRepository) Create(ctx context.Context, portfolio *models.Portfolio) error {
    qb := database.NewQueryBuilder()
    qb.AddParam("user_id", portfolio.UserID)
    qb.AddParam("name", portfolio.Name)
    qb.AddParam("description", portfolio.Description)
    qb.AddParam("balance", portfolio.Balance)
    qb.AddParam("risk", portfolio.Risk)
    qb.AddParam("strategy", portfolio.Strategy)

    query, args := qb.Build(`
        INSERT INTO portfolios (user_id, name, description, balance, risk, strategy)
        VALUES (@user_id, @name, @description, @balance, @risk, @strategy)
        RETURNING id, created_at, updated_at
    `)

    return r.db.WithTransaction(ctx, func(tx *sql.Tx) error {
        err := tx.QueryRowContext(ctx, query, args...).Scan(
            &portfolio.ID,
            &portfolio.CreatedAt,
            &portfolio.UpdatedAt,
        )
        if err != nil {
            return fmt.Errorf("create portfolio: %w", err)
        }

        return nil
    })
}

func (r *PortfolioRepository) Get(ctx context.Context, id, userID int64) (*models.Portfolio, error) {
    qb := database.NewQueryBuilder()
    qb.AddParam("id", id)
    qb.AddParam("user_id", userID)

    query, args := qb.Build(`
        SELECT id, user_id, name, description, balance, risk, strategy, created_at, updated_at
        FROM portfolios
        WHERE id = @id AND user_id = @user_id
    `)

    var portfolio models.Portfolio
    err := r.db.QueryRowSafe(ctx, query, args...).Scan(
        &portfolio.ID,
        &portfolio.UserID,
        &portfolio.Name,
        &portfolio.Description,
        &portfolio.Balance,
        &portfolio.Risk,
        &portfolio.Strategy,
        &portfolio.CreatedAt,
        &portfolio.UpdatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("get portfolio: %w", err)
    }

    return &portfolio, nil
}

func (r *PortfolioRepository) List(ctx context.Context, userID int64, limit, offset int) ([]models.Portfolio, error) {
    qb := database.NewQueryBuilder()
    qb.AddParam("user_id", userID)
    qb.AddParam("limit", database.SafeLimit(limit))
    qb.AddParam("offset", database.SafeOffset(offset))

    query, args := qb.Build(`
        SELECT id, user_id, name, description, balance, risk, strategy, created_at, updated_at
        FROM portfolios
        WHERE user_id = @user_id
        ORDER BY created_at DESC
        LIMIT @limit OFFSET @offset
    `)

    rows, err := r.db.QuerySafe(ctx, query, args...)
    if err != nil {
        return nil, fmt.Errorf("list portfolios: %w", err)
    }
    defer rows.Close()

    var portfolios []models.Portfolio
    for rows.Next() {
        var p models.Portfolio
        err := rows.Scan(
            &p.ID,
            &p.UserID,
            &p.Name,
            &p.Description,
            &p.Balance,
            &p.Risk,
            &p.Strategy,
            &p.CreatedAt,
            &p.UpdatedAt,
        )
        if err != nil {
            return nil, fmt.Errorf("scan portfolio: %w", err)
        }
        portfolios = append(portfolios, p)
    }

    return portfolios, nil
}

func (r *PortfolioRepository) Update(ctx context.Context, portfolio *models.Portfolio) error {
    qb := database.NewQueryBuilder()
    qb.AddParam("id", portfolio.ID)
    qb.AddParam("user_id", portfolio.UserID)
    qb.AddParam("name", portfolio.Name)
    qb.AddParam("description", portfolio.Description)
    qb.AddParam("balance", portfolio.Balance)
    qb.AddParam("risk", portfolio.Risk)
    qb.AddParam("strategy", portfolio.Strategy)

    query, args := qb.Build(`
        UPDATE portfolios 
        SET name = @name,
            description = @description,
            balance = @balance,
            risk = @risk,
            strategy = @strategy,
            updated_at = NOW()
        WHERE id = @id AND user_id = @user_id
    `)

    result, err := r.db.ExecSafe(ctx, query, args...)
    if err != nil {
        return fmt.Errorf("update portfolio: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("get rows affected: %w", err)
    }

    if rows == 0 {
        return fmt.Errorf("portfolio not found")
    }

    return nil
}

func (r *PortfolioRepository) Delete(ctx context.Context, id, userID int64) error {
    qb := database.NewQueryBuilder()
    qb.AddParam("id", id)
    qb.AddParam("user_id", userID)

    query, args := qb.Build(`
        DELETE FROM portfolios
        WHERE id = @id AND user_id = @user_id
    `)

    result, err := r.db.ExecSafe(ctx, query, args...)
    if err != nil {
        return fmt.Errorf("delete portfolio: %w", err)
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return fmt.Errorf("get rows affected: %w", err)
    }

    if rows == 0 {
        return fmt.Errorf("portfolio not found")
    }

    return nil
}