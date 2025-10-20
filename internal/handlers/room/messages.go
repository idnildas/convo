package room

import (
	"database/sql"
	"net/http"
	"strconv"

	"convo/internal/utils"
	"github.com/go-chi/chi/v5"
)

type RoomMessagesHandler struct {
	DB *sql.DB
}

// ServeHTTP handles GET /rooms/{id}/messages
func (h *RoomMessagesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
       // userID is not used directly, but JWT middleware ensures authentication
	roomIDStr := chi.URLParam(r, "id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "Invalid room id"})
		return
	}
	numStr := r.URL.Query().Get("num")
	num, err := strconv.Atoi(numStr)
	if err != nil || num <= 0 || num > 100 {
		utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "num param required (1-100)"})
		return
	}
	lastIDStr := r.URL.Query().Get("last_id")
	var rows *sql.Rows
       if lastIDStr != "" {
	       lastID, err := strconv.ParseInt(lastIDStr, 10, 64)
	       if err != nil {
		       utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "Invalid last_id"})
		       return
	       }
	       rows, err = h.DB.Query(`SELECT id, sender_id, content, sent_at FROM messages WHERE room_id = ? AND id < ? ORDER BY id DESC LIMIT ?`, roomID, lastID, num)
       } else {
	       rows, err = h.DB.Query(`SELECT id, sender_id, content, sent_at FROM messages WHERE room_id = ? ORDER BY id DESC LIMIT ?`, roomID, num)
       }
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "DB error", Data: map[string]interface{}{ "error": err.Error() }})
		return
	}
	defer rows.Close()

       type Message struct {
	       ID        int64  `json:"id"`
	       SenderID  int64  `json:"sender_id"`
	       Content   string `json:"content"`
	       SentAt    string `json:"sent_at"`
       }
       var messages []Message
       for rows.Next() {
	       var m Message
	       if err := rows.Scan(&m.ID, &m.SenderID, &m.Content, &m.SentAt); err != nil {
		       continue
	       }
	       messages = append(messages, m)
       }
	if len(messages) == 0 {
		utils.JSON(w, http.StatusOK, utils.APIResponse{Success: true, Message: "no history", Data: []Message{}})
		return
	}
	utils.JSON(w, http.StatusOK, utils.APIResponse{Success: true, Message: "messages fetched", Data: messages})
}
