package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/gorilla/mux"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/models"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/services/portfolio"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/services/risk"
)

type PortfolioHandler struct {
    portfolioService *portfolio.PortfolioService
    analyzer        *portfolio.PortfolioAnalyzer
    optimizer       *portfolio.PortfolioOptimizer
    riskManager     *risk.RiskManager
}

func NewPortfolioHandler(
    ps *portfolio.PortfolioService,
    pa *portfolio.PortfolioAnalyzer,
    po *portfolio.PortfolioOptimizer,
    rm *risk.RiskManager,
) *PortfolioHandler {
    return &PortfolioHandler{
        portfolioService: ps,
        analyzer:        pa,
        optimizer:       po,
        riskManager:     rm,
    }
}

func (h *PortfolioHandler) CreatePortfolio(w http.ResponseWriter, r *http.Request) {
    var portfolio models.Portfolio
    if err := json.NewDecoder(r.Body).Decode(&portfolio); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    user := r.Context().Value("user").(*models.User)
    portfolio.UserID = user.ID

    if err := h.portfolioService.Create(r.Context(), &portfolio); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(portfolio)
}

func (h *PortfolioHandler) GetPortfolio(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
        return
    }

    user := r.Context().Value("user").(*models.User)
    portfolio, err := h.portfolioService.Get(r.Context(), id, user.ID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(portfolio)
}

func (h *PortfolioHandler) AnalyzePortfolio(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
        return
    }

    user := r.Context().Value("user").(*models.User)
    metrics, err := h.analyzer.AnalyzePortfolio(r.Context(), id, user.ID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(metrics)
}

func (h *PortfolioHandler) OptimizePortfolio(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
        return
    }

    var params struct {
        RiskTolerance float64 `json:"risk_tolerance"`
    }
    if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }

    user := r.Context().Value("user").(*models.User)
    portfolio, err := h.portfolioService.Get(r.Context(), id, user.ID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    symbols := make([]string, len(portfolio.Positions))
    for i, pos := range portfolio.Positions {
        symbols[i] = pos.Symbol
    }

    result, err := h.optimizer.Optimize(r.Context(), symbols, params.RiskTolerance)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(result)
}

func (h *PortfolioHandler) GetRiskMetrics(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.ParseInt(vars["id"], 10, 64)
    if err != nil {
        http.Error(w, "Invalid portfolio ID", http.StatusBadRequest)
        return
    }

    user := r.Context().Value("user").(*models.User)
    portfolio, err := h.portfolioService.Get(r.Context(), id, user.ID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusNotFound)
        return
    }

    metrics, err := h.riskManager.AnalyzeRisk(r.Context(), portfolio.ID)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    json.NewEncoder(w).Encode(metrics)
}
