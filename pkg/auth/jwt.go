package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenManager interface {
	GenerateToken(accountID string) (string, error)
	ValidateToken(token string) (string, error)
}

type jwtManager struct {
	secretKey []byte
	issuer    string
}

func NewJWTManager(secret string) TokenManager {
	return &jwtManager{
		secretKey: []byte(secret),
		issuer:    "addis_verify",
	}
}

func (m *jwtManager) GenerateToken(accountID string) (string, error) {
	claims := jwt.MapClaims{
		"sub": accountID,                                 // Subject (User ID)
		"iss": m.issuer,                                  // Issuer
		"exp": time.Now().Add(time.Hour * 24 * 7).Unix(), // 7 days expiry
		"iat": time.Now().Unix(),                         // Issued at
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secretKey)
}

func (m *jwtManager) ValidateToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims["sub"].(string), nil
	}

	return "", fmt.Errorf("invalid token")
}
