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

    "github.com/Cryptoprojectsfun/quantai-clone/internal/api/handlers"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/middleware"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/services/portfolio"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/services/risk"
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
    portfolioService := portfolio.NewPortfolioService(db)
    portfolioAnalyzer := portfolio.NewPortfolioAnalyzer(db)
    portfolioOptimizer := portfolio.NewPortfolioOptimizer(db)
    riskManager := risk.NewRiskManager(db)

    // Initialize handlers
    portfolioHandler := handlers.NewPortfolioHandler(
        portfolioService,
        portfolioAnalyzer,
        portfolioOptimizer,
        riskManager,
    )

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
    api.HandleFunc("/auth/register", handlers.RegisterHandler).Methods("POST")
    api.HandleFunc("/auth/login", handlers.LoginHandler).Methods("POST")

    // Protected routes
    protected := api.PathPrefix("").Subrouter()
    protected.Use(authMiddleware.RequireAuth)

    // Portfolio routes
    protected.HandleFunc("/portfolios", portfolioHandler.CreatePortfolio).Methods("POST")
    protected.HandleFunc("/portfolios/{id}", portfolioHandler.GetPortfolio).Methods("GET")
    protected.HandleFunc("/portfolios/{id}/analyze", portfolioHandler.AnalyzePortfolio).Methods("GET")
    protected.HandleFunc("/portfolios/{id}/optimize", portfolioHandler.OptimizePortfolio).Methods("POST")
    protected.HandleFunc("/portfolios/{id}/risk", portfolioHandler.GetRiskMetrics).Methods("GET")

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
        DatabaseURL: getEnv("DATABASE_URL", "postgresql://localhost:5432/wolfai?sslmode=disable"),
        JWTSecret:   getEnv("JWT_SECRET", "your-secret-key"),
        RateLimit:   100,
        AllowedOrigins: []string{
            "http://localhost:3000",
            "https://wolfai.com",
        },
    }
}

func getEnv(key, fallback string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return fallback
}