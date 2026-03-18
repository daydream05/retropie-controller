package server

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"retropie-controller/internal/gamepad"
	"retropie-controller/internal/token"
)

func TestHandlerServesJoinPageForValidToken(t *testing.T) {
	t.Parallel()

	tokenManager := token.NewManager([]byte("0123456789abcdef"))
	joinToken, err := tokenManager.Generate("192.168")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	srv, err := New(Config{
		TokenManager:   tokenManager,
		Gamepads:       &fakeGamepads{count: 8},
		ControllerHTML: "<!doctype html><title>controller</title>",
		ServerIP:       net.ParseIP("192.168.1.50"),
		MaxPlayers:     8,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/join/"+joinToken, nil)
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if !strings.Contains(rec.Body.String(), "<title>controller</title>") {
		t.Fatalf("body = %q, want embedded controller HTML", rec.Body.String())
	}
}

func TestWebSocketJoinAssignsPlayer(t *testing.T) {
	t.Parallel()

	tokenManager := token.NewManager([]byte("0123456789abcdef"))
	joinToken, err := tokenManager.Generate("192.168")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	gamepads := &fakeGamepads{count: 8}
	srv, err := New(Config{
		TokenManager:     tokenManager,
		Gamepads:         gamepads,
		ControllerHTML:   "<!doctype html>",
		ServerIP:         net.ParseIP("192.168.1.50"),
		MaxPlayers:       8,
		HeartbeatTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	httpServer := httptest.NewServer(srv.Handler())
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	dialer := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("X-Forwarded-For", "192.168.1.90")

	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type":  "join",
		"token": joinToken,
	}); err != nil {
		t.Fatalf("WriteJSON(join) error = %v", err)
	}

	var response map[string]any
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}

	if response["type"] != "joined" {
		raw, _ := json.Marshal(response)
		t.Fatalf("response = %s, want joined", raw)
	}
	if response["player"] != float64(1) {
		t.Fatalf("player = %v, want %v", response["player"], 1)
	}
	if gamepads.releaseCalls != 0 {
		t.Fatalf("releaseCalls = %d, want 0 before disconnect", gamepads.releaseCalls)
	}
}

func TestWebSocketJoinAcceptsIPv6ClientWhenTokenIsValid(t *testing.T) {
	t.Parallel()

	tokenManager := token.NewManager([]byte("0123456789abcdef"))
	joinToken, err := tokenManager.Generate("192.168")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	gamepads := &fakeGamepads{count: 8}
	srv, err := New(Config{
		TokenManager:     tokenManager,
		Gamepads:         gamepads,
		ControllerHTML:   "<!doctype html>",
		ServerIP:         net.ParseIP("192.168.1.50"),
		MaxPlayers:       8,
		HeartbeatTimeout: 500 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	httpServer := httptest.NewServer(srv.Handler())
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http") + "/ws"
	dialer := websocket.Dialer{}
	headers := http.Header{}
	headers.Set("X-Forwarded-For", "2600:4041:5b6a:8500:1102:88f3:55be:ff9a")

	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		t.Fatalf("Dial() error = %v", err)
	}
	defer conn.Close()

	if err := conn.WriteJSON(map[string]any{
		"type":  "join",
		"token": joinToken,
	}); err != nil {
		t.Fatalf("WriteJSON(join) error = %v", err)
	}

	var response map[string]any
	if err := conn.ReadJSON(&response); err != nil {
		t.Fatalf("ReadJSON() error = %v", err)
	}

	if response["type"] != "joined" {
		raw, _ := json.Marshal(response)
		t.Fatalf("response = %s, want joined", raw)
	}
}

type fakeGamepads struct {
	count        int
	releaseCalls int
}

func (f *fakeGamepads) Count() int {
	return f.count
}

func (f *fakeGamepads) SetButtons(_ int, _ gamepad.Buttons) error {
	return nil
}

func (f *fakeGamepads) ReleaseAll(_ int) error {
	f.releaseCalls++
	return nil
}

func (f *fakeGamepads) ReleaseAllPlayers() error {
	return nil
}

func (f *fakeGamepads) Close() error {
	return nil
}
