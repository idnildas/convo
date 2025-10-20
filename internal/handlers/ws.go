package handlers

import (
       "database/sql"
       "net/http"
       "strconv"
       "strings"

       "convo/internal/ws"
       "convo/internal/utils"

       "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
       CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler upgrades HTTP to WebSocket, authenticates, checks room, and joins hub
func Handler(w http.ResponseWriter, r *http.Request) {
       // Parse query params
       q := r.URL.Query()
       roomIDStr := q.Get("room_id")
       token := q.Get("token")
       if roomIDStr == "" || token == "" {
	       http.Error(w, "room_id and token required", http.StatusBadRequest)
	       return
       }
       roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
       if err != nil {
	       http.Error(w, "invalid room_id", http.StatusBadRequest)
	       return
       }

       // Get JWT secret from env/config (for demo, use env)
       secret := utils.GetJWTSecret()
       userID, err := utils.ParseJWT(token, secret)
       if err != nil {
	       http.Error(w, "invalid token", http.StatusUnauthorized)
	       return
       }

       // Check DB membership (optional, but recommended)
       db := utils.GetDB()
       var exists int
       err = db.QueryRow("SELECT 1 FROM room_members WHERE room_id = ? AND user_id = ?", roomID, userID).Scan(&exists)
       if err == sql.ErrNoRows {
	       http.Error(w, "not a member of room", http.StatusForbidden)
	       return
       } else if err != nil {
	       http.Error(w, "db error", http.StatusInternalServerError)
	       return
       }

       // Upgrade to WebSocket
       conn, err := upgrader.Upgrade(w, r, nil)
       if err != nil {
	       http.Error(w, "Failed to upgrade to websocket", http.StatusInternalServerError)
	       return
       }

       // Register with room hub
       hub := ws.GetRoomHub(roomID)
       c := &ws.Connection{
	       Conn:   conn,
	       Send:   make(chan []byte, 256),
	       UserID: userID,
	       RoomID: roomID,
       }
       hub.Register <- c

       // Start read/write goroutines
       go c.StartWrite()
       c.StartRead(hub)
}
