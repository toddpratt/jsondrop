package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"jsondrop/internal/models"
)

// InsertDocument inserts a new document into a collection
func (c *CatalogDB) InsertDocument(dbID string, collection string, data map[string]interface{}) (*models.Document, error) {
	// Generate document ID
	docID, err := GenerateDocumentID()
	if err != nil {
		return nil, err
	}

	// Marshal data to JSON
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal document data: %w", err)
	}

	now := time.Now().Unix()

	// Open the database file
	dbPath := c.getDatabasePath(dbID)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Insert document
	query := fmt.Sprintf(`
		INSERT INTO %s (id, created_at, updated_at, data)
		VALUES (?, ?, ?, ?)
	`, collection)

	_, err = db.Exec(query, docID, now, now, string(dataJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to insert document: %w", err)
	}

	// Calculate size and update quota
	documentSize := int64(len(dataJSON))
	if err := c.updateQuotaAfterInsert(dbID, documentSize); err != nil {
		// Try to rollback the insert
		db.Exec(fmt.Sprintf("DELETE FROM %s WHERE id = ?", collection), docID)
		return nil, err
	}

	doc := &models.Document{
		ID:         docID,
		Collection: collection,
		Data:       data,
		CreatedAt:  time.Unix(now, 0),
		UpdatedAt:  time.Unix(now, 0),
	}

	// Broadcast insert event
	if c.broadcaster != nil {
		event := models.ChangeEvent{
			EventType:  "insert",
			DatabaseID: dbID,
			Collection: collection,
			DocumentID: docID,
			Data:       data,
			Timestamp:  time.Unix(now, 0),
		}
		c.broadcaster.Broadcast(dbID, event)
	}

	return doc, nil
}

// updateQuotaAfterInsert updates quota and checks if limit is exceeded
func (c *CatalogDB) updateQuotaAfterInsert(dbID string, additionalSize int64) error {
	// Get current quota usage
	var quotaUsed, quotaLimit int64
	query := `SELECT quota_used, quota_limit FROM databases WHERE id = ?`
	err := c.db.QueryRow(query, dbID).Scan(&quotaUsed, &quotaLimit)
	if err != nil {
		return fmt.Errorf("failed to get quota: %w", err)
	}

	newQuotaUsed := quotaUsed + additionalSize

	// Check if quota would be exceeded
	if newQuotaUsed > quotaLimit {
		return fmt.Errorf("quota exceeded: current %d bytes, limit %d bytes, attempted to add %d bytes",
			quotaUsed, quotaLimit, additionalSize)
	}

	// Update quota
	return c.UpdateQuotaUsed(dbID, newQuotaUsed)
}

// GenerateDocumentID generates a unique document ID
func GenerateDocumentID() (string, error) {
	id, err := generateRandomString(16)
	if err != nil {
		return "", fmt.Errorf("failed to generate document ID: %w", err)
	}
	return "doc_" + id, nil
}

// GetDocument retrieves a single document by ID
func (c *CatalogDB) GetDocument(dbID string, collection string, docID string) (*models.Document, error) {
	dbPath := c.getDatabasePath(dbID)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	query := fmt.Sprintf(`
		SELECT id, created_at, updated_at, data
		FROM %s
		WHERE id = ?
	`, collection)

	var doc models.Document
	var createdAt, updatedAt int64
	var dataJSON string

	err = db.QueryRow(query, docID).Scan(
		&doc.ID,
		&createdAt,
		&updatedAt,
		&dataJSON,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get document: %w", err)
	}

	// Unmarshal data
	if err := json.Unmarshal([]byte(dataJSON), &doc.Data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal document data: %w", err)
	}

	doc.Collection = collection
	doc.CreatedAt = time.Unix(createdAt, 0)
	doc.UpdatedAt = time.Unix(updatedAt, 0)

	return &doc, nil
}

