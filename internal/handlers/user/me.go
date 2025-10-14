package user

import (
    "database/sql"
    "net/http"

    "convo/internal/utils"
    "convo/internal/middleware"
)

type MeHandler struct {
    DB *sql.DB
}

func (h *MeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
    if !ok {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    var email, name string
    err := h.DB.QueryRow("SELECT email, name FROM users WHERE id=?", userID).Scan(&email, &name)
    if err == sql.ErrNoRows {
        http.Error(w, "User not found", http.StatusNotFound)
        return
    } else if err != nil {
        http.Error(w, "Database error", http.StatusInternalServerError)
        return
    }

    response := map[string]string{
        "email": email,
        "name":  name,
    }

    utils.JSON(w, http.StatusOK, utils.APIResponse{
        Success: true,
        Message: "User details retrieved successfully",
        Data:    response,
    })
}
