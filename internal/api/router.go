package api

import (
	"net/http"

	"jsondrop/internal/database"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates and configures the HTTP router
func NewRouter(handler *Handler, catalog *database.CatalogDB, corsOrigins []string) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(corsOrigins))

	// Routes
	r.Route("/api", func(r chi.Router) {
		// Database creation (no auth required)
		r.Post("/databases", handler.CreateDatabase)

		// Authenticated routes
		r.Route("/databases/{id}", func(r chi.Router) {
			r.Use(authMiddleware(catalog))

			// SSE endpoint for database events (read or write key)
			r.Get("/events", handler.StreamDatabaseEvents)

			// Schema creation (write key required)
			r.With(requireWriteKey).Post("/schemas/{name}", handler.CreateSchema)

			// Collection-specific routes
			r.Route("/{collection}", func(r chi.Router) {
				// SSE endpoint for collection-specific events (read or write key)
				r.Get("/events", handler.StreamCollectionEvents)

				// Document operations (write key required)
				r.With(requireWriteKey).Post("/", handler.InsertDocument)

				// TODO: Add GET, PUT, DELETE for documents
			})
		})
	})

	return r
}

// corsMiddleware adds CORS headers to responses
func corsMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			if len(allowedOrigins) == 1 && allowedOrigins[0] == "*" {
				allowed = true
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				for _, allowedOrigin := range allowedOrigins {
					if origin == allowedOrigin {
						allowed = true
						w.Header().Set("Access-Control-Allow-Origin", origin)
						break
					}
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
				w.Header().Set("Access-Control-Max-Age", "3600")
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
