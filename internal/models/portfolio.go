package models

import (
	"time"
)

type Portfolio struct {
	ID          int64      `json:"id" db:"id"`
	UserID      int64      `json:"user_id" db:"user_id"`
	Name        string     `json:"name" db:"name"`
	Description string     `json:"description" db:"description"`
	Balance     float64    `json:"balance" db:"balance"`
	Risk        RiskLevel  `json:"risk" db:"risk"`
	Strategy    string     `json:"strategy" db:"strategy"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at" db:"updated_at"`
	Positions   []Position `json:"positions,omitempty"`
}

type RiskLevel string

const (
	LowRisk    RiskLevel = "low"
	MediumRisk RiskLevel = "medium"
	HighRisk   RiskLevel = "high"
)

type Position struct {
	ID          int64     `json:"id" db:"id"`
	PortfolioID int64     `json:"portfolio_id" db:"portfolio_id"`
	Symbol      string    `json:"symbol" db:"symbol"`
	Quantity    float64   `json:"quantity" db:"quantity"`
	EntryPrice  float64   `json:"entry_price" db:"entry_price"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}