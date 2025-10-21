package api

import (
	"encoding/json"
	"net/http"

	"jsondrop/internal/database"
	"jsondrop/internal/models"

	"github.com/go-chi/chi/v5"
)

// Handler holds dependencies for API handlers
type Handler struct {
	catalog *database.CatalogDB
}

// NewHandler creates a new API handler
func NewHandler(catalog *database.CatalogDB) *Handler {
	return &Handler{
		catalog: catalog,
	}
}

// CreateDatabase handles POST /api/databases
func (h *Handler) CreateDatabase(w http.ResponseWriter, r *http.Request) {
	resp, err := h.catalog.CreateDatabase()
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Failed to create database", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, resp)
}

// CreateSchema handles POST /api/databases/:id/schemas/:name
func (h *Handler) CreateSchema(w http.ResponseWriter, r *http.Request) {
	db := getDatabaseFromContext(r)
	if db == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid authentication")
		return
	}

	schemaName := chi.URLParam(r, "name")
	if schemaName == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Schema name is required")
		return
	}

	// Parse request body
	var req models.CreateSchemaRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid JSON body")
		return
	}

	if len(req.Fields) == 0 {
		respondError(w, http.StatusBadRequest, "Bad Request", "Schema must have at least one field")
		return
	}

	// Validate field types
	for fieldName, fieldType := range req.Fields {
		if !fieldType.IsValid() {
			respondError(w, http.StatusBadRequest, "Bad Request", "Invalid field type: "+string(fieldType))
			return
		}
		if fieldName == "" {
			respondError(w, http.StatusBadRequest, "Bad Request", "Field name cannot be empty")
			return
		}
	}

	// Check if schema already exists
	existingSchema, err := h.catalog.GetSchema(db.ID, schemaName)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to check existing schema")
		return
	}
	if existingSchema != nil {
		respondError(w, http.StatusConflict, "Conflict", "Schema already exists")
		return
	}

	// Create schema
	schema, err := h.catalog.CreateSchema(db.ID, schemaName, req.Fields)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, schema)
}

// respondJSON writes a JSON response
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// respondError writes an error response
func respondError(w http.ResponseWriter, status int, error string, message string) {
	resp := models.ErrorResponse{
		Error:   error,
		Message: message,
	}
	respondJSON(w, status, resp)
}
