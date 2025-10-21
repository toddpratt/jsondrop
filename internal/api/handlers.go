package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"jsondrop/internal/database"
	"jsondrop/internal/events"
	"jsondrop/internal/models"

	"github.com/go-chi/chi/v5"
)

// Handler holds dependencies for API handlers
type Handler struct {
	catalog     *database.CatalogDB
	broadcaster *events.Broadcaster
}

// NewHandler creates a new API handler
func NewHandler(catalog *database.CatalogDB, broadcaster *events.Broadcaster) *Handler {
	return &Handler{
		catalog:     catalog,
		broadcaster: broadcaster,
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

// InsertDocument handles POST /api/databases/:id/:collection
func (h *Handler) InsertDocument(w http.ResponseWriter, r *http.Request) {
	db := getDatabaseFromContext(r)
	if db == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid authentication")
		return
	}

	collection := chi.URLParam(r, "collection")
	if collection == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Collection name is required")
		return
	}

	// Parse request body
	var req models.InsertDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid JSON body")
		return
	}

	if len(req.Data) == 0 {
		respondError(w, http.StatusBadRequest, "Bad Request", "Document data cannot be empty")
		return
	}

	// Get schema for validation
	schema, err := h.catalog.GetSchema(db.ID, collection)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to get schema")
		return
	}
	if schema == nil {
		respondError(w, http.StatusNotFound, "Not Found", "Schema does not exist for collection: "+collection)
		return
	}

	// Validate document against schema
	if err := models.ValidateDocument(req.Data, schema); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Validation failed: "+err.Error())
		return
	}

	// Insert document
	doc, err := h.catalog.InsertDocument(db.ID, collection, req.Data)
	if err != nil {
		// Check if it's a quota error
		if strings.Contains(err.Error(), "quota exceeded") {
			respondError(w, http.StatusPaymentRequired, "Quota Exceeded", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, doc)
}

// StreamDatabaseEvents handles GET /api/databases/:id/events (SSE)
func (h *Handler) StreamDatabaseEvents(w http.ResponseWriter, r *http.Request) {
	db := getDatabaseFromContext(r)
	if db == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid authentication")
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable buffering in nginx

	// Subscribe to events
	listener := h.broadcaster.Subscribe(db.ID)
	defer h.broadcaster.Unsubscribe(db.ID, listener)

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"database_id\":\"%s\",\"timestamp\":\"%s\"}\n\n",
		db.ID, time.Now().Format(time.RFC3339))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Heartbeat ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Stream events
	for {
		select {
		case event := <-listener.Events:
			// Send event to client
			fmt.Fprint(w, events.FormatSSE(event))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case <-ticker.C:
			// Send heartbeat/ping
			fmt.Fprint(w, events.FormatPing())
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			h.broadcaster.UpdatePing(listener)

		case <-listener.Done:
			// Listener was closed by broadcaster
			return

		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

// StreamCollectionEvents handles GET /api/databases/:id/:collection/events (SSE)
func (h *Handler) StreamCollectionEvents(w http.ResponseWriter, r *http.Request) {
	db := getDatabaseFromContext(r)
	if db == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid authentication")
		return
	}

	collection := chi.URLParam(r, "collection")
	if collection == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Collection name is required")
		return
	}

	// Verify schema exists for this collection
	schema, err := h.catalog.GetSchema(db.ID, collection)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to verify collection")
		return
	}
	if schema == nil {
		respondError(w, http.StatusNotFound, "Not Found", "Collection does not exist: "+collection)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // Disable buffering in nginx

	// Subscribe to collection-specific events
	listener := h.broadcaster.SubscribeCollection(db.ID, collection)
	defer h.broadcaster.UnsubscribeCollection(db.ID, collection, listener)

	// Send initial connection message
	fmt.Fprintf(w, "event: connected\ndata: {\"database_id\":\"%s\",\"collection\":\"%s\",\"timestamp\":\"%s\"}\n\n",
		db.ID, collection, time.Now().Format(time.RFC3339))
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// Heartbeat ticker
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Stream events
	for {
		select {
		case event := <-listener.Events:
			// Send event to client
			fmt.Fprint(w, events.FormatSSE(event))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

		case <-ticker.C:
			// Send heartbeat/ping
			fmt.Fprint(w, events.FormatPing())
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			h.broadcaster.UpdatePing(listener)

		case <-listener.Done:
			// Listener was closed by broadcaster
			return

		case <-r.Context().Done():
			// Client disconnected
			return
		}
	}
}

// QueryDocuments handles GET /api/databases/:id/:collection
func (h *Handler) QueryDocuments(w http.ResponseWriter, r *http.Request) {
	db := getDatabaseFromContext(r)
	if db == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid authentication")
		return
	}

	collection := chi.URLParam(r, "collection")
	if collection == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Collection name is required")
		return
	}

	// Verify schema exists for this collection
	schema, err := h.catalog.GetSchema(db.ID, collection)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to verify collection")
		return
	}
	if schema == nil {
		respondError(w, http.StatusNotFound, "Not Found", "Collection does not exist: "+collection)
		return
	}

	// Parse pagination parameters
	limit := 100 // Default limit
	offset := 0

	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
			if limit > 1000 {
				limit = 1000 // Max limit
			}
		}
	}

	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}

	// Parse filters from query parameters
	// Multiple values for same parameter are treated as OR (IN list)
	filters := make(map[string][]string)
	for key, values := range r.URL.Query() {
		// Skip pagination parameters
		if key == "limit" || key == "offset" {
			continue
		}
		// Only include fields that exist in the schema
		if _, exists := schema.Fields[key]; exists {
			filters[key] = values
		}
	}

	// Query documents
	documents, err := h.catalog.QueryDocuments(db.ID, collection, limit, offset, filters)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	// Return empty array if no documents found
	if documents == nil {
		documents = []*models.Document{}
	}

	respondJSON(w, http.StatusOK, documents)
}

