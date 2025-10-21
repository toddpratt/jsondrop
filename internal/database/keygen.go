package database

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const (
	databaseIDLength = 16
	writeKeyLength   = 32
	readKeyLength    = 32
)

// GenerateDatabaseID generates a unique database ID with "db_" prefix
func GenerateDatabaseID() (string, error) {
	id, err := generateRandomString(databaseIDLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate database ID: %w", err)
	}
	return "db_" + id, nil
}

// GenerateWriteKey generates a write key with "wk_" prefix
func GenerateWriteKey() (string, error) {
	key, err := generateRandomString(writeKeyLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate write key: %w", err)
	}
	return "wk_" + key, nil
}

// GenerateReadKey generates a read key with "rk_" prefix
func GenerateReadKey() (string, error) {
	key, err := generateRandomString(readKeyLength)
	if err != nil {
		return "", fmt.Errorf("failed to generate read key: %w", err)
	}
	return "rk_" + key, nil
}

// generateRandomString creates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	// Calculate bytes needed for base64 encoding
	// We need more bytes than the final string length due to base64 encoding
	byteLength := (length * 3) / 4
	if byteLength < length {
		byteLength = length
	}

	bytes := make([]byte, byteLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	// Use URL-safe base64 encoding without padding
	encoded := base64.RawURLEncoding.EncodeToString(bytes)

	// Trim to exact length needed
	if len(encoded) > length {
		encoded = encoded[:length]
	}

	return encoded, nil
}
