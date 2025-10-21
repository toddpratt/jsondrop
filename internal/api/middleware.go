package api

import (
	"context"
	"net/http"
	"strings"

	"jsondrop/internal/database"
	"jsondrop/internal/models"

	"github.com/go-chi/chi/v5"
)

// contextKey is a type for context keys
type contextKey string

const (
	contextKeyDatabase contextKey = "database"
	contextKeyIsWrite  contextKey = "is_write"
)

// authMiddleware validates the API key and loads the database
func authMiddleware(catalog *database.CatalogDB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract API key from Authorization header or query parameter
			apiKey := r.Header.Get("Authorization")
			if apiKey != "" {
				// Remove "Bearer " prefix if present
				apiKey = strings.TrimPrefix(apiKey, "Bearer ")
			} else {
				// Fallback to query parameter
				apiKey = r.URL.Query().Get("key")
			}

			if apiKey == "" {
				respondError(w, http.StatusUnauthorized, "Unauthorized", "Missing API key")
				return
			}

			// Try to authenticate with write key first
			var db *models.Database
			var isWrite bool
			var err error

			if strings.HasPrefix(apiKey, "wk_") {
				db, err = catalog.GetDatabaseByWriteKey(apiKey)
				isWrite = true
			} else if strings.HasPrefix(apiKey, "rk_") {
				db, err = catalog.GetDatabaseByReadKey(apiKey)
				isWrite = false
			} else {
				respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid API key format")
				return
			}

			if err != nil {
				respondError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to authenticate")
				return
			}

			if db == nil {
				respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid API key")
				return
			}

			// Verify the database ID in the URL matches the authenticated database
			dbIDFromURL := chi.URLParam(r, "id")
			if dbIDFromURL != "" && dbIDFromURL != db.ID {
				respondError(w, http.StatusForbidden, "Forbidden", "Database ID mismatch")
				return
			}

			// Update last accessed timestamp
			if err := catalog.UpdateLastAccessed(db.ID); err != nil {
				// Log error but don't fail the request
				// TODO: Add proper logging
			}

			// Store database and write permission in context
			ctx := context.WithValue(r.Context(), contextKeyDatabase, db)
			ctx = context.WithValue(ctx, contextKeyIsWrite, isWrite)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// requireWriteKey middleware ensures the request uses a write key
func requireWriteKey(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isWrite, ok := r.Context().Value(contextKeyIsWrite).(bool)
		if !ok || !isWrite {
			respondError(w, http.StatusForbidden, "Forbidden", "Write key required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// getDatabaseFromContext retrieves the database from request context
func getDatabaseFromContext(r *http.Request) *models.Database {
	db, _ := r.Context().Value(contextKeyDatabase).(*models.Database)
	return db
}

// isWriteKeyFromContext checks if the request is using a write key
func isWriteKeyFromContext(r *http.Request) bool {
	isWrite, _ := r.Context().Value(contextKeyIsWrite).(bool)
	return isWrite
}
