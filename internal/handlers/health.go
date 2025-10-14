package handlers

import (
    "database/sql"
    "encoding/json"
    "net/http"
)

type HealthHandler struct {
    DB *sql.DB
}

// function wrapper that works directly with chi
func HealthCheck(w http.ResponseWriter, r *http.Request) {
    response := map[string]string{
        "status": "ok",
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}
