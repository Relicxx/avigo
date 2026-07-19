package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func signHS256(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return token
}

func authRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", Auth(testSecret), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetInt64("user_id")})
	})
	return r
}

func doRequest(r *gin.Engine, authHeader string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func validClaims(tokenType string) jwt.MapClaims {
	return jwt.MapClaims{
		"user_id": float64(42),
		"type":    tokenType,
		"exp":     time.Now().Add(time.Hour).Unix(),
	}
}

func TestAuthMissingToken(t *testing.T) {
	if w := doRequest(authRouter(), ""); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthGarbageToken(t *testing.T) {
	if w := doRequest(authRouter(), "Bearer garbage"); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthWrongSecret(t *testing.T) {
	token := signHS256(t, "wrong-secret", validClaims("access"))
	if w := doRequest(authRouter(), "Bearer "+token); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAuthRejectsNoneAlgorithm(t *testing.T) {
	token, err := jwt.NewWithClaims(jwt.SigningMethodNone, validClaims("access")).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign none token: %v", err)
	}
	if w := doRequest(authRouter(), "Bearer "+token); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for alg=none, got %d", w.Code)
	}
}

func TestAuthRejectsRefreshTokenAsAccess(t *testing.T) {
	token := signHS256(t, testSecret, validClaims("refresh"))
	if w := doRequest(authRouter(), "Bearer "+token); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for refresh token, got %d", w.Code)
	}
}

func TestAuthExpiredToken(t *testing.T) {
	claims := validClaims("access")
	claims["exp"] = time.Now().Add(-time.Minute).Unix()
	token := signHS256(t, testSecret, claims)
	if w := doRequest(authRouter(), "Bearer "+token); w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for expired token, got %d", w.Code)
	}
}

func TestAuthValidToken(t *testing.T) {
	token := signHS256(t, testSecret, validClaims("access"))
	w := doRequest(authRouter(), "Bearer "+token)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
	}
	if body := w.Body.String(); body != `{"user_id":42}` {
		t.Fatalf("unexpected body: %s", body)
	}
}
