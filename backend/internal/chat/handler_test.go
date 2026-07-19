package chat

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// chatRouter поднимает роуты чата с подменённой аутентификацией:
// user_id берётся из тестового заголовка X-Test-User.
func chatRouter(repo Repository) *gin.Engine {
	gin.SetMode(gin.TestMode)
	h := NewHandler(NewService(repo))

	r := gin.New()
	r.Use(func(c *gin.Context) {
		var uid int64
		fmt.Sscanf(c.GetHeader("X-Test-User"), "%d", &uid)
		c.Set("user_id", uid)
		c.Next()
	})
	r.POST("/messages", h.Send)
	r.GET("/listings/:id/messages", h.GetByListing)
	return r
}

func getMessages(t *testing.T, r *gin.Engine, listingID int64, asUser string) (int, []*Message) {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/listings/%d/messages", listingID), nil)
	req.Header.Set("X-Test-User", asUser)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var msgs []*Message
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &msgs); err != nil {
			t.Fatalf("unmarshal response: %v (%s)", err, w.Body.String())
		}
		if msgs == nil {
			t.Fatal("empty result must be [], not null")
		}
	}
	return w.Code, msgs
}

// Регрессионный тест приватности: посторонний пользователь
// не должен видеть чужую переписку по объявлению.
func TestGetMessagesPrivacy(t *testing.T) {
	repo := newFakeChatRepo(map[int64]int64{listedID: ownerID})
	r := chatRouter(repo)

	// Покупатель пишет владельцу.
	body := fmt.Sprintf(`{"listing_id":%d,"body":"is it available?"}`, listedID)
	req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User", "2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("send: expected 201, got %d (%s)", w.Code, w.Body.String())
	}

	// Посторонний пользователь получает пустой список.
	code, msgs := getMessages(t, r, listedID, "3")
	if code != http.StatusOK || len(msgs) != 0 {
		t.Fatalf("stranger must get 200 with [], got %d with %d messages", code, len(msgs))
	}

	// Участники видят сообщение.
	for _, uid := range []string{"1", "2"} {
		code, msgs := getMessages(t, r, listedID, uid)
		if code != http.StatusOK || len(msgs) != 1 {
			t.Fatalf("user %s must see 1 message, got code %d, %d messages", uid, code, len(msgs))
		}
	}
}

func TestSendMissingListingReturns404(t *testing.T) {
	r := chatRouter(newFakeChatRepo(map[int64]int64{}))

	req := httptest.NewRequest(http.MethodPost, "/messages",
		strings.NewReader(`{"listing_id":404,"body":"hi"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Test-User", "2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d (%s)", w.Code, w.Body.String())
	}
}

func TestSendValidation(t *testing.T) {
	r := chatRouter(newFakeChatRepo(map[int64]int64{listedID: ownerID}))

	for name, body := range map[string]string{
		"empty body":      fmt.Sprintf(`{"listing_id":%d,"body":""}`, listedID),
		"missing listing": `{"body":"hi"}`,
		"too long body":   fmt.Sprintf(`{"listing_id":%d,"body":"%s"}`, listedID, strings.Repeat("a", 2001)),
	} {
		req := httptest.NewRequest(http.MethodPost, "/messages", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Test-User", "2")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Fatalf("%s: expected 400, got %d", name, w.Code)
		}
	}
}
