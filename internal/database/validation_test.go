package database

import (
	"strings"
	"testing"
)

func TestValidateIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		identifier  string
		shouldError bool
		errorMsg    string
	}{
		// Valid identifiers
		{
			name:        "simple lowercase",
			identifier:  "users",
			shouldError: false,
		},
		{
			name:        "simple uppercase",
			identifier:  "USERS",
			shouldError: false,
		},
		{
			name:        "with underscore",
			identifier:  "user_profiles",
			shouldError: false,
		},
		{
			name:        "starting with underscore",
			identifier:  "_internal",
			shouldError: false,
		},
		{
			name:        "with numbers",
			identifier:  "table_123",
			shouldError: false,
		},
		{
			name:        "mixed case",
			identifier:  "UserProfiles",
			shouldError: false,
		},

		// Invalid identifiers - empty/too long
		{
			name:        "empty string",
			identifier:  "",
			shouldError: true,
			errorMsg:    "cannot be empty",
		},
		{
			name:        "too long",
			identifier:  strings.Repeat("a", 65),
			shouldError: true,
			errorMsg:    "too long",
		},

		// Invalid identifiers - bad characters
		{
			name:        "starting with number",
			identifier:  "123table",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "with space",
			identifier:  "user table",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "with dash",
			identifier:  "user-table",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "with period",
			identifier:  "user.table",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "with semicolon",
			identifier:  "users;",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "with backtick",
			identifier:  "users`",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "with quote",
			identifier:  "users'",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "with double quote",
			identifier:  "users\"",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},

		// SQL injection attempts
		{
			name:        "sql injection with drop",
			identifier:  "users; DROP TABLE _collections--",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "sql injection with union",
			identifier:  "users' UNION SELECT * FROM databases--",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "sql injection with comment",
			identifier:  "users/**/OR/**/1=1",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},
		{
			name:        "backtick escape attempt",
			identifier:  "users`; DROP TABLE test; --",
			shouldError: true,
			errorMsg:    "must start with letter or underscore",
		},

		// SQL reserved keywords
		{
			name:        "reserved keyword SELECT",
			identifier:  "SELECT",
			shouldError: true,
			errorMsg:    "SQL reserved keyword",
		},
		{
			name:        "reserved keyword INSERT",
			identifier:  "INSERT",
			shouldError: true,
			errorMsg:    "SQL reserved keyword",
		},
		{
			name:        "reserved keyword UPDATE",
			identifier:  "UPDATE",
			shouldError: true,
			errorMsg:    "SQL reserved keyword",
		},
		{
			name:        "reserved keyword DELETE",
			identifier:  "DELETE",
			shouldError: true,
			errorMsg:    "SQL reserved keyword",
		},
		{
			name:        "reserved keyword DROP",
			identifier:  "DROP",
			shouldError: true,
			errorMsg:    "SQL reserved keyword",
		},
		{
			name:        "reserved keyword TABLE",
			identifier:  "TABLE",
			shouldError: true,
			errorMsg:    "SQL reserved keyword",
		},
		{
			name:        "reserved keyword lowercase",
			identifier:  "select",
			shouldError: true,
			errorMsg:    "SQL reserved keyword",
		},
		{
			name:        "reserved keyword mixed case",
			identifier:  "Select",
			shouldError: true,
			errorMsg:    "SQL reserved keyword",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateIdentifier(tt.identifier)
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for identifier %q, got nil", tt.identifier)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for identifier %q, got %v", tt.identifier, err)
				}
			}
		})
	}
}

func TestQuoteIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		identifier string
		expected   string
	}{
		{
			name:       "simple identifier",
			identifier: "users",
			expected:   "`users`",
		},
		{
			name:       "with underscore",
			identifier: "user_profiles",
			expected:   "`user_profiles`",
		},
		{
			name:       "with numbers",
			identifier: "table123",
			expected:   "`table123`",
		},
		{
			name:       "single backtick",
			identifier: "user`table",
			expected:   "`user``table`",
		},
		{
			name:       "multiple backticks",
			identifier: "user`table`name",
			expected:   "`user``table``name`",
		},
		{
			name:       "backtick at start",
			identifier: "`users",
			expected:   "```users`",
		},
		{
			name:       "backtick at end",
			identifier: "users`",
			expected:   "`users```",
		},
		{
			name:       "only backticks",
			identifier: "```",
			expected:   "```````````",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := QuoteIdentifier(tt.identifier)
			if result != tt.expected {
				t.Errorf("QuoteIdentifier(%q) = %q, want %q", tt.identifier, result, tt.expected)
			}
		})
	}
}

