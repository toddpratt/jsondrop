package database

import (
	"strings"
	"testing"
)

func TestGenerateDatabaseID(t *testing.T) {
	id, err := GenerateDatabaseID()
	if err != nil {
		t.Fatalf("GenerateDatabaseID() error = %v, want nil", err)
	}

	if !strings.HasPrefix(id, "db_") {
		t.Errorf("GenerateDatabaseID() = %s, want prefix 'db_'", id)
	}

	// db_ prefix (3) + 16 characters = 19 total
	if len(id) != 19 {
		t.Errorf("len(GenerateDatabaseID()) = %d, want 19", len(id))
	}
}

func TestGenerateWriteKey(t *testing.T) {
	key, err := GenerateWriteKey()
	if err != nil {
		t.Fatalf("GenerateWriteKey() error = %v, want nil", err)
	}

	if !strings.HasPrefix(key, "wk_") {
		t.Errorf("GenerateWriteKey() = %s, want prefix 'wk_'", key)
	}

	// wk_ prefix (3) + 32 characters = 35 total
	if len(key) != 35 {
		t.Errorf("len(GenerateWriteKey()) = %d, want 35", len(key))
	}
}

func TestGenerateReadKey(t *testing.T) {
	key, err := GenerateReadKey()
	if err != nil {
		t.Fatalf("GenerateReadKey() error = %v, want nil", err)
	}

	if !strings.HasPrefix(key, "rk_") {
		t.Errorf("GenerateReadKey() = %s, want prefix 'rk_'", key)
	}

	// rk_ prefix (3) + 32 characters = 35 total
	if len(key) != 35 {
		t.Errorf("len(GenerateReadKey()) = %d, want 35", len(key))
	}
}

func TestGenerateDatabaseID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		id, err := GenerateDatabaseID()
		if err != nil {
			t.Fatalf("GenerateDatabaseID() error = %v, want nil", err)
		}

		if seen[id] {
			t.Errorf("GenerateDatabaseID() produced duplicate: %s", id)
		}
		seen[id] = true
	}

	if len(seen) != iterations {
		t.Errorf("Generated %d unique IDs out of %d attempts", len(seen), iterations)
	}
}

func TestGenerateKeys_Uniqueness(t *testing.T) {
	writeKeys := make(map[string]bool)
	readKeys := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		wk, err := GenerateWriteKey()
		if err != nil {
			t.Fatalf("GenerateWriteKey() error = %v", err)
		}
		if writeKeys[wk] {
			t.Errorf("GenerateWriteKey() produced duplicate: %s", wk)
		}
		writeKeys[wk] = true

		rk, err := GenerateReadKey()
		if err != nil {
			t.Fatalf("GenerateReadKey() error = %v", err)
		}
		if readKeys[rk] {
			t.Errorf("GenerateReadKey() produced duplicate: %s", rk)
		}
		readKeys[rk] = true
	}
}

func TestGenerateRandomString(t *testing.T) {
	tests := []struct {
		length int
	}{
		{16},
		{32},
		{64},
	}

	for _, tt := range tests {
		t.Run("length_test", func(t *testing.T) {
			str, err := generateRandomString(tt.length)
			if err != nil {
				t.Fatalf("generateRandomString(%d) error = %v", tt.length, err)
			}

			if len(str) != tt.length {
				t.Errorf("len(generateRandomString(%d)) = %d, want %d", tt.length, len(str), tt.length)
			}

			// Check that string contains only valid base64 URL-safe characters
			for _, c := range str {
				if !isBase64URLChar(c) {
					t.Errorf("generateRandomString() contains invalid character: %c", c)
				}
			}
		})
	}
}

func isBase64URLChar(c rune) bool {
	return (c >= 'A' && c <= 'Z') ||
		(c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') ||
		c == '-' || c == '_'
}
