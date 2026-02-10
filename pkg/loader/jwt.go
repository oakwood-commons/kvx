package loader

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// IsJWT detects if input looks like a JWT token.
// A valid JWT has exactly 3 dot-separated parts where the first two
// are valid base64url-encoded JSON objects.
func IsJWT(input string) bool {
	// Strip common prefixes and whitespace
	input = strings.TrimPrefix(input, "Bearer ")
	input = strings.TrimSpace(input)

	// Must have exactly 3 parts
	parts := strings.Split(input, ".")
	if len(parts) != 3 {
		return false
	}

	// Each part must be non-empty
	for _, part := range parts {
		if len(part) == 0 {
			return false
		}
	}

	// First two parts must be valid base64url AND valid JSON objects
	for i := 0; i < 2; i++ {
		decoded, err := base64.RawURLEncoding.DecodeString(parts[i])
		if err != nil {
			return false
		}
		var obj map[string]any
		if err := json.Unmarshal(decoded, &obj); err != nil {
			return false
		}
	}

	// Signature just needs to be valid base64url (can contain any bytes)
	_, err := base64.RawURLEncoding.DecodeString(parts[2])
	return err == nil
}

// DecodeJWT splits and decodes a JWT token into structured data.
// Returns a flat structure with header, payload, and signature keys.
func DecodeJWT(input string) (map[string]any, error) {
	// Strip common prefixes and whitespace
	input = strings.TrimPrefix(input, "Bearer ")
	input = strings.TrimSpace(input)

	parts := strings.Split(input, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid JWT: expected 3 parts, got %d", len(parts))
	}

	// Decode header
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid JWT header: %w", err)
	}
	var header map[string]any
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return nil, fmt.Errorf("invalid JWT header JSON: %w", err)
	}

	// Decode payload
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid JWT payload: %w", err)
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return nil, fmt.Errorf("invalid JWT payload JSON: %w", err)
	}

	// Signature is kept as the raw base64url string (not decoded to bytes)
	// since it's binary data that can't be represented as JSON
	signature := parts[2]

	return map[string]any{
		"header":    header,
		"payload":   payload,
		"signature": signature,
	}, nil
}

// loadJWT parses a JWT string and returns it wrapped in []interface{}.
func loadJWT(input string) ([]interface{}, error) {
	decoded, err := DecodeJWT(input)
	if err != nil {
		return nil, err
	}
	return []interface{}{decoded}, nil
}
