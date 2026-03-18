package token

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("expired token")
)

type Claims struct {
	SubnetPrefix string
	ExpiresAt    time.Time
}

type Manager struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

type payload struct {
	SubnetPrefix string `json:"subnet_prefix"`
	ExpiresAt    int64  `json:"exp"`
}

func NewManager(secret []byte) *Manager {
	secretCopy := append([]byte(nil), secret...)
	return &Manager{
		secret: secretCopy,
		ttl:    24 * time.Hour,
		now:    time.Now,
	}
}

func (m *Manager) Generate(subnetPrefix string) (string, error) {
	return m.generateForTime(subnetPrefix, m.now().Add(m.ttl))
}

func (m *Manager) generateForTime(subnetPrefix string, expiresAt time.Time) (string, error) {
	if subnetPrefix == "" {
		return "", fmt.Errorf("subnet prefix is required")
	}

	rawPayload, err := json.Marshal(payload{
		SubnetPrefix: subnetPrefix,
		ExpiresAt:    expiresAt.Unix(),
	})
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(rawPayload)
	signature := m.sign(encodedPayload)
	encodedSignature := base64.RawURLEncoding.EncodeToString(signature)
	return encodedPayload + "." + encodedSignature, nil
}

func (m *Manager) Validate(rawToken string) (Claims, error) {
	parts := strings.Split(rawToken, ".")
	if len(parts) != 2 {
		return Claims{}, ErrInvalidToken
	}

	expectedSignature := m.sign(parts[0])
	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}
	if !hmac.Equal(expectedSignature, providedSignature) {
		return Claims{}, ErrInvalidToken
	}

	decodedPayload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims payload
	if err := json.Unmarshal(decodedPayload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}

	expiresAt := time.Unix(claims.ExpiresAt, 0)
	if m.now().After(expiresAt) {
		return Claims{}, ErrExpiredToken
	}

	if claims.SubnetPrefix == "" {
		return Claims{}, ErrInvalidToken
	}

	return Claims{
		SubnetPrefix: claims.SubnetPrefix,
		ExpiresAt:    expiresAt,
	}, nil
}

func (m *Manager) sign(message string) []byte {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(message))
	return mac.Sum(nil)
}
