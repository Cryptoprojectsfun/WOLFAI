package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
	"github.com/Cryptoprojectsfun/quantai-clone/internal/middleware"
	"github.com/Cryptoprojectsfun/quantai-clone/internal/services/ai"
	"github.com/Cryptoprojectsfun/quantai-clone/internal/services/analytics"
)

func main() {
	// Load configuration
	config := loadConfig()

	// Initialize database connection
	db, err := sql.Open("postgres", config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Initialize services
	aiService := ai.NewService(db)
	analyticsService := analytics.NewService(db, aiService)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(config.JWTSecret)

	// Create router
	router := mux.NewRouter()

	// Apply global middleware
	router.Use(middleware.Recovery)
	router.Use(middleware.RateLimit(config.RateLimit))
	router.Use(cors.New(cors.Options{
		AllowedOrigins:   config.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}).Handler)

	// API routes
	api := router.PathPrefix("/api/v1").Subrouter()

	// Public routes
	api.HandleFunc("/auth/register", handleRegister).Methods("POST")
	api.HandleFunc("/auth/login", handleLogin).Methods("POST")

	// Protected routes
	protected := api.PathPrefix("").Subrouter()
	protected.Use(authMiddleware.Authenticate)

	// Portfolio routes
	protected.HandleFunc("/portfolio", handleGetPortfolios).Methods("GET")
	protected.HandleFunc("/portfolio/{id}", handleGetPortfolio).Methods("GET")
	protected.HandleFunc("/portfolio", handleCreatePortfolio).Methods("POST")
	protected.HandleFunc("/portfolio/{id}", handleUpdatePortfolio).Methods("PUT")
	protected.HandleFunc("/portfolio/{id}", handleDeletePortfolio).Methods("DELETE")

	// Analytics routes
	protected.HandleFunc("/analytics/market/{symbol}", handleGetMarketAnalysis).Methods("GET")
	protected.HandleFunc("/analytics/predictions/{symbol}", handleGetPredictions).Methods("GET")
	protected.HandleFunc("/analytics/portfolio/{id}", handleGetPortfolioAnalytics).Methods("GET")

	// Admin routes
	admin := api.PathPrefix("/admin").Subrouter()
	admin.Use(authMiddleware.Authenticate)
	admin.Use(authMiddleware.RequireRole("admin"))

	admin.HandleFunc("/users", handleListUsers).Methods("GET")
	admin.HandleFunc("/system/status", handleSystemStatus).Methods("GET")

	// Create server
	srv := &http.Server{
		Addr:         ":" + config.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Printf("Server starting on port %s", config.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Server shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server stopped")
}

type Config struct {
	Port           string
	DatabaseURL    string
	JWTSecret      string
	RateLimit      int
	AllowedOrigins []string
}

func loadConfig() Config {
	return Config{
		Port:        getEnv("PORT", "8080"),
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://localhost:5432/quantai?sslmode=disable"),
		JWTSecret:   getEnv("JWT_SECRET", "your-secret-key"),
		RateLimit:   100,
		AllowedOrigins: []string{
			"http://localhost:3000",
			"https://quantai.com",
		},
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

// Handler functions (to be implemented)
func handleRegister(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleGetPortfolios(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleGetPortfolio(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleCreatePortfolio(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleUpdatePortfolio(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleDeletePortfolio(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleGetMarketAnalysis(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleGetPredictions(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleGetPortfolioAnalytics(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleListUsers(w http.ResponseWriter, r *http.Request) {
	// Implementation
}

func handleSystemStatus(w http.ResponseWriter, r *http.Request) {
	// Implementation
}