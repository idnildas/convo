package room

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "time"
    "strconv"

    "github.com/go-chi/chi/v5"
    "convo/internal/middleware"
    "convo/internal/utils"
)

type SendMessageRequest struct {
    Content string `json:"content"`
}

type SendMessageResponse struct {
    ID       int64     `json:"id"`
    RoomID   int64     `json:"room_id"`
    SenderID int64     `json:"sender_id"`
    Content  string    `json:"content"`
    SentAt   time.Time `json:"sent_at"`
}

type SendMessageHandler struct{
    DB *sql.DB
}

func (h *SendMessageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    roomIDStr := chi.URLParam(r, "id")
    roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
    if err != nil {
        utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "invalid room id"})
        return
    }

    // check membership
    var found int
    if err := h.DB.QueryRow("SELECT 1 FROM room_members WHERE room_id=? AND user_id=?", roomID, userID).Scan(&found); err != nil {
        if err == sql.ErrNoRows {
            utils.JSON(w, http.StatusForbidden, utils.APIResponse{Success: false, Message: "not a member of room"})
            return
        }
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "DB error checking membership", Data: map[string]interface{}{"error": err.Error()}})
        return
    }

    var req SendMessageRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "invalid request"})
        return
    }
    if req.Content == "" {
        utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "content required"})
        return
    }

    result, err := h.DB.Exec("INSERT INTO messages (room_id, sender_id, content) VALUES (?, ?, ?)", roomID, userID, req.Content)
    if err != nil {
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "failed to insert message", Data: map[string]interface{}{"error": err.Error()}})
        return
    }
    id, _ := result.LastInsertId()

    var sentAt time.Time
    _ = h.DB.QueryRow("SELECT sent_at FROM messages WHERE id = ?", id).Scan(&sentAt)

    // populate message_meta for all current room members (including sender)
    // insert ignoring duplicates
    _, _ = h.DB.Exec(`
        INSERT INTO message_meta (message_id, user_id, status, forwarded, starred, reactions, extra)
        SELECT ?, rm.user_id, 'sent', FALSE, FALSE, NULL, NULL FROM room_members rm WHERE rm.room_id = ?
        ON DUPLICATE KEY UPDATE message_id = message_id
    `, id, roomID)

    resp := SendMessageResponse{
        ID: id,
        RoomID: roomID,
        SenderID: userID,
        Content: req.Content,
        SentAt: sentAt,
    }

    utils.JSON(w, http.StatusCreated, utils.APIResponse{Success: true, Message: "Message sent", Data: resp})
}
