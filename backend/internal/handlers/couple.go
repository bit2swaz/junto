package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/bit2swaz/junto/internal/database"
	"github.com/bit2swaz/junto/internal/middleware"
)

type CoupleHandler struct {
	DB database.Service
}

type LinkPartnerRequest struct {
	Code string `json:"code"`
}

func (h *CoupleHandler) GeneratePairingCode(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(int64)

	// Generate 6-digit code
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := fmt.Sprintf("%06d", rng.Intn(1000000))

	// Store in Redis with 10 minute expiration
	key := fmt.Sprintf("pairing:%s", code)
	err := h.DB.GetRedis().Set(r.Context(), key, userID, 10*time.Minute).Err()
	if err != nil {
		http.Error(w, "Failed to generate code", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"code": code,
	})
}

func (h *CoupleHandler) LinkPartner(w http.ResponseWriter, r *http.Request) {
	currentUserID := r.Context().Value(middleware.UserIDKey).(int64)

	var req LinkPartnerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Retrieve partner ID from Redis
	key := fmt.Sprintf("pairing:%s", req.Code)
	val, err := h.DB.GetRedis().Get(r.Context(), key).Result()
	if err != nil {
		http.Error(w, "Invalid or expired code", http.StatusBadRequest)
		return
	}

	partnerID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if partnerID == currentUserID {
		http.Error(w, "Cannot link with yourself", http.StatusBadRequest)
		return
	}

	// Create couple
	couple, err := h.DB.CreateCouple(r.Context(), partnerID, currentUserID)
	if err != nil {
		http.Error(w, "Failed to create couple", http.StatusInternalServerError)
		return
	}

	// Delete code from Redis
	h.DB.GetRedis().Del(r.Context(), key)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(couple)
}
