package validators

import (
    "github.com/Cryptoprojectsfun/quantai-clone/internal/middleware"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

type CreatePortfolioRequest struct {
    Name        string         `json:"name"`
    Description string         `json:"description"`
    RiskLevel   models.RiskLevel `json:"risk_level"`
    Balance     float64        `json:"balance"`
    Strategy    string         `json:"strategy"`
}

func (r *CreatePortfolioRequest) Validate() []middleware.ValidationError {
    var errors []middleware.ValidationError

    if len(r.Name) < 3 || len(r.Name) > 50 {
        errors = append(errors, middleware.ValidationError{
            Field:   "name",
            Message: "must be between 3 and 50 characters",
        })
    }

    if r.Balance < 0 {
        errors = append(errors, middleware.ValidationError{
            Field:   "balance",
            Message: "must be non-negative",
        })
    }

    switch r.RiskLevel {
    case models.LowRisk, models.MediumRisk, models.HighRisk:
    default:
        errors = append(errors, middleware.ValidationError{
            Field:   "risk_level",
            Message: "must be one of: low, medium, high",
        })
    }

    return errors
}

type OptimizePortfolioRequest struct {
    Symbols       []string `json:"symbols"`
    RiskTolerance float64  `json:"risk_tolerance"`
    TimeHorizon   int      `json:"time_horizon"`
}

func (r *OptimizePortfolioRequest) Validate() []middleware.ValidationError {
    var errors []middleware.ValidationError

    if len(r.Symbols) == 0 {
        errors = append(errors, middleware.ValidationError{
            Field:   "symbols",
            Message: "at least one symbol required",
        })
    }

    for _, symbol := range r.Symbols {
        if !isValidSymbol(symbol) {
            errors = append(errors, middleware.ValidationError{
                Field:   "symbols",
                Message: "invalid symbol format",
            })
            break
        }
    }

    if r.RiskTolerance < 0 || r.RiskTolerance > 1 {
        errors = append(errors, middleware.ValidationError{
            Field:   "risk_tolerance",
            Message: "must be between 0 and 1",
        })
    }

    if r.TimeHorizon < 1 || r.TimeHorizon > 365 {
        errors = append(errors, middleware.ValidationError{
            Field:   "time_horizon",
            Message: "must be between 1 and 365 days",
        })
    }

    return errors
}

func isValidSymbol(symbol string) bool {
    if len(symbol) == 0 || len(symbol) > 10 {
        return false
    }
    
    for _, char := range symbol {
        if !((char >= 'A' && char <= 'Z') || char == '.') {
            return false
        }
    }
    
    return true
}