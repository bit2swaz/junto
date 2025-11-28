package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/bit2swaz/junto/internal/database"
	"github.com/bit2swaz/junto/internal/middleware"
)

type VaultHandler struct {
	DB database.Service
}

type CreateVaultItemRequest struct {
	Content  string    `json:"content"`
	UnlockAt time.Time `json:"unlock_at"`
}

func (h *VaultHandler) AddToVault(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	var req CreateVaultItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Get user to find couple_id
	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if user.CoupleID == nil {
		http.Error(w, "User is not in a couple", http.StatusBadRequest)
		return
	}

	item, err := h.DB.CreateVaultItem(r.Context(), *user.CoupleID, userID, req.Content, req.UnlockAt)
	if err != nil {
		http.Error(w, "Failed to create vault item", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(item)
}

func (h *VaultHandler) GetVaultItems(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	// Get user to find couple_id
	user, err := h.DB.GetUserByID(r.Context(), userID)
	if err != nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if user.CoupleID == nil {
		http.Error(w, "User is not in a couple", http.StatusBadRequest)
		return
	}

	items, err := h.DB.GetVaultItems(r.Context(), *user.CoupleID, userID)
	if err != nil {
		http.Error(w, "Failed to fetch vault items", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(items)
}
