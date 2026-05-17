package chat

import "time"

type Message struct {
	ID         int64     `json:"id"`
	ListingID  int64     `json:"listing_id"`
	SenderID   int64     `json:"sender_id"`
	ReceiverID int64     `json:"receiver_id"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
}
