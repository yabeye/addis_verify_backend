package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims extends standard JWT claims with our custom fields
type Claims struct {
	AccountID string `json:"sub"`
	Type      string `json:"typ"` // "access" or "refresh"
	jwt.RegisteredClaims
}

type TokenDetails struct {
	AccessToken  string
	RefreshToken string
	AtExpires    int64
	RtExpires    int64
}

type TokenManager interface {
	GenerateTokenPair(id string, iat time.Time) (*TokenDetails, error)
	VerifyToken(token string) (*Claims, error)
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

func (m *jwtManager) GenerateTokenPair(accountID string, iat time.Time) (*TokenDetails, error) {
	td := &TokenDetails{}

	// 1. Set Expiry Times
	// Access Token: 15 minutes (Short-lived for security)
	td.AtExpires = time.Now().Add(time.Minute * 15).Unix()
	// Refresh Token: 7 days (Long-lived for UX)
	td.RtExpires = time.Now().Add(time.Hour * 24 * 7).Unix()

	// 2. Create Access Token
	atClaims := &Claims{
		AccountID: accountID,
		Type:      "access",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   accountID,
			Issuer:    m.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Unix(td.AtExpires, 0)),
			IssuedAt:  jwt.NewNumericDate(iat), // Must match DB token_valid_from
		},
	}
	at := jwt.NewWithClaims(jwt.SigningMethodHS256, atClaims)
	var err error
	td.AccessToken, err = at.SignedString(m.secretKey)
	if err != nil {
		return nil, err
	}

	// 3. Create Refresh Token
	rtClaims := &Claims{
		AccountID: accountID,
		Type:      "refresh",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   accountID,
			Issuer:    m.issuer,
			ExpiresAt: jwt.NewNumericDate(time.Unix(td.RtExpires, 0)),
			IssuedAt:  jwt.NewNumericDate(iat), // Also tied to the login version
		},
	}
	rt := jwt.NewWithClaims(jwt.SigningMethodHS256, rtClaims)
	td.RefreshToken, err = rt.SignedString(m.secretKey)
	if err != nil {
		return nil, err
	}

	return td, nil
}

func (m *jwtManager) VerifyToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
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
