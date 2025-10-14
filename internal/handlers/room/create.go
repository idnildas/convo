package room

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "time"

    "convo/internal/middleware"
    "convo/internal/utils"
)

type CreateRoomHandler struct {
    DB *sql.DB
}

type CreateRoomRequest struct {
    Name string `json:"name"`
    OtherEmail string `json:"other_email,omitempty"`
}

type CreateRoomResponse struct {
    ID        int64     `json:"id"`
    Name      string    `json:"name"`
    CreatedBy int64     `json:"created_by"`
    CreatedAt time.Time `json:"created_at"`
}

func (h *CreateRoomHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var req CreateRoomRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "Invalid request body"})
        return
    }
    if req.Name == "" {
        utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "name is required"})
        return
    }

    tx, err := h.DB.Begin()
    if err != nil {
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "Failed to start tx", Data: map[string]interface{}{"error": err.Error()}})
        return
    }
    defer tx.Rollback()

    result, err := tx.Exec("INSERT INTO rooms (name, created_by) VALUES (?, ?)", req.Name, userID)
    if err != nil {
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "Failed to create room", Data: map[string]interface{}{"error": err.Error()}})
        return
    }
    id, _ := result.LastInsertId()

    // add creator to room_members
    if _, err := tx.Exec("INSERT INTO room_members (room_id, user_id) VALUES (?, ?)", id, userID); err != nil {
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "Failed to add creator to room", Data: map[string]interface{}{"error": err.Error()}})
        return
    }

    // optionally add other user by email
    if req.OtherEmail != "" {
        var otherID int64
        err := tx.QueryRow("SELECT id FROM users WHERE email = ?", req.OtherEmail).Scan(&otherID)
        if err == nil {
            // user exists, insert membership
            if _, err := tx.Exec("INSERT INTO room_members (room_id, user_id) VALUES (?, ?)", id, otherID); err != nil {
                // ignore duplicate membership errors
            }
        }
        // if user not found, we silently ignore — caller can invite later
    }

    // commit tx
    if err := tx.Commit(); err != nil {
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "Failed to commit tx", Data: map[string]interface{}{"error": err.Error()}})
        return
    }

    // fetch created_at
    var createdAt time.Time
    _ = h.DB.QueryRow("SELECT created_at FROM rooms WHERE id = ?", id).Scan(&createdAt)

    resp := CreateRoomResponse{
        ID:        id,
        Name:      req.Name,
        CreatedBy: userID,
        CreatedAt: createdAt,
    }

    utils.JSON(w, http.StatusCreated, utils.APIResponse{Success: true, Message: "Room created", Data: resp})
}
