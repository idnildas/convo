package room

import (
    "database/sql"
    "net/http"
    "strconv"

    "github.com/go-chi/chi/v5"

    "convo/internal/middleware"
    "convo/internal/utils"
)

// RoomCheckHandler verifies if the authenticated user is a member of the room
type RoomCheckHandler struct{
    DB *sql.DB
}

// ServeHTTP handles GET /rooms/{id}/check
func (h *RoomCheckHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
    if !ok {
        utils.JSON(w, http.StatusUnauthorized, utils.APIResponse{Success: false, Message: "Unauthorized"})
        return
    }

    roomIDStr := chi.URLParam(r, "id")
    if roomIDStr == "" {
        utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "room id required in path"})
        return
    }
    roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
    if err != nil {
        utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "invalid room id"})
        return
    }

    var tmp int
    // check room exists
    if err := h.DB.QueryRow("SELECT 1 FROM rooms WHERE id = ?", roomID).Scan(&tmp); err == sql.ErrNoRows || err != nil {
        if err == sql.ErrNoRows {
            utils.JSON(w, http.StatusNotFound, utils.APIResponse{Success: false, Message: "room not found"})
            return
        }
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "DB error checking room", Data: map[string]interface{}{"error": err.Error()}})
        return
    }

    // check membership
    var exists int
    if err := h.DB.QueryRow("SELECT 1 FROM room_members WHERE room_id = ? AND user_id = ?", roomID, userID).Scan(&exists); err == sql.ErrNoRows {
        utils.JSON(w, http.StatusForbidden, utils.APIResponse{Success: false, Message: "user not a member of the room"})
        return
    } else if err != nil {
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "DB error checking membership", Data: map[string]interface{}{"error": err.Error()}})
        return
    }

    utils.JSON(w, http.StatusOK, utils.APIResponse{Success: true, Message: "user is a member", Data: map[string]interface{}{"room_id": roomID, "user_id": userID}})
}
