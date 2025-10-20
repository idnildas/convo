package handlers

import (
       "database/sql"
       "encoding/json"
       "net/http"
       "strconv"
       "time"

       "convo/internal/ws"
       "convo/internal/utils"

       "github.com/gorilla/websocket"
)
// WSMessage is the envelope for all websocket messages
type WSMessage struct {
       Type      string `json:"type"`
       RoomID    int64  `json:"room_id"`
       Content   string `json:"content,omitempty"`
       // Add more fields as needed
}

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

       // Start write goroutine
       go c.StartWrite()

       // Read loop: handle different message types
       for {
              _, msg, err := conn.ReadMessage()
              if err != nil {
                     break // disconnect
              }
              var wsmsg WSMessage
              if err := json.Unmarshal(msg, &wsmsg); err != nil {
                     sendError(c, "invalid message format")
                     continue
              }
              switch wsmsg.Type {
              case "send_message":
                     // Save to DB (reuse send-message handler logic)
                     if wsmsg.Content == "" {
                            sendError(c, "content required")
                            continue
                     }
                     // Insert into DB (use sender_id)
                     _, err := db.Exec("INSERT INTO messages (room_id, sender_id, content, sent_at) VALUES (?, ?, ?, ?)", roomID, userID, wsmsg.Content, time.Now())
                     if err != nil {
                            sendError(c, "db error sending message")
                            continue
                     }
                     // Broadcast to room
                       wsmsg.RoomID = roomID
                       wsmsg.Type = "message" // outgoing type
                       b, _ := json.Marshal(wsmsg)
                       hub.Broadcast <- b
              case "read":
                     // Optionally: mark as read in DB, or just acknowledge
                     // For now, just send ack
                     sendAck(c, "read received")
              case "join":
                     // Already joined on connect, but can send ack
                     sendAck(c, "joined room")
              case "leave":
                     // Unregister and close
                     hub.Unregister <- c
                     return
              default:
                     sendError(c, "unknown message type")
              }
       }
       // On disconnect
       hub.Unregister <- c
       conn.Close()
}

func sendError(c *ws.Connection, msg string) {
       m := map[string]interface{}{"type": "error", "message": msg}
       b, _ := json.Marshal(m)
       c.Send <- b
}

func sendAck(c *ws.Connection, msg string) {
       m := map[string]interface{}{"type": "ack", "message": msg}
       b, _ := json.Marshal(m)
       c.Send <- b
}
