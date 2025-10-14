package models

import "time"

type RoomMember struct {
	RoomID   int64     `json:"room_id"`
	UserID   int64     `json:"user_id"`
	JoinedAt time.Time `json:"joined_at"`
}