// QueryDocuments retrieves documents from a collection with pagination and filtering
func (c *CatalogDB) QueryDocuments(dbID string, collection string, limit int, offset int, filters map[string][]string) ([]*models.Document, error) {
	dbPath := c.getDatabasePath(dbID)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Build query
	query := fmt.Sprintf(`
		SELECT id, created_at, updated_at, data
		FROM %s
		ORDER BY created_at DESC
	`, collection)

	// Add limit and offset
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query documents: %w", err)
	}
	defer rows.Close()

	var documents []*models.Document
	for rows.Next() {
		var doc models.Document
		var createdAt, updatedAt int64
		var dataJSON string

		err := rows.Scan(
			&doc.ID,
			&createdAt,
			&updatedAt,
			&dataJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan document: %w", err)
		}

		// Unmarshal data
		if err := json.Unmarshal([]byte(dataJSON), &doc.Data); err != nil {
			return nil, fmt.Errorf("failed to unmarshal document data: %w", err)
		}

		doc.Collection = collection
		doc.CreatedAt = time.Unix(createdAt, 0)
		doc.UpdatedAt = time.Unix(updatedAt, 0)

		// Apply in-memory filtering
		if matchesFilters(&doc, filters) {
			documents = append(documents, &doc)
		}
	}

	return documents, rows.Err()
}

// matchesFilters checks if a document matches the provided filters
// Multiple values for the same field are treated as OR (IN list)
func matchesFilters(doc *models.Document, filters map[string][]string) bool {
	if len(filters) == 0 {
		return true
	}

	for fieldName, filterValues := range filters {
		if len(filterValues) == 0 {
			continue
		}

		// Get the field value from the document
		fieldValue, exists := doc.Data[fieldName]
		if !exists {
			return false // Field doesn't exist in document
		}

		// Check if field value matches any of the filter values (OR logic)
		matched := false
		for _, filterValue := range filterValues {
			if matchesValue(fieldValue, filterValue) {
				matched = true
				break
			}
		}

		if !matched {
			return false // AND logic between different fields
		}
	}

	return true
}

// matchesValue checks if a field value matches a filter value
func matchesValue(fieldValue interface{}, filterValue string) bool {
	switch v := fieldValue.(type) {
	case string:
		return v == filterValue
	case float64:
		// Try to parse filter as number
		if filterNum, err := strconv.ParseFloat(filterValue, 64); err == nil {
			return v == filterNum
		}
		return false
	case bool:
		// Try to parse filter as boolean
		if filterBool, err := strconv.ParseBool(filterValue); err == nil {
			return v == filterBool
		}
		return false
	default:
		// Convert to string and compare
		return fmt.Sprintf("%v", fieldValue) == filterValue
	}
}

// DeleteDocument deletes a single document by ID
func (c *CatalogDB) DeleteDocument(dbID string, collection string, docID string) error {
	dbPath := c.getDatabasePath(dbID)
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	// Get document size before deletion for quota update
	var dataJSON string
	query := fmt.Sprintf(`SELECT data FROM %s WHERE id = ?`, collection)
	err = db.QueryRow(query, docID).Scan(&dataJSON)
	if err == sql.ErrNoRows {
		return fmt.Errorf("document not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get document: %w", err)
	}

	documentSize := int64(len(dataJSON))

	// Delete the document
	deleteQuery := fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, collection)
	result, err := db.Exec(deleteQuery, docID)
	if err != nil {
		return fmt.Errorf("failed to delete document: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("document not found")
	}

	// Update quota
	var quotaUsed int64
	quotaQuery := `SELECT quota_used FROM databases WHERE id = ?`
	err = c.db.QueryRow(quotaQuery, dbID).Scan(&quotaUsed)
	if err != nil {
		// Log error but don't fail the delete
		return nil
	}

	newQuotaUsed := quotaUsed - documentSize
	if newQuotaUsed < 0 {
		newQuotaUsed = 0
	}
	c.UpdateQuotaUsed(dbID, newQuotaUsed)

	// Broadcast delete event
	if c.broadcaster != nil {
		event := models.ChangeEvent{
			EventType:  "delete",
			DatabaseID: dbID,
			Collection: collection,
			DocumentID: docID,
			Data:       nil, // No data for delete events
			Timestamp:  time.Now(),
		}
		c.broadcaster.Broadcast(dbID, event)
	}

	return nil
}
