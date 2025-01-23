package middleware

import (
    "context"
    "net/http"
    "strings"

    "github.com/Cryptoprojectsfun/quantai-clone/internal/auth"
)

type AuthMiddleware struct {
    authService *auth.Service
}

func NewAuthMiddleware(as *auth.Service) *AuthMiddleware {
    return &AuthMiddleware{authService: as}
}

func (m *AuthMiddleware) RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        authHeader := r.Header.Get("Authorization")
        if authHeader == "" {
            http.Error(w, "Authorization header required", http.StatusUnauthorized)
            return
        }

        bearerToken := strings.Split(authHeader, " ")
        if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
            http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
            return
        }

        user, err := m.authService.ValidateToken(bearerToken[1])
        if err != nil {
            http.Error(w, "Invalid token", http.StatusUnauthorized)
            return
        }

        ctx := context.WithValue(r.Context(), "user", user)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func (m *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            user := r.Context().Value("user")
            if user == nil {
                http.Error(w, "Unauthorized", http.StatusUnauthorized)
                return
            }

            if user.(*auth.User).Role != role {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}