package room

import (
    "database/sql"
    "encoding/json"
    "net/http"
    "strings"
    "strconv"
    "fmt"

    mysql "github.com/go-sql-driver/mysql"
    "github.com/go-chi/chi/v5"

    "convo/internal/middleware"
    "convo/internal/utils"
)

type AddMembersRequest struct {
    IDs    string `json:"ids,omitempty"`    // comma-separated IDs
    Emails string `json:"emails,omitempty"` // comma-separated emails
}

type AddMembersResponse struct {
    AddedBy int64 `json:"added_by"`
    RoomID  int64 `json:"room_id"`
}

// ServeHTTP handles POST /rooms/{id}/members
func (h *AddMembersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    // parse room id from chi URL param
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

    var req AddMembersRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.JSON(w, http.StatusBadRequest, utils.APIResponse{Success: false, Message: "Invalid request body"})
        return
    }

    // ensure room exists
    var tmp int
    if err := h.DB.QueryRow("SELECT 1 FROM rooms WHERE id = ?", roomID).Scan(&tmp); err == sql.ErrNoRows || err != nil {
        if err == sql.ErrNoRows {
            utils.JSON(w, http.StatusNotFound, utils.APIResponse{Success: false, Message: "room not found"})
            return
        }
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "DB error checking room", Data: map[string]interface{}{"error": err.Error()}})
        return
    }

    tx, err := h.DB.Begin()
    if err != nil {
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "Failed to start tx", Data: map[string]interface{}{"error": err.Error()}})
        return
    }
    defer tx.Rollback()

    // collect errors
    var errs []string

    // priority: IDs then emails
    if strings.TrimSpace(req.IDs) != "" {
        for _, s := range strings.Split(req.IDs, ",") {
            s = strings.TrimSpace(s)
            if s == "" {
                continue
            }
            idVal, err := strconv.ParseInt(s, 10, 64)
            if err != nil {
                // skip invalid id
                continue
            }
            if _, err := tx.Exec("INSERT INTO room_members (room_id, user_id) VALUES (?, ?)", roomID, idVal); err != nil {
                // if duplicate key, ignore; otherwise record error
                if me, ok := err.(*mysql.MySQLError); ok {
                    if me.Number == 1062 {
                        // duplicate entry, ignore
                        continue
                    }
                }
                errs = append(errs, fmt.Sprintf("id %d: %v", idVal, err))
            }
        }
    } else if strings.TrimSpace(req.Emails) != "" {
        for _, e := range strings.Split(req.Emails, ",") {
            e = strings.TrimSpace(e)
            if e == "" {
                continue
            }
            var id int64
            if err := tx.QueryRow("SELECT id FROM users WHERE email = ?", e).Scan(&id); err == nil {
                if _, err := tx.Exec("INSERT INTO room_members (room_id, user_id) VALUES (?, ?)", roomID, id); err != nil {
                    if me, ok := err.(*mysql.MySQLError); ok {
                        if me.Number == 1062 {
                            continue
                        }
                    }
                    errs = append(errs, fmt.Sprintf("email %s: %v", e, err))
                }
            } else if err == sql.ErrNoRows {
                // user not found: record as info
                errs = append(errs, fmt.Sprintf("email %s: not found", e))
            } else {
                errs = append(errs, fmt.Sprintf("email %s: %v", e, err))
            }
        }
    }

    if err := tx.Commit(); err != nil {
        utils.JSON(w, http.StatusInternalServerError, utils.APIResponse{Success: false, Message: "Failed to commit tx", Data: map[string]interface{}{"error": err.Error()}})
        return
    }

    if len(errs) > 0 {
        utils.JSON(w, http.StatusOK, utils.APIResponse{Success: true, Message: "Members added with errors", Data: map[string]interface{}{"added_by": userID, "room_id": roomID, "errors": errs}})
        return
    }

    utils.JSON(w, http.StatusOK, utils.APIResponse{Success: true, Message: "Members added", Data: AddMembersResponse{AddedBy: userID, RoomID: roomID}})
}

// small type to attach DB
type AddMembersHandler struct{
    DB *sql.DB
}
