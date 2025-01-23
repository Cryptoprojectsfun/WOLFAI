package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/Cryptoprojectsfun/quantai-clone/internal/models"
	"github.com/Cryptoprojectsfun/quantai-clone/internal/services"
)

type PortfolioHandler struct {
	service *services.PortfolioService
}

func NewPortfolioHandler(service *services.PortfolioService) *PortfolioHandler {
	return &PortfolioHandler{service: service}
}

func (h *PortfolioHandler) Create(w http.ResponseWriter, r *http.Request) {
	var portfolio models.Portfolio
	if err := json.NewDecoder(r.Body).Decode(&portfolio); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user := r.Context().Value("user").(*models.User)
	portfolio.UserID = user.ID

	if err := h.service.Create(&portfolio); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(portfolio)
}