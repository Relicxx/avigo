package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Relicxx/avigo/internal/apperr"
	"github.com/Relicxx/avigo/internal/user"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 7 * 24 * time.Hour
)

// ErrInvalidToken возвращается при невалидном или просроченном refresh-токене.
var ErrInvalidToken = errors.New("invalid refresh token")

// Tokens — пара токенов, выдаваемая при логине и обновлении.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Service struct {
	repo      user.Repository
	jwtSecret string
}

func NewService(repo user.Repository, jwtSecret string) *Service {
	return &Service{repo: repo, jwtSecret: jwtSecret}
}

func (s *Service) generateToken(userID int64, tokenType string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    tokenType,
		"exp":     time.Now().Add(ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(s.jwtSecret))
}

func (s *Service) generateTokens(userID int64) (*Tokens, error) {
	access, err := s.generateToken(userID, "access", accessTokenTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := s.generateToken(userID, "refresh", refreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &Tokens{AccessToken: access, RefreshToken: refresh}, nil
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

func (s *Service) Login(ctx context.Context, email, password string) (*Tokens, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			// Не раскрываем, существует ли email.
			return nil, apperr.ErrUnauthorized
		}
		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, apperr.ErrUnauthorized
	}

	return s.generateTokens(u.ID)
}

// Refresh проверяет refresh-токен и выдаёт новую пару токенов.
func (s *Service) Refresh(refreshToken string) (*Tokens, error) {
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		return nil, ErrInvalidToken
	}

	uid, ok := claims["user_id"].(float64)
	if !ok {
		return nil, ErrInvalidToken
	}

	return s.generateTokens(int64(uid))
}
