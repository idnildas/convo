package models

import "time"

type Room struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedBy int64     `json:"created_by"` // User ID
	CreatedAt time.Time `json:"created_at"`
}
