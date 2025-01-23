package middleware

import (
    "encoding/json"
    "net/http"
    "errors"
)

type ErrorResponse struct {
    Error   string `json:"error"`
    Code    int    `json:"code"`
    Details string `json:"details,omitempty"`
}

var (
    ErrInvalidInput     = errors.New("invalid input")
    ErrUnauthorized     = errors.New("unauthorized")
    ErrForbidden        = errors.New("forbidden")
    ErrNotFound         = errors.New("resource not found")
    ErrInternalServer   = errors.New("internal server error")
)

func ErrorHandler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        response := &ErrorResponse{}
        var err error

        defer func() {
            if rec := recover(); rec != nil {
                switch v := rec.(type) {
                case error:
                    err = v
                default:
                    err = ErrInternalServer
                }
            }
            if err != nil {
                w.Header().Set("Content-Type", "application/json")
                switch {
                case errors.Is(err, ErrInvalidInput):
                    response.Code = http.StatusBadRequest
                case errors.Is(err, ErrUnauthorized):
                    response.Code = http.StatusUnauthorized
                case errors.Is(err, ErrForbidden):
                    response.Code = http.StatusForbidden
                case errors.Is(err, ErrNotFound):
                    response.Code = http.StatusNotFound
                default:
                    response.Code = http.StatusInternalServerError
                }
                response.Error = err.Error()
                w.WriteHeader(response.Code)
                json.NewEncoder(w).Encode(response)
                return
            }
        }()

        next.ServeHTTP(w, r)
    })
}
