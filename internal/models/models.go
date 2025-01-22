package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	Email           string     `json:"email" db:"email"`
	HashedPassword  string     `json:"-" db:"hashed_password"`
	Name            string     `json:"name" db:"name"`
	Role            string     `json:"role" db:"role"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at" db:"updated_at"`
	LastLoginAt     *time.Time `json:"last_login_at" db:"last_login_at"`
	SubscriptionTier string    `json:"subscription_tier" db:"subscription_tier"`
}

type Portfolio struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	UserID        uuid.UUID    `json:"user_id" db:"user_id"`
	Name          string       `json:"name" db:"name"`
	Description   string       `json:"description" db:"description"`
	TotalValue    float64      `json:"total_value" db:"total_value"`
	Assets        []Asset      `json:"assets" db:"assets"`
	Performance   Performance  `json:"performance" db:"performance"`
	RiskScore     float64      `json:"risk_score" db:"risk_score"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
}

type Asset struct {
	Symbol     string    `json:"symbol" db:"symbol"`
	Type       string    `json:"type" db:"type"`
	Quantity   float64   `json:"quantity" db:"quantity"`
	Value      float64   `json:"value" db:"value"`
	LastUpdate time.Time `json:"last_update" db:"last_update"`
}

type Performance struct {
	DailyReturn   float64   `json:"daily_return" db:"daily_return"`
	WeeklyReturn  float64   `json:"weekly_return" db:"weekly_return"`
	MonthlyReturn float64   `json:"monthly_return" db:"monthly_return"`
	YearlyReturn  float64   `json:"yearly_return" db:"yearly_return"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type Prediction struct {
	ID            uuid.UUID    `json:"id" db:"id"`
	AssetSymbol   string       `json:"asset_symbol" db:"asset_symbol"`
	Timeframe     string       `json:"timeframe" db:"timeframe"`
	PredictedHigh float64      `json:"predicted_high" db:"predicted_high"`
	PredictedLow  float64      `json:"predicted_low" db:"predicted_low"`
	Confidence    float64      `json:"confidence" db:"confidence"`
	Indicators    []Indicator  `json:"indicators" db:"indicators"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	ValidUntil    time.Time    `json:"valid_until" db:"valid_until"`
}

type Indicator struct {
	Name   string  `json:"name" db:"name"`
	Value  float64 `json:"value" db:"value"`
	Weight float64 `json:"weight" db:"weight"`
}

type MarketAnalysis struct {
	ID             uuid.UUID   `json:"id" db:"id"`
	AssetSymbol    string      `json:"asset_symbol" db:"asset_symbol"`
	Sentiment      float64     `json:"sentiment" db:"sentiment"`
	Volume24h      float64     `json:"volume_24h" db:"volume_24h"`
	PriceChange24h float64     `json:"price_change_24h" db:"price_change_24h"`
	TrendStrength  float64     `json:"trend_strength" db:"trend_strength"`
	UpdatedAt      time.Time   `json:"updated_at" db:"updated_at"`
	Signals        []Signal    `json:"signals" db:"signals"`
}

type Signal struct {
	Type        string    `json:"type" db:"type"`
	Strength    float64   `json:"strength" db:"strength"`
	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}