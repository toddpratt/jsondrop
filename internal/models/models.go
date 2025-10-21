package models

import "time"

// Database represents a user-created database in the catalog
type Database struct {
	ID           string    `json:"id"`
	WriteKey     string    `json:"-"` // Never expose in JSON responses
	ReadKey      string    `json:"-"` // Never expose in JSON responses
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
	QuotaUsed    int64     `json:"quota_used"`    // bytes
	QuotaLimit   int64     `json:"quota_limit"`   // bytes
}

// Schema represents a collection schema definition
type Schema struct {
	DatabaseID string                 `json:"database_id"`
	Name       string                 `json:"name"`
	Fields     map[string]FieldType   `json:"fields"`
	CreatedAt  time.Time              `json:"created_at"`
}

// FieldType represents the type of a field in a schema
type FieldType string

const (
	FieldTypeString FieldType = "string"
	FieldTypeNumber FieldType = "number"
	FieldTypeBool   FieldType = "bool"
)

// IsValid checks if a field type is valid
func (ft FieldType) IsValid() bool {
	switch ft {
	case FieldTypeString, FieldTypeNumber, FieldTypeBool:
		return true
	default:
		return false
	}
}

// Document represents a JSON document in a collection
type Document struct {
	ID         string                 `json:"id"`
	Collection string                 `json:"collection"`
	Data       map[string]interface{} `json:"data"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

// CreateDatabaseResponse is the response when creating a new database
type CreateDatabaseResponse struct {
	DatabaseID string `json:"database_id"`
	WriteKey   string `json:"write_key"`
	ReadKey    string `json:"read_key"`
}

// CreateSchemaRequest is the request to define a schema
type CreateSchemaRequest struct {
	Fields map[string]FieldType `json:"fields"`
}

// InsertDocumentRequest is the request to insert a document
type InsertDocumentRequest struct {
	Data map[string]interface{} `json:"data"`
}

// UpdateDocumentRequest is the request to update a document
type UpdateDocumentRequest struct {
	Data map[string]interface{} `json:"data"`
}

// DatabaseInfoResponse returns quota and usage information
type DatabaseInfoResponse struct {
	DatabaseID   string    `json:"database_id"`
	QuotaUsed    int64     `json:"quota_used"`
	QuotaLimit   int64     `json:"quota_limit"`
	QuotaPercent float64   `json:"quota_percent"`
	CreatedAt    time.Time `json:"created_at"`
	LastAccessed time.Time `json:"last_accessed"`
}

// ErrorResponse represents an API error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// ChangeEvent represents a change notification for SSE
type ChangeEvent struct {
	EventType  string                 `json:"event_type"` // "insert", "update", "delete"
	DatabaseID string                 `json:"database_id"`
	Collection string                 `json:"collection"`
	DocumentID string                 `json:"document_id"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}
