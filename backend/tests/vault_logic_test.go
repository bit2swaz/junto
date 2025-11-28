package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bit2swaz/junto/internal/database"
	"github.com/bit2swaz/junto/internal/handlers"
	"github.com/bit2swaz/junto/internal/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVaultLogic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Setup router
	vaultHandler := &handlers.VaultHandler{DB: db}
	authHandler := &handlers.AuthHandler{DB: db}
	coupleHandler := &handlers.CoupleHandler{DB: db}

	r := chi.NewRouter()
	r.Post("/login", authHandler.Login)
	r.Post("/register", authHandler.Register)

	r.Group(func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)
		r.Post("/couples/code", coupleHandler.GeneratePairingCode)
		r.Post("/couples/link", coupleHandler.LinkPartner)
		r.Post("/vault", vaultHandler.AddToVault)
		r.Get("/vault", vaultHandler.GetVaultItems)
	})

	ts := httptest.NewServer(r)
	defer ts.Close()
	client := ts.Client()

	// 1. Setup Couple (User A and User B)
	emailA := "alice@example.com"
	passA := "password"
	registerUser(t, client, ts.URL, emailA, passA)
	tokenA := loginUser(t, client, ts.URL, emailA, passA)

	emailB := "bob@example.com"
	passB := "password"
	registerUser(t, client, ts.URL, emailB, passB)
	tokenB := loginUser(t, client, ts.URL, emailB, passB)

	code := generatePairingCode(t, client, ts.URL, tokenA)
	linkPartner(t, client, ts.URL, tokenB, code)

	// 2. Test Case 1 (Future): Insert item with unlock_at tomorrow (User A creates)
	futureContent := "Future Secret"
	futureUnlock := time.Now().Add(24 * time.Hour)
	createVaultItem(t, client, ts.URL, tokenA, futureContent, futureUnlock)

	// 3. Test Case 2 (Past): Insert item with unlock_at yesterday (User A creates)
	pastContent := "Past Secret"
	pastUnlock := time.Now().Add(-24 * time.Hour)
	createVaultItem(t, client, ts.URL, tokenA, pastContent, pastUnlock)

	// 4. Verify visibility for Partner (User B)
	// User B should see:
	// - Future item: Locked=true, Content=""
	// - Past item: Locked=false, Content="Past Secret"
	itemsB := getVaultItems(t, client, ts.URL, tokenB)

	// Find items
	var futureItemB, pastItemB database.VaultItem
	for _, item := range itemsB {
		if item.UnlockAt.After(time.Now()) {
			futureItemB = item
		} else {
			pastItemB = item
		}
	}

	// Assert Future Item for Partner
	assert.True(t, futureItemB.Locked, "Future item should be locked for partner")
	assert.Empty(t, futureItemB.ContentText, "Future item content should be masked for partner")

	// Assert Past Item for Partner
	assert.False(t, pastItemB.Locked, "Past item should be unlocked for partner")
	assert.Equal(t, pastContent, pastItemB.ContentText, "Past item content should be visible for partner")

	// 5. Test Case 3 (Owner): Ensure User A can see their own future note
	itemsA := getVaultItems(t, client, ts.URL, tokenA)

	var futureItemA database.VaultItem
	for _, item := range itemsA {
		if item.UnlockAt.After(time.Now()) {
			futureItemA = item
		}
	}

	// Assert Future Item for Owner
	// Note: Depending on implementation, Locked might be true (because it is time-locked),
	// but ContentText MUST be visible.
	assert.Equal(t, futureContent, futureItemA.ContentText, "Owner should see content of future item")
}

func createVaultItem(t *testing.T, client *http.Client, baseURL, token, content string, unlockAt time.Time) {
	reqBody, _ := json.Marshal(map[string]interface{}{
		"content":   content,
		"unlock_at": unlockAt,
	})
	req, err := http.NewRequest("POST", baseURL+"/vault", bytes.NewBuffer(reqBody))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)
}

func getVaultItems(t *testing.T, client *http.Client, baseURL, token string) []database.VaultItem {
	req, err := http.NewRequest("GET", baseURL+"/vault", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	var items []database.VaultItem
	err = json.NewDecoder(resp.Body).Decode(&items)
	require.NoError(t, err)
	return items
}