func TestSafeIdentifier(t *testing.T) {
	tests := []struct {
		name        string
		identifier  string
		expected    string
		shouldError bool
	}{
		{
			name:        "valid identifier",
			identifier:  "users",
			expected:    "`users`",
			shouldError: false,
		},
		{
			name:        "valid with underscore",
			identifier:  "user_profiles",
			expected:    "`user_profiles`",
			shouldError: false,
		},
		{
			name:        "invalid - sql injection",
			identifier:  "users; DROP TABLE test--",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "invalid - reserved keyword",
			identifier:  "SELECT",
			expected:    "",
			shouldError: true,
		},
		{
			name:        "invalid - special characters",
			identifier:  "user-table",
			expected:    "",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := SafeIdentifier(tt.identifier)
			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for identifier %q, got nil", tt.identifier)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for identifier %q, got %v", tt.identifier, err)
				}
				if result != tt.expected {
					t.Errorf("SafeIdentifier(%q) = %q, want %q", tt.identifier, result, tt.expected)
				}
			}
		})
	}
}

// TestSQLInjectionPrevention tests that malicious identifiers are rejected
func TestSQLInjectionPrevention(t *testing.T) {
	maliciousIdentifiers := []string{
		// Drop table attempts
		"users; DROP TABLE _collections--",
		"users'; DROP TABLE _collections--",
		"users`; DROP TABLE _collections--",

		// Union select attempts
		"users' UNION SELECT * FROM databases--",
		"users UNION SELECT write_key FROM databases--",

		// Comment injection
		"users/**/OR/**/1=1",
		"users--",
		"users#",

		// Batch execution
		"users; DELETE FROM databases; --",

		// Backtick escape attempts
		"users` WHERE 1=1; --",
		"`users",

		// Quote escape attempts
		"users'",
		"users\"",

		// Path traversal in table names
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32",

		// Special SQL characters
		"users;",
		"users/*",
		"users*/",
		"users--",
		"users#",
		"users@",
		"users$",
		"users%",

		// Newlines and control characters
		"users\n",
		"users\r",
		"users\t",
		"users\x00",
	}

	for _, identifier := range maliciousIdentifiers {
		t.Run("reject_"+identifier, func(t *testing.T) {
			err := ValidateIdentifier(identifier)
			if err == nil {
				t.Errorf("Expected malicious identifier %q to be rejected, but it was accepted", identifier)
			}
		})
	}
}

// TestValidIdentifiersAccepted ensures legitimate use cases work
func TestValidIdentifiersAccepted(t *testing.T) {
	validIdentifiers := []string{
		"users",
		"user_profiles",
		"UserProfiles",
		"_internal",
		"table123",
		"TABLE_123",
		"a",
		"A",
		"_",
		"a1b2c3",
		"CamelCaseTable",
		"snake_case_table",
		"SCREAMING_SNAKE_CASE",
		"users2",
		"v1_migration",
		"temp_2024_10_21",
	}

	for _, identifier := range validIdentifiers {
		t.Run("accept_"+identifier, func(t *testing.T) {
			err := ValidateIdentifier(identifier)
			if err != nil {
				t.Errorf("Expected valid identifier %q to be accepted, but got error: %v", identifier, err)
			}

			// Also test that it can be safely quoted
			quoted := QuoteIdentifier(identifier)
			if !strings.HasPrefix(quoted, "`") || !strings.HasSuffix(quoted, "`") {
				t.Errorf("QuoteIdentifier(%q) = %q, expected backtick-quoted result", identifier, quoted)
			}
		})
	}
}
