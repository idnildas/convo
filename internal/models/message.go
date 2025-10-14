package models

import "time"

type Message struct {
	ID        int64     `json:"id"`
	RoomID    int64     `json:"room_id"`
	SenderID  int64     `json:"sender_id"`
	Content   string    `json:"content"`
	SentAt    time.Time `json:"sent_at"`
}
