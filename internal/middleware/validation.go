package middleware

import (
    "context"
    "encoding/json"
    "net/http"
    "strings"
)

type ValidationError struct {
    Field   string `json:"field"`
    Message string `json:"message"`
}

type Validator interface {
    Validate() []ValidationError
}

func ValidateRequest(v Validator) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if errors := v.Validate(); len(errors) > 0 {
                w.Header().Set("Content-Type", "application/json")
                w.WriteHeader(http.StatusBadRequest)
                json.NewEncoder(w).Encode(map[string]interface{}{
                    "errors": errors,
                })
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

func ContentTypeJSON(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method != http.MethodGet && r.Method != http.MethodDelete {
            contentType := r.Header.Get("Content-Type")
            if !strings.Contains(contentType, "application/json") {
                http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
                return
            }
        }
        next.ServeHTTP(w, r)
    })
}

func MaxBodySize(size int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            if r.ContentLength > size {
                http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
                return
            }
            next.ServeHTTP(w, r)
        })
    }
}

func ValidateSymbols(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        symbols := r.URL.Query()["symbols"]
        if len(symbols) == 0 {
            http.Error(w, "At least one symbol required", http.StatusBadRequest)
            return
        }
        
        for _, symbol := range symbols {
            if !isValidSymbol(symbol) {
                http.Error(w, "Invalid symbol format", http.StatusBadRequest)
                return
            }
        }
        
        ctx := context.WithValue(r.Context(), "symbols", symbols)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
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