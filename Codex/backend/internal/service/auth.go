package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/gfl94/Forge-Storage/Codex/backend/internal/model"
	"github.com/gfl94/Forge-Storage/Codex/backend/internal/repository"
)

// AuthService handles user authentication and session lifecycle.
type AuthService struct {
	users    repository.UserRepository
	sessions repository.SessionRepository
	ttl      time.Duration
}

// NewAuthService constructs an AuthService.
func NewAuthService(users repository.UserRepository, sessions repository.SessionRepository, ttl time.Duration) *AuthService {
	return &AuthService{
		users:    users,
		sessions: sessions,
		ttl:      ttl,
	}
}

// EnsureDefaultAdmin seeds a default admin user if it does not exist.
func (a *AuthService) EnsureDefaultAdmin(ctx context.Context, username, password string) error {
	if username == "" || password == "" {
		return nil
	}
	existing, err := a.users.GetByUsername(ctx, username)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash default admin password: %w", err)
	}
	user := model.User{
		ID:           "u_" + uuid.NewString(),
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now().UTC(),
	}
	return a.users.Create(ctx, user)
}

// Login validates credentials and creates a session.
func (a *AuthService) Login(ctx context.Context, username, password string) (*model.User, *model.Session, error) {
	user, err := a.users.GetByUsername(ctx, username)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		return nil, nil, ErrUnauthorized
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, ErrUnauthorized
	}
	now := time.Now().UTC()
	session := model.Session{
		ID:         "s_" + uuid.NewString(),
		UserID:     user.ID,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  now.Add(a.ttl),
	}
	if err := a.sessions.CreateSession(ctx, session); err != nil {
		return nil, nil, err
	}
	return user, &session, nil
}

// ValidateSession returns the session and user if valid.
func (a *AuthService) ValidateSession(ctx context.Context, id string) (*model.User, *model.Session, error) {
	if id == "" {
		return nil, nil, ErrUnauthorized
	}
	session, err := a.sessions.GetSession(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if session == nil {
		return nil, nil, ErrUnauthorized
	}
	if time.Now().After(session.ExpiresAt) {
		_ = a.sessions.DeleteSession(ctx, id)
		return nil, nil, ErrUnauthorized
	}
	user, err := a.users.GetByID(ctx, session.UserID)
	if err != nil {
		return nil, nil, err
	}
	if user == nil {
		_ = a.sessions.DeleteSession(ctx, id)
		return nil, nil, ErrUnauthorized
	}
	_ = a.sessions.UpdateLastSeen(ctx, id, time.Now().UTC())
	return user, session, nil
}

// Logout removes a session.
func (a *AuthService) Logout(ctx context.Context, id string) error {
	if id == "" {
		return nil
	}
	return a.sessions.DeleteSession(ctx, id)
}

// ErrUnauthorized represents an authentication failure.
var ErrUnauthorized = errors.New("unauthorized")
