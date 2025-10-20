package room

import (
	"database/sql"
	"net/http"

	"convo/internal/middleware"
	"convo/internal/utils"
)

type RoomListHandler struct {
	DB *sql.DB
}

// ServeHTTP handles GET /rooms
func (h *RoomListHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		utils.JSON(w, http.StatusUnauthorized, utils.APIResponse{Success: false, Message: "Unauthorized"})
		return
	}

	rows, err := h.DB.Query(`SELECT r.id, r.name, r.created_by, r.created_at FROM rooms r
		JOIN room_members m ON r.id = m.room_id WHERE m.user_id = ?`, userID)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "DB error", Data: map[string]interface{}{ "error": err.Error() }})
		return
	}
	defer rows.Close()

	type Room struct {
		ID        int64  `json:"id"`
		Name      string `json:"name"`
		CreatedBy int64  `json:"created_by"`
		CreatedAt string `json:"created_at"`
	}
	var rooms []Room
	for rows.Next() {
		var r Room
		if err := rows.Scan(&r.ID, &r.Name, &r.CreatedBy, &r.CreatedAt); err != nil {
			continue
		}
		rooms = append(rooms, r)
	}
	utils.JSON(w, http.StatusOK, utils.APIResponse{Success: true, Message: "rooms fetched", Data: rooms})
}
