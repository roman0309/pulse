package services

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims is the JWT payload for access tokens.
type Claims struct {
	UserID uuid.UUID `json:"uid"`
	Email  string    `json:"email"`
	jwt.RegisteredClaims
}

// TokenService issues and validates JWT access tokens and opaque refresh tokens.
type TokenService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewTokenService(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *TokenService {
	return &TokenService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

func (s *TokenService) RefreshTTL() time.Duration { return s.refreshTTL }

// GenerateAccess returns a signed short-lived access token.
func (s *TokenService) GenerateAccess(userID uuid.UUID, email string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.accessSecret)
}

// GenerateRefresh returns a signed long-lived refresh token.
func (s *TokenService) GenerateRefresh(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID.String(),
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(s.refreshTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.refreshSecret)
}

// ParseAccess validates an access token and returns its claims.
func (s *TokenService) ParseAccess(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.accessSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}
	return claims, nil
}

// ParseRefresh validates a refresh token and returns the subject user id.
func (s *TokenService) ParseRefresh(tokenStr string) (uuid.UUID, error) {
	claims := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.refreshSecret, nil
	})
	if err != nil || !token.Valid {
		return uuid.Nil, ErrInvalidToken
	}
	return uuid.Parse(claims.Subject)
}
