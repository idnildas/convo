package utils

import (
	"os"
	"time"
	"database/sql"
	"convo/internal/database"
	"github.com/golang-jwt/jwt/v5"
)

// GetJWTSecret returns the JWT secret from env
func GetJWTSecret() string {
	return os.Getenv("JWT_SECRET")
}

// GetDB returns the global DB connection
func GetDB() *sql.DB {
	return database.GetDB()
}

func GenerateJWT(userID int64, secret string, ttlHours int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Duration(ttlHours) * time.Hour).Unix(),
		"iat":     time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseJWT returns user id from token string if valid
func ParseJWT(tokenStr string, secret string) (int64, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return 0, jwt.ErrTokenMalformed
	}
	userIDFloat, ok := claims["user_id"].(float64)
	if !ok {
		return 0, jwt.ErrTokenMalformed
	}
	return int64(userIDFloat), nil
}
