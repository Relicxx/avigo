package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

// ErrInvalidToken возвращается при невалидном, просроченном или отозванном refresh-токене.
var ErrInvalidToken = fmt.Errorf("%w: invalid refresh token", apperr.ErrUnauthorized)

// Tokens — пара токенов, выдаваемая при логине и обновлении.
type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type Service struct {
	repo      user.Repository
	tokens    TokenStore
	jwtSecret string
}

func NewService(repo user.Repository, tokens TokenStore, jwtSecret string) *Service {
	return &Service{repo: repo, tokens: tokens, jwtSecret: jwtSecret}
}

func newJTI() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate jti: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func (s *Service) generateToken(userID int64, tokenType, jti string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"type":    tokenType,
		"exp":     time.Now().Add(ttl).Unix(),
	}
	if jti != "" {
		claims["jti"] = jti
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(s.jwtSecret))
}

// generateTokens выдаёт новую пару токенов и регистрирует refresh в хранилище.
func (s *Service) generateTokens(ctx context.Context, userID int64) (*Tokens, error) {
	access, err := s.generateToken(userID, "access", "", accessTokenTTL)
	if err != nil {
		return nil, err
	}

	jti, err := newJTI()
	if err != nil {
		return nil, err
	}
	refresh, err := s.generateToken(userID, "refresh", jti, refreshTokenTTL)
	if err != nil {
		return nil, err
	}
	if err := s.tokens.Save(ctx, jti, userID, refreshTokenTTL); err != nil {
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
		if errors.Is(err, apperr.ErrConflict) {
			return nil, apperr.ErrConflict
		}
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

	return s.generateTokens(ctx, u.ID)
}

// parseRefreshToken валидирует подпись и claims refresh-токена,
// возвращая userID и jti.
func (s *Service) parseRefreshToken(refreshToken string) (int64, string, error) {
	token, err := jwt.Parse(refreshToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(s.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		return 0, "", ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || claims["type"] != "refresh" {
		return 0, "", ErrInvalidToken
	}

	uid, ok := claims["user_id"].(float64)
	if !ok {
		return 0, "", ErrInvalidToken
	}
	jti, ok := claims["jti"].(string)
	if !ok || jti == "" {
		return 0, "", ErrInvalidToken
	}

	return int64(uid), jti, nil
}

// Refresh проверяет refresh-токен и выдаёт новую пару токенов с ротацией:
// использованный токен инвалидируется, повторное использование (в т.ч.
// утёкшего и уже отозванного токена) отклоняется.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*Tokens, error) {
	uid, jti, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return nil, err
	}

	storedUID, err := s.tokens.Consume(ctx, jti)
	if err != nil {
		return nil, err
	}
	if storedUID != uid {
		return nil, ErrInvalidToken
	}

	return s.generateTokens(ctx, uid)
}

// Logout отзывает refresh-токен: после выхода им нельзя обновить сессию.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	_, jti, err := s.parseRefreshToken(refreshToken)
	if err != nil {
		return err
	}
	return s.tokens.Delete(ctx, jti)
}
