package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/google/uuid"
	"github.com/Cryptoprojectsfun/quantai-clone/internal/models"
	"github.com/Cryptoprojectsfun/quantai-clone/internal/middleware"
)

// Custom errors for portfolio operations
var (
	ErrInvalidPortfolioName = NewValidationError("invalid portfolio name")
	ErrEmptyPortfolio      = NewValidationError("portfolio must contain at least one asset")
	ErrInvalidAssetSymbol  = NewValidationError("invalid asset symbol")
	ErrInvalidAssetQuantity = NewValidationError("invalid asset quantity")
)

type PortfolioHandler struct {
	portfolioService PortfolioService
	analyticsService AnalyticsService
}

type PortfolioService interface {
	GetPortfolio(ctx context.Context, id uuid.UUID) (*models.Portfolio, error)
	GetUserPortfolios(ctx context.Context, userID uuid.UUID) ([]*models.Portfolio, error)
	CreatePortfolio(ctx context.Context, portfolio *models.Portfolio) error
	UpdatePortfolio(ctx context.Context, portfolio *models.Portfolio) error
	DeletePortfolio(ctx context.Context, id uuid.UUID) error
	UpdatePortfolioValue(ctx context.Context, portfolio *models.Portfolio) error
}

type AnalyticsService interface {
	GetAdvancedAnalytics(ctx context.Context, portfolioID string) (*models.AdvancedAnalytics, error)
	GetMarketAnalysis(ctx context.Context, symbol string) (*models.MarketAnalysis, error)
}

type CreatePortfolioRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Assets      []models.Asset `json:"assets"`
}

type UpdatePortfolioRequest struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Assets      []models.Asset `json:"assets"`
}

type PortfolioResponse struct {
	*models.Portfolio
	Analytics *models.AdvancedAnalytics `json:"analytics,omitempty"`
}

func NewPortfolioHandler(portfolioService PortfolioService, analyticsService AnalyticsService) *PortfolioHandler {
	return &PortfolioHandler{
		portfolioService: portfolioService,
		analyticsService: analyticsService,
	}
}

