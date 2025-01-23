package middleware

import (
    "net/http"
    "strings"
    "crypto/subtle"
)

type SecurityHeaders struct {
    CSPDirectives    []string
    TrustedProxies  []string
    AllowedOrigins  []string
}

func Security(opts SecurityHeaders) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Content Security Policy
            csp := strings.Join(opts.CSPDirectives, "; ")
            w.Header().Set("Content-Security-Policy", csp)

            // Security headers
            w.Header().Set("X-Content-Type-Options", "nosniff")
            w.Header().Set("X-Frame-Options", "DENY")
            w.Header().Set("X-XSS-Protection", "1; mode=block")
            w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
            w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
            w.Header().Set("Permissions-Policy", "accelerometer=(), camera=(), geolocation=(), gyroscope=(), magnetometer=(), microphone=(), payment=(), usb=()")

            // CORS
            origin := r.Header.Get("Origin")
            if origin != "" {
                // Check against allowed origins
                for _, allowed := range opts.AllowedOrigins {
                    if origin == allowed {
                        w.Header().Set("Access-Control-Allow-Origin", origin)
                        break
                    }
                }
            }

            // Real IP handling
            realIP := r.Header.Get("X-Real-IP")
            if realIP != "" {
                // Validate proxy
                remoteIP := strings.Split(r.RemoteAddr, ":")[0]
                for _, trusted := range opts.TrustedProxies {
                    if subtle.ConstantTimeCompare([]byte(remoteIP), []byte(trusted)) == 1 {
                        r.RemoteAddr = realIP
                        break
                    }
                }
            }

            next.ServeHTTP(w, r)
        })
    }
}

func SecureHeaders() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            // Remove sensitive headers
            w.Header().Del("Server")
            w.Header().Del("X-Powered-By")

            // Add security headers
            w.Header().Set("X-Content-Type-Options", "nosniff")
            w.Header().Set("X-Frame-Options", "DENY")
            w.Header().Set("X-XSS-Protection", "1; mode=block")
            
            next.ServeHTTP(w, r)
        })
    }
}

func IPFilter(allowedIPs []string) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            ip := strings.Split(r.RemoteAddr, ":")[0]
            allowed := false

            for _, allowedIP := range allowedIPs {
                if subtle.ConstantTimeCompare([]byte(ip), []byte(allowedIP)) == 1 {
                    allowed = true
                    break
                }
            }

            if !allowed {
                http.Error(w, "Forbidden", http.StatusForbidden)
                return
            }

            next.ServeHTTP(w, r)
        })
    }
}

func TLSRedirect(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("X-Forwarded-Proto") != "https" {
            sslUrl := "https://" + r.Host + r.RequestURI
            http.Redirect(w, r, sslUrl, http.StatusMovedPermanently)
            return
        }
        next.ServeHTTP(w, r)
    })
}