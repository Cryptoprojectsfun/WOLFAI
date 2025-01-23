package auth

import (
    "context"
    "database/sql"
    "time"
    "errors"

    "github.com/golang-jwt/jwt"
    "golang.org/x/crypto/bcrypt"
    "github.com/Cryptoprojectsfun/quantai-clone/internal/models"
)

type Service struct {
    db        *sql.DB
    jwtSecret []byte
}

func NewService(db *sql.DB, jwtSecret string) *Service {
    return &Service{
        db:        db,
        jwtSecret: []byte(jwtSecret),
    }
}

func (s *Service) Register(ctx context.Context, email, password, name string) error {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }

    query := `
        INSERT INTO users (email, password_hash, name)
        VALUES ($1, $2, $3)
    `
    _, err = s.db.ExecContext(ctx, query, email, hashedPassword, name)
    return err
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
    var user models.User
    query := `
        SELECT id, email, password_hash, name, role
        FROM users
        WHERE email = $1
    `
    err := s.db.QueryRowContext(ctx, query, email).Scan(
        &user.ID, &user.Email, &user.PasswordHash, &user.Name, &user.Role,
    )
    if err != nil {
        return "", err
    }

    if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
        return "", errors.New("invalid password")
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
        "user_id": user.ID,
        "email":   user.Email,
        "role":    user.Role,
        "exp":     time.Now().Add(24 * time.Hour).Unix(),
    })

    return token.SignedString(s.jwtSecret)
}

func (s *Service) ValidateToken(tokenString string) (*models.User, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, errors.New("invalid signing method")
        }
        return s.jwtSecret, nil
    })
    if err != nil {
        return nil, err
    }

    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        userID := int64(claims["user_id"].(float64))
        var user models.User
        
        query := `
            SELECT id, email, name, role
            FROM users
            WHERE id = $1
        `
        err := s.db.QueryRow(query, userID).Scan(
            &user.ID, &user.Email, &user.Name, &user.Role,
        )
        if err != nil {
            return nil, err
        }

        return &user, nil
    }

    return nil, errors.New("invalid token")
}
