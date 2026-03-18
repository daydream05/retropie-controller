package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"retropie-controller/internal/gamepad"
	"retropie-controller/internal/token"
)

const (
	defaultMaxPlayers       = 8
	defaultHeartbeatTimeout = 500 * time.Millisecond
	defaultJoinWindow       = time.Minute
	defaultJoinLimit        = 10
)

type GamepadManager interface {
	Count() int
	SetButtons(player int, buttons gamepad.Buttons) error
	ReleaseAll(player int) error
	ReleaseAllPlayers() error
	Close() error
}

type Config struct {
	TokenManager     *token.Manager
	Gamepads         GamepadManager
	ControllerHTML   string
	ServerIP         net.IP
	MaxPlayers       int
	HeartbeatTimeout time.Duration
	Logger           *log.Logger
}

type Server struct {
	tokenManager     *token.Manager
	gamepads         GamepadManager
	controllerHTML   string
	serverIP         net.IP
	maxPlayers       int
	heartbeatTimeout time.Duration
	logger           *log.Logger
	upgrader         websocket.Upgrader
	limiter          *joinLimiter

	mu      sync.Mutex
	players []bool
}

type clientMessage struct {
	Type    string          `json:"type"`
	Token   string          `json:"token,omitempty"`
	Buttons gamepad.Buttons `json:"buttons,omitempty"`
}

type serverMessage struct {
	Type    string `json:"type"`
	Player  int    `json:"player,omitempty"`
	Message string `json:"message,omitempty"`
}

func New(cfg Config) (*Server, error) {
	if cfg.TokenManager == nil {
		return nil, fmt.Errorf("token manager is required")
	}
	if cfg.Gamepads == nil {
		return nil, fmt.Errorf("gamepads manager is required")
	}
	if cfg.ControllerHTML == "" {
		return nil, fmt.Errorf("controller HTML is required")
	}
	if cfg.ServerIP == nil {
		return nil, fmt.Errorf("server IP is required")
	}

	maxPlayers := cfg.MaxPlayers
	if maxPlayers == 0 {
		maxPlayers = defaultMaxPlayers
	}
	if maxPlayers > cfg.Gamepads.Count() {
		return nil, fmt.Errorf("max players %d exceeds available gamepads %d", maxPlayers, cfg.Gamepads.Count())
	}

	timeout := cfg.HeartbeatTimeout
	if timeout == 0 {
		timeout = defaultHeartbeatTimeout
	}

	logger := cfg.Logger
	if logger == nil {
		logger = log.Default()
	}

	srv := &Server{
		tokenManager:     cfg.TokenManager,
		gamepads:         cfg.Gamepads,
		controllerHTML:   cfg.ControllerHTML,
		serverIP:         cfg.ServerIP,
		maxPlayers:       maxPlayers,
		heartbeatTimeout: timeout,
		logger:           logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
		limiter: &joinLimiter{
			limit:   defaultJoinLimit,
			window:  defaultJoinWindow,
			records: make(map[string][]time.Time),
		},
		players: make([]bool, maxPlayers),
	}

	return srv, nil
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/join/", s.handleJoinPage)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	return mux
}

func (s *Server) Shutdown(_ context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for player, taken := range s.players {
		if !taken {
			continue
		}
		if err := s.gamepads.ReleaseAll(player + 1); err != nil {
			s.logger.Printf("release player %d: %v", player+1, err)
		}
		s.players[player] = false
	}

	if err := s.gamepads.ReleaseAllPlayers(); err != nil {
		s.logger.Printf("release all players: %v", err)
	}
	return s.gamepads.Close()
}

