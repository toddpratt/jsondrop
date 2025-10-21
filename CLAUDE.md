# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

JSONDrop is an anonymous JSON storage API service that allows users to create ephemeral databases with read/write key-based authentication. Each database has schema validation, quota enforcement, and auto-expires after 30 days of inactivity.

## Architecture

The project follows a clean Go architecture pattern:

- `cmd/server/` - Entry point with HTTP server initialization
- `internal/config/` - Configuration management (environment variables, defaults)
- `internal/api/` - HTTP handlers and routing logic
- `internal/database/` - SQLite operations for both metadata catalog and per-database storage
- `internal/models/` - Data structures and types
- `internal/auth/` - Key validation middleware for read_key and write_key
- `internal/quota/` - Storage quota tracking and enforcement
- `internal/events/` - Server-Sent Events (SSE) system for real-time change notifications

### Key Design Decisions

**Two-tier authentication**: Each database has two keys:
- `write_key` (wk_ prefix, 32 random chars) - Full CRUD access
- `read_key` (rk_ prefix, 32 random chars) - Read-only access

**Database isolation**: Each database gets its own SQLite file for document storage, with a central catalog tracking metadata, quotas, and expiry.

**Storage model**: SQLite for both catalog metadata and per-database document storage. No external database dependencies.

**Schema validation**: Schemas must be explicitly defined before inserting documents. Supported types: string, number, bool.

**Quota enforcement**: 100MB default per database. Writes are rejected when quota is exceeded. Track total storage size on each write operation.

**Auto-expiry**: Background job deletes databases with `last_accessed` timestamp older than 30 days.

**Real-time events**: Server-Sent Events (SSE) endpoints allow clients to listen for changes at database-level or collection-level granularity. Events are broadcast on INSERT, UPDATE, and DELETE operations.

**Configuration management**: Server configuration is loaded from environment variables with sensible defaults. All configuration is centralized in the `internal/config` package.

## Key Generation Format

- Database ID: `db_` + 16 random alphanumeric characters
- Write key: `wk_` + 32 random alphanumeric characters
- Read key: `rk_` + 32 random alphanumeric characters

## API Endpoints

```
POST   /api/databases                              Create database, returns ID and keys
POST   /api/databases/:id/schemas/:name            Define schema for collection
POST   /api/databases/:id/:collection              Insert document (requires write_key)
GET    /api/databases/:id/:collection              Query documents (requires read_key or write_key)
GET    /api/databases/:id/:collection/:docId       Get single document (requires read_key or write_key)
PUT    /api/databases/:id/:collection/:docId       Update document (requires write_key)
DELETE /api/databases/:id/:collection/:docId       Delete document (requires write_key)
GET    /api/databases/:id/info                     Get quota usage info (requires read_key or write_key)
GET    /api/databases/:id/events                   SSE stream for all database changes (requires read_key or write_key)
GET    /api/databases/:id/:collection/events       SSE stream for collection-specific changes (requires read_key or write_key)
```

## Configuration

Configuration is managed through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `PORT` | HTTP server port | `8080` |
| `DB_BASE_DIR` | Base directory for SQLite database files | `./data` |
| `CATALOG_DB_PATH` | Path to catalog database file | `./data/catalog.db` |
| `CORS_ORIGINS` | Comma-separated list of allowed CORS origins | `*` |
| `DEFAULT_QUOTA_MB` | Default quota per database in MB | `100` |
| `EXPIRY_DAYS` | Days of inactivity before database expiry | `30` |
| `EXPIRY_CHECK_INTERVAL` | How often to run expiry cleanup (e.g., "24h") | `24h` |

## Development Commands

**Build the server:**
```bash
go build -o bin/jsondrop cmd/server/main.go
```

**Run the server:**
```bash
go run cmd/server/main.go
```

**Run with custom configuration:**
```bash
PORT=3000 DB_BASE_DIR=/var/lib/jsondrop CORS_ORIGINS="https://example.com,https://app.example.com" go run cmd/server/main.go
```

**Run tests:**
```bash
go test ./...
```

**Run tests for a specific package:**
```bash
go test ./internal/api
```

**Install dependencies:**
```bash
go mod tidy
```

## HTTP Router

Use `chi` router for HTTP routing (as specified in PROJECT.md). Install with:
```bash
go get github.com/go-chi/chi/v5
```

## Implementation Notes

- Rate limiting is handled externally by Traefik, not in the application
- Each HTTP request to a database should update the `last_accessed` timestamp
- Schema validation must occur before document insertion
- Background expiry job should run periodically based on `EXPIRY_CHECK_INTERVAL` configuration
- Storage quota checks must happen before accepting write operations
- Configuration is loaded once at startup from environment variables
- Database files are stored in `DB_BASE_DIR` with naming pattern: `{database_id}.db`
- CORS origins should be validated against the configured allowlist; `*` allows all origins

### Server-Sent Events (SSE) Implementation

The SSE system broadcasts real-time change notifications to connected clients:

- **Event types**: `insert`, `update`, `delete` - corresponding to CRUD operations
- **Event payload**: JSON containing operation type, collection name, document ID, and timestamp
- **Connection management**: Track active SSE connections per database and per collection
- **Broadcasting**: When a write operation occurs, broadcast events to:
  - All database-level listeners (`/api/databases/:id/events`)
  - Collection-level listeners for the affected collection (`/api/databases/:id/:collection/events`)
- **Authentication**: SSE endpoints require either read_key or write_key in query parameters or Authorization header
- **Keep-alive**: Send periodic heartbeat comments to prevent connection timeouts
- **Cleanup**: Remove disconnected clients from listener pools to prevent memory leaks