// DeleteDocument handles DELETE /api/databases/:id/:collection/:docId
func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	db := getDatabaseFromContext(r)
	if db == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid authentication")
		return
	}

	collection := chi.URLParam(r, "collection")
	if collection == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Collection name is required")
		return
	}

	docID := chi.URLParam(r, "docId")
	if docID == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Document ID is required")
		return
	}

	// Delete document
	err := h.catalog.DeleteDocument(db.ID, collection, docID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "Not Found", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// UpdateDocument handles PUT /api/databases/:id/:collection/:docId
func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	db := getDatabaseFromContext(r)
	if db == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid authentication")
		return
	}

	collection := chi.URLParam(r, "collection")
	if collection == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Collection name is required")
		return
	}

	docID := chi.URLParam(r, "docId")
	if docID == "" {
		respondError(w, http.StatusBadRequest, "Bad Request", "Document ID is required")
		return
	}

	// Parse request body
	var req models.UpdateDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Invalid JSON body")
		return
	}

	if len(req.Data) == 0 {
		respondError(w, http.StatusBadRequest, "Bad Request", "Document data cannot be empty")
		return
	}

	// Get schema for validation
	schema, err := h.catalog.GetSchema(db.ID, collection)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal Server Error", "Failed to get schema")
		return
	}
	if schema == nil {
		respondError(w, http.StatusNotFound, "Not Found", "Schema does not exist for collection: "+collection)
		return
	}

	// Validate document against schema
	if err := models.ValidateDocument(req.Data, schema); err != nil {
		respondError(w, http.StatusBadRequest, "Bad Request", "Validation failed: "+err.Error())
		return
	}

	// Update document
	doc, err := h.catalog.UpdateDocument(db.ID, collection, docID, req.Data)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "Not Found", err.Error())
			return
		}
		if strings.Contains(err.Error(), "quota exceeded") {
			respondError(w, http.StatusPaymentRequired, "Quota Exceeded", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	respondJSON(w, http.StatusOK, doc)
}

// DeleteSchema handles DELETE /api/databases/:id/schemas/:name
func (h *Handler) DeleteSchema(w http.ResponseWriter, r *http.Request) {
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

	// Delete schema
	err := h.catalog.DeleteSchema(db.ID, schemaName)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, "Not Found", err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteDatabase handles DELETE /api/databases/:id
func (h *Handler) DeleteDatabase(w http.ResponseWriter, r *http.Request) {
	db := getDatabaseFromContext(r)
	if db == nil {
		respondError(w, http.StatusUnauthorized, "Unauthorized", "Invalid authentication")
		return
	}

	// Delete database
	err := h.catalog.DeleteDatabase(db.ID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "Internal Server Error", err.Error())
		return
	}

	w.WriteHeader(http.StatusNoContent)
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
