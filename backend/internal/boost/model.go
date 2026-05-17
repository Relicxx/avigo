package boost

import "time"

type Boost struct {
	ID        int64     `json:"id"`
	ListingID int64     `json:"listing_id"`
	UserID    int64     `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}
