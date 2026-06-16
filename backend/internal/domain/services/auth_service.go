package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/acme/observability/internal/domain/entities"
	"github.com/acme/observability/internal/domain/repositories"
	"github.com/acme/observability/pkg/hash"
	"github.com/google/uuid"
)

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrRefreshRejected    = errors.New("refresh token rejected")
)

// AuthService implements registration, login, refresh and logout.
type AuthService struct {
	users  repositories.UserRepository
	tokens *TokenService
}

func NewAuthService(users repositories.UserRepository, tokens *TokenService) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

// TokenPair bundles the access + refresh tokens returned to clients.
type TokenPair struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	User         *entities.User `json:"user"`
}

func (s *AuthService) Register(ctx context.Context, email, name, password string) (*TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, ErrEmailTaken
	}

	pwHash, err := hash.Password(password)
	if err != nil {
		return nil, err
	}
	u := &entities.User{Email: email, Name: name, PasswordHash: pwHash}
	if err := s.users.CreateUser(ctx, u); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, u)
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}
	if !hash.Verify(u.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokens(ctx, u)
}

// Refresh validates a refresh token, rotates it and returns a new pair.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	userID, err := s.tokens.ParseRefresh(refreshToken)
	if err != nil {
		return nil, ErrRefreshRejected
	}
	stored, err := s.users.GetRefreshToken(ctx, hash.SHA256(refreshToken))
	if err != nil || stored.Revoked || stored.UserID != userID || time.Now().After(stored.ExpiresAt) {
		return nil, ErrRefreshRejected
	}
	// rotate: revoke the used token
	_ = s.users.RevokeRefreshToken(ctx, stored.ID)

	u, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrRefreshRejected
	}
	return s.issueTokens(ctx, u)
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	stored, err := s.users.GetRefreshToken(ctx, hash.SHA256(refreshToken))
	if err != nil {
		return nil // idempotent
	}
	return s.users.RevokeRefreshToken(ctx, stored.ID)
}

func (s *AuthService) Me(ctx context.Context, id uuid.UUID) (*entities.User, error) {
	return s.users.GetByID(ctx, id)
}

func (s *AuthService) issueTokens(ctx context.Context, u *entities.User) (*TokenPair, error) {
	access, err := s.tokens.GenerateAccess(u.ID, u.Email)
	if err != nil {
		return nil, err
	}
	refresh, err := s.tokens.GenerateRefresh(u.ID)
	if err != nil {
		return nil, err
	}
	rt := &entities.RefreshToken{
		UserID:    u.ID,
		TokenHash: hash.SHA256(refresh),
		ExpiresAt: time.Now().Add(s.tokens.RefreshTTL()),
	}
	if err := s.users.SaveRefreshToken(ctx, rt); err != nil {
		return nil, err
	}
	return &TokenPair{AccessToken: access, RefreshToken: refresh, User: u}, nil
}
