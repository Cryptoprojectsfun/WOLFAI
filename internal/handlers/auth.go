package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Cryptoprojectsfun/quantai-clone/internal/models"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userService    UserService
	authMiddleware AuthMiddleware
}

type UserService interface {
	CreateUser(user *models.User) error
	GetUserByEmail(email string) (*models.User, error)
	UpdateLastLogin(userID uuid.UUID) error
}

type AuthMiddleware interface {
	GenerateToken(userID uuid.UUID, role string) (string, error)
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	Token     string       `json:"token"`
	User      *models.User `json:"user"`
	ExpiresAt time.Time   `json:"expires_at"`
}

func NewAuthHandler(userService UserService, authMiddleware AuthMiddleware) *AuthHandler {
	return &AuthHandler{
		userService:    userService,
		authMiddleware: authMiddleware,
	}
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate request
	if err := validateRegisterRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "Error processing request", http.StatusInternalServerError)
		return
	}

	// Create user
	user := &models.User{
		ID:             uuid.New(),
		Email:          req.Email,
		HashedPassword: string(hashedPassword),
		Name:           req.Name,
		Role:           "user",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := h.userService.CreateUser(user); err != nil {
		if err.Error() == "user already exists" {
			http.Error(w, "Email already registered", http.StatusConflict)
			return
		}
		http.Error(w, "Error creating user", http.StatusInternalServerError)
		return
	}

	// Generate JWT token
	token, err := h.authMiddleware.GenerateToken(user.ID, user.Role)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// Create response
	res := AuthResponse{
		Token:     token,
		User:      user,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user by email
	user, err := h.userService.GetUserByEmail(req.Email)
	if err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.HashedPassword), []byte(req.Password)); err != nil {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Generate JWT token
	token, err := h.authMiddleware.GenerateToken(user.ID, user.Role)
	if err != nil {
		http.Error(w, "Error generating token", http.StatusInternalServerError)
		return
	}

	// Update last login
	if err := h.userService.UpdateLastLogin(user.ID); err != nil {
		// Log error but don't fail the request
		// logger.Error("Failed to update last login", "error", err)
	}

	// Create response
	res := AuthResponse{
		Token:     token,
		User:      user,
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}

func validateRegisterRequest(req RegisterRequest) error {
	if req.Email == "" {
		return ErrInvalidEmail
	}
	if len(req.Password) < 8 {
		return ErrPasswordTooShort
	}
	if req.Name == "" {
		return ErrInvalidName
	}
	return nil
}

// Custom errors
var (
	ErrInvalidEmail     = NewValidationError("invalid email")
	ErrPasswordTooShort = NewValidationError("password must be at least 8 characters")
	ErrInvalidName      = NewValidationError("invalid name")
)

type ValidationError struct {
	Message string
}

func (e ValidationError) Error() string {
	return e.Message
}

func NewValidationError(message string) ValidationError {
	return ValidationError{Message: message}
}