func (h *PortfolioHandler) GetPortfolios(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get portfolios
	portfolios, err := h.portfolioService.GetUserPortfolios(r.Context(), userID)
	if err != nil {
		http.Error(w, "Error fetching portfolios", http.StatusInternalServerError)
		return
	}

	// For each portfolio, update current values and get analytics
	var response []PortfolioResponse
	for _, portfolio := range portfolios {
		// Update portfolio value
		if err := h.portfolioService.UpdatePortfolioValue(r.Context(), portfolio); err != nil {
			// Log error but continue
			// logger.Error("Failed to update portfolio value", "error", err)
		}

		// Get analytics
		analytics, err := h.analyticsService.GetAdvancedAnalytics(r.Context(), portfolio.ID.String())
		if err != nil {
			// Log error but continue
			// logger.Error("Failed to get portfolio analytics", "error", err)
		}

		response = append(response, PortfolioResponse{
			Portfolio:  portfolio,
			Analytics: analytics,
		})
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
	// Get portfolio ID from URL
	vars := mux.Vars(r)
	portfolioID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	// Get portfolio
	portfolio, err := h.portfolioService.GetPortfolio(r.Context(), portfolioID)
	if err != nil {
		http.Error(w, "Error fetching portfolio", http.StatusInternalServerError)
		return
	}

	// Check if portfolio belongs to user
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok || portfolio.UserID != userID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Update portfolio value
	if err := h.portfolioService.UpdatePortfolioValue(r.Context(), portfolio); err != nil {
		// Log error but continue
		// logger.Error("Failed to update portfolio value", "error", err)
	}

	// Get analytics
	analytics, err := h.analyticsService.GetAdvancedAnalytics(r.Context(), portfolio.ID.String())
	if err != nil {
		// Log error but continue
		// logger.Error("Failed to get portfolio analytics", "error", err)
	}

	response := PortfolioResponse{
		Portfolio:  portfolio,
		Analytics: analytics,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *PortfolioHandler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
	// Get user ID from context
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req CreatePortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validateCreatePortfolioRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create portfolio model
	portfolio := &models.Portfolio{
		ID:          uuid.New(),
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Assets:      req.Assets,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Update portfolio value
	if err := h.portfolioService.UpdatePortfolioValue(r.Context(), portfolio); err != nil {
		http.Error(w, "Error calculating portfolio value", http.StatusInternalServerError)
		return
	}

	// Create portfolio
	if err := h.portfolioService.CreatePortfolio(r.Context(), portfolio); err != nil {
		http.Error(w, "Error creating portfolio", http.StatusInternalServerError)
		return
	}

	// Get initial analytics
	analytics, err := h.analyticsService.GetAdvancedAnalytics(r.Context(), portfolio.ID.String())
	if err != nil {
		// Log error but continue
		// logger.Error("Failed to get portfolio analytics", "error", err)
	}

	response := PortfolioResponse{
		Portfolio:  portfolio,
		Analytics: analytics,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

func (h *PortfolioHandler) UpdatePortfolio(w http.ResponseWriter, r *http.Request) {
	// Get portfolio ID from URL
	vars := mux.Vars(r)
	portfolioID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	// Get existing portfolio
	portfolio, err := h.portfolioService.GetPortfolio(r.Context(), portfolioID)
	if err != nil {
		http.Error(w, "Error fetching portfolio", http.StatusInternalServerError)
		return
	}

	// Check if portfolio belongs to user
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok || portfolio.UserID != userID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse request
	var req UpdatePortfolioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update portfolio
	portfolio.Name = req.Name
	portfolio.Description = req.Description
	portfolio.Assets = req.Assets
	portfolio.UpdatedAt = time.Now()

	// Update portfolio value
	if err := h.portfolioService.UpdatePortfolioValue(r.Context(), portfolio); err != nil {
		http.Error(w, "Error calculating portfolio value", http.StatusInternalServerError)
		return
	}

	// Save changes
	if err := h.portfolioService.UpdatePortfolio(r.Context(), portfolio); err != nil {
		http.Error(w, "Error updating portfolio", http.StatusInternalServerError)
		return
	}

	// Get updated analytics
	analytics, err := h.analyticsService.GetAdvancedAnalytics(r.Context(), portfolio.ID.String())
	if err != nil {
		// Log error but continue
		// logger.Error("Failed to get portfolio analytics", "error", err)
	}

	response := PortfolioResponse{
		Portfolio:  portfolio,
		Analytics: analytics,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *PortfolioHandler) DeletePortfolio(w http.ResponseWriter, r *http.Request) {
	// Get portfolio ID from URL
	vars := mux.Vars(r)
	portfolioID, err := uuid.Parse(vars["id"])
	if err != nil {
		http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
		return
	}

	// Get existing portfolio
	portfolio, err := h.portfolioService.GetPortfolio(r.Context(), portfolioID)
	if err != nil {
		http.Error(w, "Error fetching portfolio", http.StatusInternalServerError)
		return
	}

	// Check if portfolio belongs to user
	userID, ok := r.Context().Value(middleware.UserIDKey).(uuid.UUID)
	if !ok || portfolio.UserID != userID {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Delete portfolio
	if err := h.portfolioService.DeletePortfolio(r.Context(), portfolioID); err != nil {
		http.Error(w, "Error deleting portfolio", http.StatusInternalServerError)
		return
	}

	// Send response
	w.WriteHeader(http.StatusNoContent)
}

func validateCreatePortfolioRequest(req CreatePortfolioRequest) error {
	if req.Name == "" {
		return ErrInvalidPortfolioName
	}
	if len(req.Assets) == 0 {
		return ErrEmptyPortfolio
	}
	
	// Validate each asset
	for _, asset := range req.Assets {
		if asset.Symbol == "" {
			return ErrInvalidAssetSymbol
		}
		if asset.Quantity <= 0 {
			return ErrInvalidAssetQuantity
		}
	}
	
	return nil
}
