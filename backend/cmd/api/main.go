package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bit2swaz/junto/internal/database"
	"github.com/bit2swaz/junto/internal/handlers"
	"github.com/bit2swaz/junto/internal/middleware"
	"github.com/bit2swaz/junto/internal/websocket"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func main() {
	db, err := database.NewService()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	authHandler := &handlers.AuthHandler{DB: db}
	coupleHandler := &handlers.CoupleHandler{DB: db}
	hub := websocket.NewHub(db)

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Get("/me", authHandler.Me)
		r.Post("/couples/code", coupleHandler.GeneratePairingCode)
		r.Post("/couples/link", coupleHandler.LinkPartner)
		r.Get("/ws", hub.HandleWebSocket)
	})

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
