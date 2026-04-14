package jwt

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

// ValidateToken 验证 JWT token
func ValidateToken(tokenString, secret string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// ExtractUserID 从 token 中提取用户ID
func ExtractUserID(tokenString, secret string) (string, error) {
	claims, err := ValidateToken(tokenString, secret)
	if err != nil {
		return "", err
	}
	return claims.UserID, nil
}
