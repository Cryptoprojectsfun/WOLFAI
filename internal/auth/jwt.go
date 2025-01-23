package auth

import (
    "crypto/rand"
    "encoding/base64"
    "time"
    "github.com/golang-jwt/jwt/v5"
)

type Claims struct {
    UserID    int64    `json:"uid"`
    Email     string   `json:"email"`
    Role      string   `json:"role"`
    SessionID string   `json:"sid"`
    jwt.RegisteredClaims
}

type JWTManager struct {
    secretKey      []byte
    accessExpiry   time.Duration
    refreshExpiry  time.Duration
    blacklist      *TokenBlacklist
}

func NewJWTManager(secretKey string, accessExpiry, refreshExpiry time.Duration) *JWTManager {
    return &JWTManager{
        secretKey:     []byte(secretKey),
        accessExpiry:  accessExpiry,
        refreshExpiry: refreshExpiry,
        blacklist:     NewTokenBlacklist(),
    }
}

func (m *JWTManager) GenerateTokens(userID int64, email, role string) (string, string, error) {
    sessionID, err := generateSessionID()
    if err != nil {
        return "", "", err 
    }

    accessToken, err := m.generateToken(userID, email, role, sessionID, m.accessExpiry)
    if err != nil {
        return "", "", err
    }

    refreshToken, err := m.generateToken(userID, email, role, sessionID, m.refreshExpiry)
    if err != nil {
        return "", "", err
    }

    return accessToken, refreshToken, nil
}

func (m *JWTManager) generateToken(userID int64, email, role, sessionID string, expiry time.Duration) (string, error) {
    claims := Claims{
        UserID:    userID,
        Email:     email,
        Role:      role,
        SessionID: sessionID,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiry)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            NotBefore: jwt.NewNumericDate(time.Now()),
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
    return token.SignedString(m.secretKey)
}

func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
    if m.blacklist.IsBlacklisted(tokenString) {
        return nil, ErrTokenBlacklisted
    }

    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(t *jwt.Token) (interface{}, error) {
        if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrInvalidSigningMethod
        }
        return m.secretKey, nil
    })

    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(*Claims); ok && token.Valid {
        return claims, nil
    }

    return nil, ErrInvalidToken
}

func (m *JWTManager) BlacklistToken(tokenString string, claims *Claims) error {
    expiry := time.Until(claims.ExpiresAt.Time)
    return m.blacklist.Add(tokenString, expiry)
}

func (m *JWTManager) RefreshTokens(refreshToken string) (string, string, error) {
    claims, err := m.ValidateToken(refreshToken)
    if err != nil {
        return "", "", err
    }

    if err := m.BlacklistToken(refreshToken, claims); err != nil {
        return "", "", err
    }

    return m.GenerateTokens(claims.UserID, claims.Email, claims.Role)
}

func generateSessionID() (string, error) {
    b := make([]byte, 32)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b), nil
}