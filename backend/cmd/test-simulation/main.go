package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

const baseURL = "http://localhost:8080"
const wsURL = "ws://localhost:8080/ws"

func main() {
	// 1. Setup Users and Link them
	fmt.Println("Setting up users...")
	tokenA := setupUserAndGetToken("sim_user_a@junto.app", "password")
	tokenB := setupUserAndGetToken("sim_user_b@junto.app", "password")

	code := getPairingCode(tokenA)
	linkPartner(tokenB, code)
	fmt.Println("Users linked.")

	// 2. Connect Client A
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	connA, _, err := websocket.Dial(ctx, fmt.Sprintf("%s?token=%s", wsURL, tokenA), nil)
	if err != nil {
		log.Fatalf("Client A failed to connect: %v", err)
	}
	defer connA.Close(websocket.StatusNormalClosure, "")

	// 3. Connect Client B
	connB, _, err := websocket.Dial(ctx, fmt.Sprintf("%s?token=%s", wsURL, tokenB), nil)
	if err != nil {
		log.Fatalf("Client B failed to connect: %v", err)
	}
	defer connB.Close(websocket.StatusNormalClosure, "")

	fmt.Println("WebSockets connected.")

	// 4. Client A sends move
	moveMsg := map[string]interface{}{
		"type": "move",
		"x":    50,
		"y":    50,
	}
	fmt.Printf("Client A sending: %v\n", moveMsg)
	if err := wsjson.Write(ctx, connA, moveMsg); err != nil {
		log.Fatalf("Client A failed to send: %v", err)
	}

	// 5. Client B listens
	done := make(chan struct{})
	go func() {
		var msg map[string]interface{}
		// Read with a short timeout for the assertion
		readCtx, readCancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer readCancel()

		err := wsjson.Read(readCtx, connB, &msg)
		if err != nil {
			fail("Client B failed to receive message or timed out: " + err.Error())
			return
		}

		// Assert content
		if msg["type"] != "move" || msg["x"].(float64) != 50 || msg["y"].(float64) != 50 {
			fail(fmt.Sprintf("Client B received unexpected message: %v", msg))
			return
		}
		close(done)
	}()

	select {
	case <-done:
		pass("PASS: Client B received the move message correctly.")
	case <-time.After(3 * time.Second):
		fail("FAIL: Test timed out waiting for Client B.")
	}
}

func pass(msg string) {
	// Green
	fmt.Printf("\033[32m%s\033[0m\n", msg)
}

func fail(msg string) {
	// Red
	fmt.Printf("\033[31m%s\033[0m\n", msg)
	// Don't exit 1 to allow cleanup if needed, but for this script it's fine
}

// Helpers
func setupUserAndGetToken(email, password string) string {
	// Register (ignore error if exists)
	postJSON("/register", map[string]string{"email": email, "password": password})
	// Login
	resp := postJSON("/login", map[string]string{"email": email, "password": password})
	return resp["token"].(string)
}

func getPairingCode(token string) string {
	resp := postJSONWithAuth("/couples/code", nil, token)
	return resp["code"].(string)
}

func linkPartner(token, code string) {
	postJSONWithAuth("/couples/link", map[string]string{"code": code}, token)
}

func postJSON(path string, body interface{}) map[string]interface{} {
	return doRequest("POST", path, body, "")
}

func postJSONWithAuth(path string, body interface{}, token string) map[string]interface{} {
	return doRequest("POST", path, body, token)
}

func doRequest(method, path string, body interface{}, token string) map[string]interface{} {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(b)
	}

	req, _ := http.NewRequest(method, baseURL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Request failed %s: %v", path, err)
	}
	defer resp.Body.Close()

	// We expect 200 or 201. If 400+, log error body
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		// Exception: Register might fail if user exists, we handle that in setupUserAndGetToken by ignoring?
		// But for simplicity, we just log.
		// If user already exists (500 or 400 depending on impl), we might want to proceed to login
		if path == "/register" {
			return nil
		}
		log.Fatalf("Request failed %s status %d: %s", path, resp.StatusCode, string(b))
	}

	var res map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&res)
	return res
}
