package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"jsondrop/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

// EventBroadcaster is an interface for broadcasting events
type EventBroadcaster interface {
	Broadcast(dbID string, event models.ChangeEvent)
}

// CatalogDB manages the catalog database
type CatalogDB struct {
	db           *sql.DB
	dbBaseDir    string
	defaultQuota int64
	broadcaster  EventBroadcaster
}

// NewCatalogDB creates a new catalog database connection
func NewCatalogDB(catalogPath string, dbBaseDir string, defaultQuotaMB int64, broadcaster EventBroadcaster) (*CatalogDB, error) {
	// Ensure the directory exists
	dir := filepath.Dir(catalogPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create catalog directory: %w", err)
	}

	// Ensure base directory for database files exists
	if err := os.MkdirAll(dbBaseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database base directory: %w", err)
	}

	db, err := sql.Open("sqlite3", catalogPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open catalog database: %w", err)
	}

	catalog := &CatalogDB{
		db:           db,
		dbBaseDir:    dbBaseDir,
		defaultQuota: defaultQuotaMB * 1024 * 1024, // Convert MB to bytes
		broadcaster:  broadcaster,
	}

	if err := catalog.initSchema(); err != nil {
		db.Close()
		return nil, err
	}

	return catalog, nil
}

// initSchema creates the catalog tables
func (c *CatalogDB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS databases (
		id TEXT PRIMARY KEY,
		write_key TEXT UNIQUE NOT NULL,
		read_key TEXT UNIQUE NOT NULL,
		created_at INTEGER NOT NULL,
		last_accessed INTEGER NOT NULL,
		quota_used INTEGER NOT NULL DEFAULT 0,
		quota_limit INTEGER NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_write_key ON databases(write_key);
	CREATE INDEX IF NOT EXISTS idx_read_key ON databases(read_key);
	CREATE INDEX IF NOT EXISTS idx_last_accessed ON databases(last_accessed);

	CREATE TABLE IF NOT EXISTS schemas (
		database_id TEXT NOT NULL,
		name TEXT NOT NULL,
		fields TEXT NOT NULL,
		created_at INTEGER NOT NULL,
		PRIMARY KEY (database_id, name),
		FOREIGN KEY (database_id) REFERENCES databases(id) ON DELETE CASCADE
	);
	`

	_, err := c.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize catalog schema: %w", err)
	}

	return nil
}

// CreateDatabase creates a new database entry in the catalog
func (c *CatalogDB) CreateDatabase() (*models.CreateDatabaseResponse, error) {
	// Generate unique identifiers
	dbID, err := GenerateDatabaseID()
	if err != nil {
		return nil, err
	}

	writeKey, err := GenerateWriteKey()
	if err != nil {
		return nil, err
	}

	readKey, err := GenerateReadKey()
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()

	// Insert into catalog
	query := `
		INSERT INTO databases (id, write_key, read_key, created_at, last_accessed, quota_used, quota_limit)
		VALUES (?, ?, ?, ?, ?, 0, ?)
	`

	_, err = c.db.Exec(query, dbID, writeKey, readKey, now, now, c.defaultQuota)
	if err != nil {
		return nil, fmt.Errorf("failed to create database entry: %w", err)
	}

	// Create the SQLite database file
	dbPath := c.getDatabasePath(dbID)
	if err := c.initDatabaseFile(dbPath); err != nil {
		// Rollback: delete from catalog
		c.db.Exec("DELETE FROM databases WHERE id = ?", dbID)
		return nil, fmt.Errorf("failed to create database file: %w", err)
	}

	return &models.CreateDatabaseResponse{
		DatabaseID: dbID,
		WriteKey:   writeKey,
		ReadKey:    readKey,
	}, nil
}

// initDatabaseFile creates a new SQLite database file for a user database
func (c *CatalogDB) initDatabaseFile(dbPath string) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create collections table to track all collections in this database
	schema := `
	CREATE TABLE IF NOT EXISTS _collections (
		name TEXT PRIMARY KEY,
		created_at INTEGER NOT NULL
	);
	`

	_, err = db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize database file schema: %w", err)
	}

	return nil
}

// getDatabasePath returns the file path for a database
func (c *CatalogDB) getDatabasePath(dbID string) string {
	return filepath.Join(c.dbBaseDir, dbID+".db")
}

// GetDatabaseByWriteKey retrieves a database by its write key
func (c *CatalogDB) GetDatabaseByWriteKey(writeKey string) (*models.Database, error) {
	return c.getDatabaseByKey("write_key", writeKey)
}

// GetDatabaseByReadKey retrieves a database by its read key
func (c *CatalogDB) GetDatabaseByReadKey(readKey string) (*models.Database, error) {
	return c.getDatabaseByKey("read_key", readKey)
}

// getDatabaseByKey is a helper to retrieve database by any key field
func (c *CatalogDB) getDatabaseByKey(keyField, keyValue string) (*models.Database, error) {
	query := fmt.Sprintf(`
		SELECT id, write_key, read_key, created_at, last_accessed, quota_used, quota_limit
		FROM databases
		WHERE %s = ?
	`, keyField)

	var db models.Database
	var createdAt, lastAccessed int64

	err := c.db.QueryRow(query, keyValue).Scan(
		&db.ID,
		&db.WriteKey,
		&db.ReadKey,
		&createdAt,
		&lastAccessed,
		&db.QuotaUsed,
		&db.QuotaLimit,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get database: %w", err)
	}

	db.CreatedAt = time.Unix(createdAt, 0)
	db.LastAccessed = time.Unix(lastAccessed, 0)

	return &db, nil
}

// UpdateLastAccessed updates the last_accessed timestamp for a database
func (c *CatalogDB) UpdateLastAccessed(dbID string) error {
	query := `UPDATE databases SET last_accessed = ? WHERE id = ?`
	_, err := c.db.Exec(query, time.Now().Unix(), dbID)
	if err != nil {
		return fmt.Errorf("failed to update last_accessed: %w", err)
	}
	return nil
}

// UpdateQuotaUsed updates the quota_used for a database
func (c *CatalogDB) UpdateQuotaUsed(dbID string, quotaUsed int64) error {
	query := `UPDATE databases SET quota_used = ? WHERE id = ?`
	_, err := c.db.Exec(query, quotaUsed, dbID)
	if err != nil {
		return fmt.Errorf("failed to update quota_used: %w", err)
	}
	return nil
}

// GetExpiredDatabases returns databases that haven't been accessed in the specified number of days
func (c *CatalogDB) GetExpiredDatabases(expiryDays int) ([]string, error) {
	cutoff := time.Now().AddDate(0, 0, -expiryDays).Unix()

	query := `SELECT id FROM databases WHERE last_accessed < ?`
	rows, err := c.db.Query(query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to get expired databases: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, rows.Err()
}

// DeleteDatabase removes a database from the catalog and deletes its file
func (c *CatalogDB) DeleteDatabase(dbID string) error {
	// Delete the database file
	dbPath := c.getDatabasePath(dbID)
	if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete database file: %w", err)
	}

	// Delete from catalog (cascade will delete schemas)
	query := `DELETE FROM databases WHERE id = ?`
	_, err := c.db.Exec(query, dbID)
	if err != nil {
		return fmt.Errorf("failed to delete database from catalog: %w", err)
	}

	return nil
}

// CreateSchema creates a new schema for a collection
func (c *CatalogDB) CreateSchema(dbID string, name string, fields map[string]models.FieldType) (*models.Schema, error) {
	// Validate fields
	for fieldName, fieldType := range fields {
		if fieldName == "" {
			return nil, fmt.Errorf("field name cannot be empty")
		}
		if !fieldType.IsValid() {
			return nil, fmt.Errorf("invalid field type for %s: %s", fieldName, fieldType)
		}
	}

	if len(fields) == 0 {
		return nil, fmt.Errorf("schema must have at least one field")
	}

	// Marshal fields to JSON
	fieldsJSON, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fields: %w", err)
	}

	now := time.Now().Unix()

	// Insert into catalog
	query := `
		INSERT INTO schemas (database_id, name, fields, created_at)
		VALUES (?, ?, ?, ?)
	`

	_, err = c.db.Exec(query, dbID, name, string(fieldsJSON), now)
	if err != nil {
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	// Create the table in the database file
	dbPath := c.getDatabasePath(dbID)
	if err := c.createCollectionTable(dbPath, name, fields); err != nil {
		// Rollback: delete from catalog
		c.db.Exec("DELETE FROM schemas WHERE database_id = ? AND name = ?", dbID, name)
		return nil, fmt.Errorf("failed to create collection table: %w", err)
	}

	schema := &models.Schema{
		DatabaseID: dbID,
		Name:       name,
		Fields:     fields,
		CreatedAt:  time.Unix(now, 0),
	}

	// Broadcast schema creation event
	if c.broadcaster != nil {
		event := models.ChangeEvent{
			EventType:  "schema_created",
			DatabaseID: dbID,
			Collection: name,
			DocumentID: "", // Not applicable for schema events
			Data: map[string]interface{}{
				"schema_name": name,
				"fields":      fields,
			},
			Timestamp: time.Unix(now, 0),
		}
		c.broadcaster.Broadcast(dbID, event)
	}

	return schema, nil
}

// createCollectionTable creates a table in a user's database file
func (c *CatalogDB) createCollectionTable(dbPath string, collectionName string, fields map[string]models.FieldType) error {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Build CREATE TABLE statement
	createSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (", collectionName)
	createSQL += "id TEXT PRIMARY KEY, "
	createSQL += "created_at INTEGER NOT NULL, "
	createSQL += "updated_at INTEGER NOT NULL, "
	createSQL += "data TEXT NOT NULL" // Store entire JSON document
	createSQL += ")"

	if _, err := db.Exec(createSQL); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Register collection
	_, err = db.Exec(
		"INSERT OR IGNORE INTO _collections (name, created_at) VALUES (?, ?)",
		collectionName,
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to register collection: %w", err)
	}

	return nil
}

// GetSchema retrieves a schema by database ID and name
func (c *CatalogDB) GetSchema(dbID string, name string) (*models.Schema, error) {
	query := `
		SELECT database_id, name, fields, created_at
		FROM schemas
		WHERE database_id = ? AND name = ?
	`

	var schema models.Schema
	var fieldsJSON string
	var createdAt int64

	err := c.db.QueryRow(query, dbID, name).Scan(
		&schema.DatabaseID,
		&schema.Name,
		&fieldsJSON,
		&createdAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}

	// Unmarshal fields
	if err := json.Unmarshal([]byte(fieldsJSON), &schema.Fields); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fields: %w", err)
	}

	schema.CreatedAt = time.Unix(createdAt, 0)

	return &schema, nil
}

// Close closes the catalog database connection
func (c *CatalogDB) Close() error {
	return c.db.Close()
}
