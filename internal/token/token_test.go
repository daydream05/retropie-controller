package token

import (
	"testing"
	"time"
)

func TestManagerGenerateAndValidateRoundTrip(t *testing.T) {
	t.Parallel()

	manager := NewManager([]byte("0123456789abcdef"))

	token, err := manager.Generate("192.168")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	claims, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if claims.SubnetPrefix != "192.168" {
		t.Fatalf("SubnetPrefix = %q, want %q", claims.SubnetPrefix, "192.168")
	}

	if claims.ExpiresAt.Before(time.Now()) {
		t.Fatalf("ExpiresAt = %v, want future time", claims.ExpiresAt)
	}
}

func TestManagerValidateRejectsTamperedToken(t *testing.T) {
	t.Parallel()

	manager := NewManager([]byte("0123456789abcdef"))

	token, err := manager.Generate("192.168")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	tampered := token[:len(token)-1] + "x"
	if _, err := manager.Validate(tampered); err == nil {
		t.Fatal("Validate() error = nil, want invalid token error")
	}
}

func TestManagerValidateRejectsExpiredToken(t *testing.T) {
	t.Parallel()

	manager := NewManager([]byte("0123456789abcdef"))

	token, err := manager.generateForTime("192.168", time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("generateForTime() error = %v", err)
	}

	if _, err := manager.Validate(token); err == nil {
		t.Fatal("Validate() error = nil, want expiration error")
	}
}