func (s *Server) handleJoinPage(w http.ResponseWriter, r *http.Request) {
	tokenValue := strings.TrimPrefix(r.URL.Path, "/join/")
	if tokenValue == "" {
		s.logger.Printf("join page missing token from %s", r.RemoteAddr)
		http.NotFound(w, r)
		return
	}

	if _, err := s.tokenManager.Validate(tokenValue); err != nil {
		s.logger.Printf("join page invalid token from %s: %v", r.RemoteAddr, err)
		http.Error(w, "invalid token", http.StatusForbidden)
		return
	}

	s.logger.Printf("join page served to %s", r.RemoteAddr)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(s.controllerHTML))
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		s.logger.Printf("websocket upgrade failed for %s: %v", r.RemoteAddr, err)
		return
	}
	s.logger.Printf("websocket upgraded for %s", r.RemoteAddr)

	clientIP, err := clientIPFromRequest(r)
	if err != nil {
		s.logger.Printf("websocket client IP parse failed for %s: %v", r.RemoteAddr, err)
		_ = s.writeMessage(conn, serverMessage{Type: "error", Message: "invalid_token"})
		_ = conn.Close()
		return
	}
	s.logger.Printf("websocket client IP %s from remote %s", clientIP.String(), r.RemoteAddr)

	conn.SetReadLimit(2048)
	_ = conn.SetReadDeadline(time.Now().Add(s.heartbeatTimeout))

	var joinRequest clientMessage
	if err := conn.ReadJSON(&joinRequest); err != nil {
		s.logger.Printf("websocket read join failed for %s: %v", clientIP.String(), err)
		_ = conn.Close()
		return
	}

	if joinRequest.Type != "join" {
		s.logger.Printf("websocket invalid first message from %s: %s", clientIP.String(), joinRequest.String())
		_ = s.writeMessage(conn, serverMessage{Type: "error", Message: "invalid_token"})
		_ = conn.Close()
		return
	}

	if !s.limiter.Allow(clientIP.String(), time.Now()) {
		s.logger.Printf("websocket join rate limited for %s", clientIP.String())
		_ = s.writeMessage(conn, serverMessage{Type: "error", Message: "invalid_token"})
		_ = conn.Close()
		return
	}

	claims, err := s.tokenManager.Validate(joinRequest.Token)
	if err != nil || claims.SubnetPrefix != subnetPrefix(s.serverIP) || subnetPrefix(clientIP) != subnetPrefix(s.serverIP) {
		s.logger.Printf("websocket join rejected for %s: validate_err=%v token_subnet=%q server_subnet=%q client_subnet=%q", clientIP.String(), err, claims.SubnetPrefix, subnetPrefix(s.serverIP), subnetPrefix(clientIP))
		_ = s.writeMessage(conn, serverMessage{Type: "error", Message: "invalid_token"})
		_ = conn.Close()
		return
	}

	player, ok := s.assignPlayer()
	if !ok {
		s.logger.Printf("websocket join rejected for %s: server full", clientIP.String())
		_ = s.writeMessage(conn, serverMessage{Type: "error", Message: "server_full"})
		_ = conn.Close()
		return
	}
	s.logger.Printf("websocket join accepted for %s as player %d", clientIP.String(), player)

	defer func() {
		if err := s.gamepads.ReleaseAll(player); err != nil {
			s.logger.Printf("release player %d: %v", player, err)
		}
		s.freePlayer(player)
		s.logger.Printf("websocket disconnected for %s player %d", clientIP.String(), player)
		_ = conn.Close()
	}()

	if err := s.writeMessage(conn, serverMessage{Type: "joined", Player: player}); err != nil {
		return
	}

	for {
		_ = conn.SetReadDeadline(time.Now().Add(s.heartbeatTimeout))

		var message clientMessage
		if err := conn.ReadJSON(&message); err != nil {
			if websocket.IsCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				return
			}
			var netErr net.Error
			if errors.As(err, &netErr) && netErr.Timeout() {
				return
			}
			return
		}

		switch message.Type {
		case "input":
			if err := s.gamepads.SetButtons(player, message.Buttons); err != nil {
				s.logger.Printf("set buttons for player %d: %v", player, err)
				return
			}
		case "heartbeat":
			if err := s.writeMessage(conn, serverMessage{Type: "heartbeat_ack"}); err != nil {
				return
			}
		case "release_all":
			if err := s.gamepads.ReleaseAll(player); err != nil {
				s.logger.Printf("release_all for player %d: %v", player, err)
				return
			}
		default:
			if err := s.writeMessage(conn, serverMessage{Type: "error", Message: "invalid_token"}); err != nil {
				return
			}
		}
	}
}

func (s *Server) writeMessage(conn *websocket.Conn, message serverMessage) error {
	conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	return conn.WriteJSON(message)
}

func (s *Server) assignPlayer() (int, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, taken := range s.players {
		if taken {
			continue
		}
		s.players[i] = true
		return i + 1, true
	}

	return 0, false
}

func (s *Server) freePlayer(player int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if player >= 1 && player <= len(s.players) {
		s.players[player-1] = false
	}
}

func clientIPFromRequest(r *http.Request) (net.IP, error) {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		ip := net.ParseIP(first)
		if ip != nil {
			return ip, nil
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip := net.ParseIP(r.RemoteAddr)
		if ip == nil {
			return nil, err
		}
		return ip, nil
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, fmt.Errorf("invalid remote addr %q", r.RemoteAddr)
	}
	return ip, nil
}

func subnetPrefix(ip net.IP) string {
	ip = ip.To4()
	if ip == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d", ip[0], ip[1])
}

type joinLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	records map[string][]time.Time
}

func (l *joinLimiter) Allow(ip string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	recent := l.records[ip][:0]
	for _, ts := range l.records[ip] {
		if ts.After(cutoff) {
			recent = append(recent, ts)
		}
	}
	if len(recent) >= l.limit {
		l.records[ip] = recent
		return false
	}

	l.records[ip] = append(recent, now)
	return true
}

func (m clientMessage) String() string {
	raw, _ := json.Marshal(m)
	return string(raw)
}
