package chat

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Relicxx/avigo/internal/apperr"
)

// fakeChatRepo эмулирует SQL-фильтрацию репозитория в памяти.
type fakeChatRepo struct {
	owners map[int64]int64 // listingID -> ownerID
	msgs   []*Message
}

func newFakeChatRepo(owners map[int64]int64) *fakeChatRepo {
	return &fakeChatRepo{owners: owners}
}

func (f *fakeChatRepo) Send(_ context.Context, m *Message) error {
	m.ID = int64(len(f.msgs) + 1)
	m.CreatedAt = time.Now()
	f.msgs = append(f.msgs, m)
	return nil
}

func (f *fakeChatRepo) GetForUser(_ context.Context, listingID, userID int64, limit, offset int) ([]*Message, error) {
	out := []*Message{}
	for _, m := range f.msgs {
		if m.ListingID == listingID && (m.SenderID == userID || m.ReceiverID == userID) {
			out = append(out, m)
		}
	}
	if offset >= len(out) {
		return []*Message{}, nil
	}
	out = out[offset:]
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeChatRepo) HasMessageFrom(_ context.Context, listingID, senderID, receiverID int64) (bool, error) {
	for _, m := range f.msgs {
		if m.ListingID == listingID && m.SenderID == senderID && m.ReceiverID == receiverID {
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeChatRepo) ListingOwner(_ context.Context, listingID int64) (int64, error) {
	owner, ok := f.owners[listingID]
	if !ok {
		return 0, apperr.ErrNotFound
	}
	return owner, nil
}

const (
	ownerID  = int64(1)
	buyerID  = int64(2)
	otherID  = int64(3)
	listedID = int64(10)
)

func TestSendToMissingListingNotFound(t *testing.T) {
	s := NewService(newFakeChatRepo(map[int64]int64{}))
	_, err := s.Send(context.Background(), buyerID, 404, 0, "hi")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSendBuyerReceiverForcedToOwner(t *testing.T) {
	s := NewService(newFakeChatRepo(map[int64]int64{listedID: ownerID}))
	// Клиент подсовывает чужой receiver_id — сервер должен его игнорировать.
	m, err := s.Send(context.Background(), buyerID, listedID, otherID, "hi")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if m.ReceiverID != ownerID {
		t.Fatalf("receiver must be the listing owner, got %d", m.ReceiverID)
	}
}

func TestSendOwnerCannotStartConversation(t *testing.T) {
	s := NewService(newFakeChatRepo(map[int64]int64{listedID: ownerID}))
	_, err := s.Send(context.Background(), ownerID, listedID, buyerID, "hello?")
	if !errors.Is(err, ErrNoConversation) {
		t.Fatalf("expected ErrNoConversation, got %v", err)
	}
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("ErrNoConversation must map to ErrForbidden, got %v", err)
	}
}

func TestSendOwnerReplyToBuyer(t *testing.T) {
	repo := newFakeChatRepo(map[int64]int64{listedID: ownerID})
	s := NewService(repo)
	ctx := context.Background()

	if _, err := s.Send(ctx, buyerID, listedID, 0, "interested"); err != nil {
		t.Fatalf("buyer send: %v", err)
	}
	m, err := s.Send(ctx, ownerID, listedID, buyerID, "still available")
	if err != nil {
		t.Fatalf("owner reply: %v", err)
	}
	if m.ReceiverID != buyerID {
		t.Fatalf("expected receiver %d, got %d", buyerID, m.ReceiverID)
	}
}

func TestSendOwnerRequiresReceiver(t *testing.T) {
	s := NewService(newFakeChatRepo(map[int64]int64{listedID: ownerID}))
	_, err := s.Send(context.Background(), ownerID, listedID, 0, "hi")
	if !errors.Is(err, ErrReceiverRequired) {
		t.Fatalf("expected ErrReceiverRequired, got %v", err)
	}
}

func TestSendToSelfRejected(t *testing.T) {
	s := NewService(newFakeChatRepo(map[int64]int64{listedID: ownerID}))
	_, err := s.Send(context.Background(), ownerID, listedID, ownerID, "note to self")
	if !errors.Is(err, ErrSelfMessage) {
		t.Fatalf("expected ErrSelfMessage, got %v", err)
	}
}

func TestGetForUserHidesForeignConversation(t *testing.T) {
	repo := newFakeChatRepo(map[int64]int64{listedID: ownerID})
	s := NewService(repo)
	ctx := context.Background()

	if _, err := s.Send(ctx, buyerID, listedID, 0, "secret talk"); err != nil {
		t.Fatalf("send: %v", err)
	}

	// Посторонний пользователь не должен видеть чужую переписку.
	msgs, err := s.GetForUser(ctx, listedID, otherID, 50, 0)
	if err != nil {
		t.Fatalf("get for stranger: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("stranger must not see the conversation, got %d messages", len(msgs))
	}

	// Участники видят.
	for _, uid := range []int64{buyerID, ownerID} {
		msgs, err := s.GetForUser(ctx, listedID, uid, 50, 0)
		if err != nil {
			t.Fatalf("get for %d: %v", uid, err)
		}
		if len(msgs) != 1 {
			t.Fatalf("participant %d must see 1 message, got %d", uid, len(msgs))
		}
	}
}
