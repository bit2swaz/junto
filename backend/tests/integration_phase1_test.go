package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bit2swaz/junto/internal/database"
	"github.com/bit2swaz/junto/internal/handlers"
	"github.com/bit2swaz/junto/internal/middleware"
	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestEnv() {
	// Load .env from parent directory if not already loaded
	if os.Getenv("DATABASE_URL") == "" {
		_ = godotenv.Load("../.env")
	}
}

func setupTestDB(t *testing.T) database.Service {
	setupTestEnv()

	// Override DB name for testing if needed, or just use the one in .env but clean it
	// For this example, we assume the configured DB is safe to use/clean for tests

	db, err := database.NewService()
	require.NoError(t, err)

	// Clean database
	pool := db.GetPool()
	_, err = pool.Exec(context.Background(), "TRUNCATE TABLE users, couples RESTART IDENTITY CASCADE")
	require.NoError(t, err)

	// Clean Redis
	rdb := db.GetRedis()
	err = rdb.FlushAll(context.Background()).Err()
	require.NoError(t, err)

	return db
}

func setupRouter(db database.Service) *chi.Mux {
	authHandler := &handlers.AuthHandler{DB: db}
	coupleHandler := &handlers.CoupleHandler{DB: db}

	r := chi.NewRouter()
	r.Use(chiMiddleware.Logger)

	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/couples/code", coupleHandler.GeneratePairingCode)
		r.Post("/couples/link", coupleHandler.LinkPartner)
	})

	return r
}

func TestIntegrationPhase1(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	r := setupRouter(db)
	ts := httptest.NewServer(r)
	defer ts.Close()

	client := ts.Client()

	// 1. Register User A
	userAEmail := "usera@example.com"
	userAPass := "password123"
	registerUser(t, client, ts.URL, userAEmail, userAPass)

	// 2. Register User B
	userBEmail := "userb@example.com"
	userBPass := "password123"
	registerUser(t, client, ts.URL, userBEmail, userBPass)

	// 3. Login User A
	tokenA := loginUser(t, client, ts.URL, userAEmail, userAPass)

	// 4. Login User B
	tokenB := loginUser(t, client, ts.URL, userBEmail, userBPass)

	// 5. User A generates pairing code
	code := generatePairingCode(t, client, ts.URL, tokenA)
	assert.Len(t, code, 6)

	// 6. User B links with code
	linkPartner(t, client, ts.URL, tokenB, code)

	// 7. Verify DB state
	verifyCouples(t, db, userAEmail, userBEmail)
}

func registerUser(t *testing.T, client *http.Client, baseURL, email, password string) {
	reqBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	resp, err := client.Post(baseURL+"/register", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func loginUser(t *testing.T, client *http.Client, baseURL, email, password string) string {
	reqBody, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": password,
	})
	resp, err := client.Post(baseURL+"/login", "application/json", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var res map[string]string
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)
	return res["token"]
}

func generatePairingCode(t *testing.T, client *http.Client, baseURL, token string) string {
	req, err := http.NewRequest("POST", baseURL+"/couples/code", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var res map[string]string
	err = json.NewDecoder(resp.Body).Decode(&res)
	require.NoError(t, err)
	return res["code"]
}

func linkPartner(t *testing.T, client *http.Client, baseURL, token, code string) {
	reqBody, _ := json.Marshal(map[string]string{
		"code": code,
	})
	req, err := http.NewRequest("POST", baseURL+"/couples/link", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func verifyCouples(t *testing.T, db database.Service, emailA, emailB string) {
	ctx := context.Background()

	// Get User IDs
	userA, err := db.GetUserByEmail(ctx, emailA)
	require.NoError(t, err)
	userB, err := db.GetUserByEmail(ctx, emailB)
	require.NoError(t, err)

	// Check couples table
	var count int
	err = db.GetPool().QueryRow(ctx, "SELECT COUNT(*) FROM couples WHERE (user1_id = $1 AND user2_id = $2) OR (user1_id = $2 AND user2_id = $1)", userA.ID, userB.ID).Scan(&count)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "Couple row should exist")

	// Check users table couple_id
	// Refresh users
	userA, err = db.GetUserByEmail(ctx, emailA)
	require.NoError(t, err)
	userB, err = db.GetUserByEmail(ctx, emailB)
	require.NoError(t, err)

	assert.NotNil(t, userA.CoupleID)
	assert.NotNil(t, userB.CoupleID)
	assert.Equal(t, *userA.CoupleID, *userB.CoupleID)
}
