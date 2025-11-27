package websocket

import (
	"context"
	"log"
	"net/http"
	"sync"

	"github.com/bit2swaz/junto/internal/database"
	"github.com/bit2swaz/junto/internal/middleware"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type Hub struct {
	// Map coupleID -> list of userIDs
	rooms   map[int64][]int64
	roomsMu sync.RWMutex

	// Map userID -> connection
	conns   map[int64]*websocket.Conn
	connsMu sync.RWMutex

	db database.Service
}

func NewHub(db database.Service) *Hub {
	return &Hub{
		rooms: make(map[int64][]int64),
		conns: make(map[int64]*websocket.Conn),
		db:    db,
	}
}

func (h *Hub) Add(userID int64, coupleID *int64, conn *websocket.Conn) {
	h.connsMu.Lock()
	if oldConn, ok := h.conns[userID]; ok {
		oldConn.Close(websocket.StatusNormalClosure, "New connection replaced this one")
	}
	h.conns[userID] = conn
	h.connsMu.Unlock()

	if coupleID != nil {
		h.roomsMu.Lock()
		h.rooms[*coupleID] = append(h.rooms[*coupleID], userID)
		h.roomsMu.Unlock()
	}
}

func (h *Hub) Remove(userID int64, coupleID *int64) {
	h.connsMu.Lock()
	delete(h.conns, userID)
	h.connsMu.Unlock()

	if coupleID != nil {
		h.roomsMu.Lock()
		defer h.roomsMu.Unlock()
		users := h.rooms[*coupleID]
		for i, uid := range users {
			if uid == userID {
				h.rooms[*coupleID] = append(users[:i], users[i+1:]...)
				break
			}
		}
		if len(h.rooms[*coupleID]) == 0 {
			delete(h.rooms, *coupleID)
		}
	}
}

func (h *Hub) BroadcastToCouple(coupleID int64, message interface{}, excludeUserID int64) {
	h.roomsMu.RLock()
	userIDs := h.rooms[coupleID]
	h.roomsMu.RUnlock()

	for _, uid := range userIDs {
		if uid == excludeUserID {
			continue
		}

		h.connsMu.RLock()
		conn, ok := h.conns[uid]
		h.connsMu.RUnlock()

		if ok {
			go func(c *websocket.Conn) {
				err := wsjson.Write(context.Background(), c, message)
				if err != nil {
					log.Printf("failed to write to websocket: %v", err)
				}
			}(conn)
		}
	}
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// 1. Authenticate
	userIDVal := r.Context().Value(middleware.UserIDKey)
	if userIDVal == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	userID, ok := userIDVal.(int64)
	if !ok {
		http.Error(w, "Invalid user ID", http.StatusUnauthorized)
		return
	}

	// Fetch user to get couple_id
	user, err := h.db.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to fetch user", http.StatusInternalServerError)
		return
	}

	// 2. Upgrade
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{"*"}, // Allow all origins for now
	})
	if err != nil {
		log.Printf("failed to accept websocket connection: %v", err)
		return
	}

	// 3. Register
	h.Add(userID, user.CoupleID, c)
	log.Printf("User %d connected via WebSocket", userID)

	// 4. Listen (Keep connection open)
	ctx := r.Context()
	defer func() {
		h.Remove(userID, user.CoupleID)
		c.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		// Read loop
		var msg map[string]interface{}
		err := wsjson.Read(ctx, c, &msg)
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return
			}
			log.Printf("failed to read from websocket: %v", err)
			return
		}

		// Handle messages
		if msgType, ok := msg["type"].(string); ok {
			switch msgType {
			case "move":
				if user.CoupleID != nil {
					h.BroadcastToCouple(*user.CoupleID, msg, userID)
				}
			}
		}
	}
}
