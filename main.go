package main

import (
	"context"
	"crypto/rand"
	"embed"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	qrcode "github.com/skip2/go-qrcode"

	"retropie-controller/internal/gamepad"
	"retropie-controller/internal/server"
	"retropie-controller/internal/token"
)

const (
	defaultPort  = 80
	fallbackPort = 3000
	maxPlayers   = 8
)

//go:embed public/controller.html
var embeddedFiles embed.FS

func main() {
	logger := log.New(os.Stdout, "[retropie-controller] ", log.LstdFlags)

	serverIP, err := detectLANIP()
	if err != nil {
		logger.Fatalf("detect LAN IP: %v", err)
	}

	secret := make([]byte, 16)
	if _, err := rand.Read(secret); err != nil {
		logger.Fatalf("generate secret: %v", err)
	}

	tokenManager := token.NewManager(secret)
	sessionToken, err := tokenManager.Generate(subnetPrefix(serverIP))
	if err != nil {
		logger.Fatalf("generate session token: %v", err)
	}

	controllerHTML, err := embeddedFiles.ReadFile("public/controller.html")
	if err != nil {
		logger.Fatalf("read embedded controller: %v", err)
	}

	gamepads, err := gamepad.NewManager(maxPlayers)
	if err != nil {
		logger.Fatalf("create gamepads: %v", err)
	}

	srv, err := server.New(server.Config{
		TokenManager:   tokenManager,
		Gamepads:       gamepads,
		ControllerHTML: string(controllerHTML),
		ServerIP:       serverIP,
		MaxPlayers:     maxPlayers,
		Logger:         logger,
	})
	if err != nil {
		logger.Fatalf("create server: %v", err)
	}

	httpServer, port, err := listenAndServe(logger, srv.Handler())
	if err != nil {
		logger.Fatalf("start HTTP server: %v", err)
	}

	joinURL := fmt.Sprintf("http://%s/join/%s", serverIP.String(), sessionToken)
	if port != defaultPort {
		joinURL = fmt.Sprintf("http://%s:%d/join/%s", serverIP.String(), port, sessionToken)
	}

	if err := qrcode.WriteFile(joinURL, qrcode.Medium, 256, "/tmp/controller-qr.png"); err != nil {
		logger.Printf("write QR PNG: %v", err)
	}

	qr, err := qrcode.New(joinURL, qrcode.Medium)
	if err == nil {
		logger.Printf("Scan to join:\n%s", qr.ToSmallString(false))
	} else {
		logger.Printf("create ASCII QR: %v", err)
	}
	logger.Printf("Controller URL: %s", joinURL)
	logger.Printf("QR PNG written to /tmp/controller-qr.png")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Printf("http shutdown: %v", err)
	}
	if err := srv.Shutdown(ctx); err != nil {
		logger.Printf("server shutdown: %v", err)
	}
}

func listenAndServe(logger *log.Logger, handler http.Handler) (*http.Server, int, error) {
	ports := []int{fallbackPort}
	if os.Geteuid() == 0 {
		ports = []int{defaultPort, fallbackPort}
	}

	var lastErr error
	for _, port := range ports {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			lastErr = err
			continue
		}

		httpServer := &http.Server{
			Addr:              addr,
			Handler:           handler,
			ReadHeaderTimeout: 5 * time.Second,
		}

		go func() {
			if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
				logger.Printf("http serve error: %v", err)
			}
		}()

		return httpServer, port, nil
	}

	return nil, 0, lastErr
}

func detectLANIP() (net.IP, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var fallback net.IP
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch value := addr.(type) {
			case *net.IPNet:
				ip = value.IP
			case *net.IPAddr:
				ip = value.IP
			default:
				continue
			}

			ip = ip.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			if isPreferredPrivateIP(ip) {
				return ip, nil
			}
			if isPrivateIP(ip) && fallback == nil {
				fallback = ip
			}
		}
	}

	if fallback != nil {
		return fallback, nil
	}
	return nil, fmt.Errorf("no LAN IPv4 address found")
}

func isPreferredPrivateIP(ip net.IP) bool {
	return strings.HasPrefix(ip.String(), "192.168.") || strings.HasPrefix(ip.String(), "10.")
}

func isPrivateIP(ip net.IP) bool {
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}
	if ip4[0] == 10 {
		return true
	}
	if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
		return true
	}
	return ip4[0] == 192 && ip4[1] == 168
}

func subnetPrefix(ip net.IP) string {
	ip4 := ip.To4()
	if ip4 == nil {
		return ""
	}
	return fmt.Sprintf("%d.%d", ip4[0], ip4[1])
}
