package chat

import (
	"context"
	"fmt"

	"github.com/Relicxx/avigo/internal/apperr"
)

var (
	// ErrSelfMessage — попытка написать самому себе.
	ErrSelfMessage = fmt.Errorf("%w: cannot message yourself", apperr.ErrInvalidInput)
	// ErrNoConversation — владелец может отвечать только тем, кто ему писал.
	ErrNoConversation = fmt.Errorf("%w: can reply only to users who messaged you", apperr.ErrForbidden)
	// ErrReceiverRequired — владелец обязан указать, кому отвечает.
	ErrReceiverRequired = fmt.Errorf("%w: receiver_id is required for the listing owner", apperr.ErrInvalidInput)
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Send отправляет сообщение по объявлению. Получатель определяется сервером:
// если пишет не владелец объявления — получателем всегда становится владелец
// (receiver_id от клиента игнорируется); владелец может отвечать только тем,
// кто уже писал ему по этому объявлению. Писать самому себе нельзя.
func (s *Service) Send(ctx context.Context, senderID, listingID, receiverID int64, body string) (*Message, error) {
	ownerID, err := s.repo.ListingOwner(ctx, listingID)
	if err != nil {
		return nil, err
	}

	if senderID != ownerID {
		// Покупатель всегда пишет владельцу; receiver_id от клиента не учитывается.
		receiverID = ownerID
	} else {
		if receiverID == 0 {
			return nil, ErrReceiverRequired
		}
		if receiverID == ownerID {
			return nil, ErrSelfMessage
		}
		ok, err := s.repo.HasMessageFrom(ctx, listingID, receiverID, ownerID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrNoConversation
		}
	}

	m := &Message{
		ListingID:  listingID,
		SenderID:   senderID,
		ReceiverID: receiverID,
		Body:       body,
	}
	if err := s.repo.Send(ctx, m); err != nil {
		return nil, err
	}
	return m, nil
}

// GetForUser возвращает переписку по объявлению, видимую пользователю userID.
func (s *Service) GetForUser(ctx context.Context, listingID, userID int64, limit, offset int) ([]*Message, error) {
	return s.repo.GetForUser(ctx, listingID, userID, limit, offset)
}
