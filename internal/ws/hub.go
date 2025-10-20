package ws

import (
    "encoding/json"
    "fmt"
    "sync"
    "time"

    "github.com/gorilla/websocket"
)

// Connection represents a websocket connection to a client
type Connection struct {
    Conn   *websocket.Conn
    Send   chan []byte
    UserID int64
    RoomID int64
}

// RoomHub maintains the set of active connections for a room and broadcasts messages
type RoomHub struct {
    RoomID    int64
    Conns     map[*Connection]bool
    Register   chan *Connection
    Unregister chan *Connection
    Broadcast  chan []byte
    mu         sync.Mutex
}

var (
    hubs   = make(map[int64]*RoomHub)
    hubsMu sync.Mutex
)

// GetRoomHub returns the hub for a room, creating it if necessary
func GetRoomHub(roomID int64) *RoomHub {
    hubsMu.Lock()
    defer hubsMu.Unlock()
    if h, ok := hubs[roomID]; ok {
        return h
    }
    h := &RoomHub{
        RoomID:     roomID,
        Conns:      make(map[*Connection]bool),
        Register:   make(chan *Connection),
        Unregister: make(chan *Connection),
        Broadcast:  make(chan []byte),
    }
    hubs[roomID] = h
    go h.run()
    return h
}

func (h *RoomHub) run() {
    ticker := time.NewTicker(time.Minute * 5)
    defer ticker.Stop()
    for {
        select {
        case c := <-h.Register:
            h.mu.Lock()
            h.Conns[c] = true
            h.mu.Unlock()
            fmt.Printf("user %d joined room %d\n", c.UserID, h.RoomID)
        case c := <-h.Unregister:
            h.mu.Lock()
            if _, ok := h.Conns[c]; ok {
                delete(h.Conns, c)
                close(c.Send)
            }
            h.mu.Unlock()
            fmt.Printf("user %d left room %d\n", c.UserID, h.RoomID)
        case msg := <-h.Broadcast:
            h.mu.Lock()
            for c := range h.Conns {
                select {
                case c.Send <- msg:
                default:
                    // If send buffer is full, drop connection
                    delete(h.Conns, c)
                    close(c.Send)
                }
            }
            h.mu.Unlock()
        case <-ticker.C:
            // periodic: clean closed connections
            h.mu.Lock()
            for c := range h.Conns {
                // noop for now; future health checks could be added
                _ = c
            }
            h.mu.Unlock()
        }
    }
}

// StartRead starts reading messages from the websocket and forwards them to the hub
func (c *Connection) StartRead(hub *RoomHub) {
    defer func() {
        hub.Unregister <- c
        c.Conn.Close()
    }()
    for {
        _, message, err := c.Conn.ReadMessage()
        if err != nil {
            // client disconnected or error
            return
        }

        // wrap the message with metadata
        envelope := map[string]interface{}{
            "type":    "message",
            "from":    c.UserID,
            "room_id": c.RoomID,
            "body":    json.RawMessage(message),
        }
        b, _ := json.Marshal(envelope)
        hub.Broadcast <- b
    }
}

// StartWrite writes messages from the Send channel to the websocket
func (c *Connection) StartWrite() {
    defer c.Conn.Close()
    for msg := range c.Send {
        if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
            return
        }
    }
}
