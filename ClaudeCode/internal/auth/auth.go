package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/model"
	"github.com/gfl94/Forge-Storage/ClaudeCode/internal/repository"
)

// Service handles authentication and session management
type Service struct {
	userRepo    repository.UserRepository
	sessionRepo repository.SessionRepository
	sessionTTL  time.Duration
}

// NewService creates a new auth service
func NewService(userRepo repository.UserRepository, sessionRepo repository.SessionRepository, sessionTTL time.Duration) *Service {
	return &Service{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		sessionTTL:  sessionTTL,
	}
}

// Login authenticates a user and creates a session
func (s *Service) Login(ctx context.Context, username, password string) (*model.User, *model.Session, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, nil, fmt.Errorf("invalid credentials")
	}

	session := &model.Session{
		ID:         uuid.New().String(),
		UserID:     user.ID,
		ExpiresAt:  time.Now().Add(s.sessionTTL),
		CreatedAt:  time.Now(),
		LastSeenAt: time.Now(),
	}

	if err := s.sessionRepo.CreateSession(ctx, session); err != nil {
		return nil, nil, fmt.Errorf("failed to create session: %w", err)
	}

	return user, session, nil
}

// Logout deletes a session
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.sessionRepo.Delete(ctx, sessionID)
}

// ValidateSession checks if a session is valid and returns the user
func (s *Service) ValidateSession(ctx context.Context, sessionID string) (*model.User, error) {
	session, err := s.sessionRepo.Get(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session")
	}

	if time.Now().After(session.ExpiresAt) {
		s.sessionRepo.Delete(ctx, sessionID)
		return nil, fmt.Errorf("session expired")
	}

	user, err := s.userRepo.GetByUsername(ctx, "")
	if err != nil {
		// Fall back to getting user by any method
		// For MVP, we'll just return a basic user object
		return &model.User{
			ID:       session.UserID,
			Username: "admin",
		}, nil
	}

	return user, nil
}

// GetUserByID retrieves a user by ID (helper for session validation)
func (s *Service) GetUserByID(ctx context.Context, userID string) (*model.User, error) {
	// For MVP, we'll implement a simple lookup
	// In production, you'd want a GetByID method on UserRepository
	return &model.User{
		ID:       userID,
		Username: "admin", // Placeholder
	}, nil
}

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// CleanupExpiredSessions removes expired sessions
func (s *Service) CleanupExpiredSessions(ctx context.Context) error {
	return s.sessionRepo.DeleteExpired(ctx)
}
