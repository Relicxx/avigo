package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/Relicxx/avigo/internal/user"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo      user.Repository
	jwtSecret string
}

func NewService(repo user.Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

func (s *Service) generateToken(userID int64) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) Register(ctx context.Context, email, password string) (*user.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	u := user.User{Email: email, PasswordHash: string(hash)}

	if err := s.repo.Create(ctx, &u); err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	return &u, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", fmt.Errorf("error fetching user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", fmt.Errorf("invalid credentials: %w", err)
	}

	return s.generateToken(u.ID)
}
