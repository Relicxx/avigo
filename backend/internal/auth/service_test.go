package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Relicxx/avigo/internal/apperr"
	"github.com/Relicxx/avigo/internal/user"
	"github.com/golang-jwt/jwt/v5"
)

type fakeUserRepo struct {
	users map[string]*user.User
	next  int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]*user.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, u *user.User) error {
	if _, ok := f.users[u.Email]; ok {
		return apperr.ErrConflict
	}
	f.next++
	u.ID = f.next
	u.CreatedAt = time.Now()
	f.users[u.Email] = u
	return nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*user.User, error) {
	u, ok := f.users[email]
	if !ok {
		return nil, apperr.ErrNotFound
	}
	return u, nil
}

type fakeTokenStore struct {
	tokens map[string]int64
}

func newFakeTokenStore() *fakeTokenStore {
	return &fakeTokenStore{tokens: map[string]int64{}}
}

func (f *fakeTokenStore) Save(_ context.Context, jti string, userID int64, _ time.Duration) error {
	f.tokens[jti] = userID
	return nil
}

func (f *fakeTokenStore) Consume(_ context.Context, jti string) (int64, error) {
	uid, ok := f.tokens[jti]
	if !ok {
		return 0, ErrInvalidToken
	}
	delete(f.tokens, jti)
	return uid, nil
}

func (f *fakeTokenStore) Delete(_ context.Context, jti string) error {
	delete(f.tokens, jti)
	return nil
}

func newTestService() *Service {
	return NewService(newFakeUserRepo(), newFakeTokenStore(), "test-secret")
}

func registerAndLogin(t *testing.T, s *Service) *Tokens {
	t.Helper()
	ctx := context.Background()
	if _, err := s.Register(ctx, "user@example.com", "password123"); err != nil {
		t.Fatalf("register: %v", err)
	}
	tokens, err := s.Login(ctx, "user@example.com", "password123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	return tokens
}

func TestRegisterDuplicateEmail(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	if _, err := s.Register(ctx, "dup@example.com", "password123"); err != nil {
		t.Fatalf("first register: %v", err)
	}
	_, err := s.Register(ctx, "dup@example.com", "password123")
	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("expected ErrConflict, got %v", err)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	s := newTestService()
	registerAndLogin(t, s)
	_, err := s.Login(context.Background(), "user@example.com", "wrong-password")
	if !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLoginUnknownEmail(t *testing.T) {
	s := newTestService()
	_, err := s.Login(context.Background(), "nobody@example.com", "password123")
	if !errors.Is(err, apperr.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRefreshRotation(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	tokens := registerAndLogin(t, s)

	fresh, err := s.Refresh(ctx, tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if fresh.RefreshToken == tokens.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	// Повторное использование старого (уже потреблённого) токена должно быть отклонено.
	if _, err := s.Refresh(ctx, tokens.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken on reuse, got %v", err)
	}

	// Новый токен всё ещё работает.
	if _, err := s.Refresh(ctx, fresh.RefreshToken); err != nil {
		t.Fatalf("refresh with rotated token: %v", err)
	}
}

func TestRefreshRejectsAccessToken(t *testing.T) {
	s := newTestService()
	tokens := registerAndLogin(t, s)
	_, err := s.Refresh(context.Background(), tokens.AccessToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for access token, got %v", err)
	}
}

func TestRefreshRejectsGarbage(t *testing.T) {
	s := newTestService()
	_, err := s.Refresh(context.Background(), "not-a-jwt")
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken, got %v", err)
	}
}

func TestRefreshRejectsForeignSecret(t *testing.T) {
	s := newTestService()
	other := NewService(newFakeUserRepo(), newFakeTokenStore(), "another-secret")
	tokens := registerAndLogin(t, other)

	_, err := s.Refresh(context.Background(), tokens.RefreshToken)
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for token signed with foreign secret, got %v", err)
	}
}

func TestRefreshRejectsTokenWithoutExpiry(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	tokens := registerAndLogin(t, s)

	// Достаём jti валидного refresh-токена и собираем токен без exp:
	// «вечный» refresh должен отклоняться, даже если jti существует.
	uid, jti, err := s.parseRefreshToken(tokens.RefreshToken)
	if err != nil {
		t.Fatalf("parse refresh: %v", err)
	}
	claims := jwt.MapClaims{"user_id": float64(uid), "type": "refresh", "jti": jti}
	eternal, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("test-secret"))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	if _, err := s.Refresh(ctx, eternal); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken for token without exp, got %v", err)
	}
}

func TestLogoutRevokesRefreshToken(t *testing.T) {
	s := newTestService()
	ctx := context.Background()
	tokens := registerAndLogin(t, s)

	if err := s.Logout(ctx, tokens.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	// Отозванный (утёкший) токен не должен работать.
	if _, err := s.Refresh(ctx, tokens.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("expected ErrInvalidToken after logout, got %v", err)
	}
}
