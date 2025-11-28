package tests

import (
	"context"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bit2swaz/junto/internal/database"
	"github.com/bit2swaz/junto/internal/handlers"
	"github.com/bit2swaz/junto/internal/middleware"
	wsInternal "github.com/bit2swaz/junto/internal/websocket"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupRouterWithWS(db database.Service) *chi.Mux {
	authHandler := &handlers.AuthHandler{DB: db}
	coupleHandler := &handlers.CoupleHandler{DB: db}
	hub := wsInternal.NewHub(db)

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)

	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/couples/code", coupleHandler.GeneratePairingCode)
		r.Post("/couples/link", coupleHandler.LinkPartner)
		r.Get("/ws", hub.HandleWebSocket)
	})

	return r
}

func TestHaptics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	r := setupRouterWithWS(db)
	ts := httptest.NewServer(r)
	defer ts.Close()

	client := ts.Client()
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	// 1. Setup Users
	userAEmail := "haptic_a@example.com"
	userAPass := "password123"
	registerUser(t, client, ts.URL, userAEmail, userAPass)
	tokenA := loginUser(t, client, ts.URL, userAEmail, userAPass)

	userBEmail := "haptic_b@example.com"
	userBPass := "password123"
	registerUser(t, client, ts.URL, userBEmail, userBPass)
	tokenB := loginUser(t, client, ts.URL, userBEmail, userBPass)

	// 2. Link Users
	code := generatePairingCode(t, client, ts.URL, tokenA)
	linkPartner(t, client, ts.URL, tokenB, code)

	// 3. Connect WebSockets
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Connect Client A
	connA, _, err := websocket.Dial(ctx, fmt.Sprintf("%s?token=%s", wsURL, tokenA), nil)
	require.NoError(t, err, "Client A failed to connect")
	defer connA.Close(websocket.StatusNormalClosure, "")

	// Connect Client B
	connB, _, err := websocket.Dial(ctx, fmt.Sprintf("%s?token=%s", wsURL, tokenB), nil)
	require.NoError(t, err, "Client B failed to connect")
	defer connB.Close(websocket.StatusNormalClosure, "")

	// 4. Test TOUCH_START
	t.Run("TOUCH_START", func(t *testing.T) {
		msg := map[string]interface{}{
			"type": "TOUCH_START",
		}
		err := wsjson.Write(ctx, connA, msg)
		require.NoError(t, err, "Client A failed to send TOUCH_START")

		// Client B should receive it
		var received map[string]interface{}
		err = wsjson.Read(ctx, connB, &received)
		require.NoError(t, err, "Client B failed to read message")
		assert.Equal(t, "TOUCH_START", received["type"])
	})

	// 5. Test TOUCH_END
	t.Run("TOUCH_END", func(t *testing.T) {
		msg := map[string]interface{}{
			"type": "TOUCH_END",
		}
		err := wsjson.Write(ctx, connA, msg)
		require.NoError(t, err, "Client A failed to send TOUCH_END")

		// Client B should receive it
		var received map[string]interface{}
		err = wsjson.Read(ctx, connB, &received)
		require.NoError(t, err, "Client B failed to read message")
		assert.Equal(t, "TOUCH_END", received["type"])
	})
}
