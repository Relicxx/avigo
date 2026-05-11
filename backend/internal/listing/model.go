package listing

import "time"

type Listing struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Category    string    `json:"category"`
	IsBoosted   bool      `json:"is_boosted"`
	CreatedAt   time.Time `json:"created_at"`
}
