package database

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	// identifierPattern matches valid SQL identifiers: alphanumeric and underscore, starting with letter or underscore
	identifierPattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
)

// ValidateIdentifier checks if a name is a valid SQL identifier
// Returns error if invalid to prevent SQL injection
func ValidateIdentifier(name string) error {
	if name == "" {
		return fmt.Errorf("identifier cannot be empty")
	}

	if len(name) > 64 {
		return fmt.Errorf("identifier too long (max 64 characters)")
	}

	// Check for valid characters
	if !identifierPattern.MatchString(name) {
		return fmt.Errorf("identifier must start with letter or underscore and contain only alphanumeric characters and underscores")
	}

	// Reject SQL reserved keywords that could be dangerous
	upperName := strings.ToUpper(name)
	reservedKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE", "ALTER",
		"TABLE", "INDEX", "VIEW", "DATABASE", "SCHEMA", "WHERE", "FROM",
		"JOIN", "UNION", "ORDER", "GROUP", "HAVING", "LIMIT", "OFFSET",
	}

	for _, keyword := range reservedKeywords {
		if upperName == keyword {
			return fmt.Errorf("identifier cannot be a SQL reserved keyword: %s", name)
		}
	}

	return nil
}

// QuoteIdentifier safely quotes an identifier for use in SQL queries
// Even though we validate identifiers, this provides defense in depth
func QuoteIdentifier(name string) string {
	// Double any existing backticks to escape them
	escaped := strings.ReplaceAll(name, "`", "``")
	return "`" + escaped + "`"
}

// SafeIdentifier validates and quotes an identifier
// This is the primary function to use for all user-provided identifiers
func SafeIdentifier(name string) (string, error) {
	if err := ValidateIdentifier(name); err != nil {
		return "", err
	}
	return QuoteIdentifier(name), nil
}
