package main

import (
	"net"
	"testing"
)

func TestControllerHostPrefersConfiguredHost(t *testing.T) {
	t.Parallel()

	host := controllerHost("controller.example.test", "piclawd", net.ParseIP("192.168.1.162"))

	if host != "controller.example.test" {
		t.Fatalf("controllerHost() = %q, want %q", host, "controller.example.test")
	}
}

func TestControllerHostUsesMDNSHostname(t *testing.T) {
	t.Parallel()

	host := controllerHost("", "piclawd", net.ParseIP("192.168.1.162"))

	if host != "piclawd.local" {
		t.Fatalf("controllerHost() = %q, want %q", host, "piclawd.local")
	}
}

func TestControllerHostFallsBackToIP(t *testing.T) {
	t.Parallel()

	host := controllerHost("", "", net.ParseIP("192.168.1.162"))

	if host != "192.168.1.162" {
		t.Fatalf("controllerHost() = %q, want %q", host, "192.168.1.162")
	}
}
