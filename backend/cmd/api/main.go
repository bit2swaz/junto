package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/bit2swaz/junto/internal/database"
	"github.com/bit2swaz/junto/internal/handlers"
	"github.com/bit2swaz/junto/internal/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
)

func main() {
	db, err := database.NewService()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	authHandler := &handlers.AuthHandler{DB: db}
	coupleHandler := &handlers.CoupleHandler{DB: db}

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/couples/code", coupleHandler.GeneratePairingCode)
		r.Post("/couples/link", coupleHandler.LinkPartner)
	})

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Fatal(err)
	}
}
