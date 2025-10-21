# Anonymous JSON Storage API

## Core Features
1. Anonymous database creation - returns write_key and read_key
2. Schema definition with validation (string, number, bool)
3. CRUD operations on JSON documents
4. Two-tier auth: read-only and read-write keys
5. Per-database storage quotas (enforced)
6. Auto-expire inactive databases after 30 days

## Tech Stack
- Go 1.21+
- HTTP router: chi or gin
- Database: SQLite (one file per database)
- Storage: filesystem or single SQLite catalog
- No rate limiting (handled by Traefik)

## API Endpoints
POST   /api/databases          -> Create database, returns keys
POST   /api/databases/:id/schemas/:name  -> Define schema
POST   /api/databases/:id/:collection    -> Insert document
GET    /api/databases/:id/:collection    -> Query documents
GET    /api/databases/:id/:collection/:docId -> Get single document
PUT    /api/databases/:id/:collection/:docId -> Update document
DELETE /api/databases/:id/:collection/:docId -> Delete document
GET    /api/databases/:id/info -> Get quota usage

## Key Generation
- database_id: db_ + random 16 chars
- write_key: wk_ + random 32 chars
- read_key: rk_ + random 32 chars

## Quotas
- Default: 100MB per database
- Track: total storage size
- Reject writes when quota exceeded

## Database Expiry
- Mark last_accessed timestamp on each request
- Background job to delete databases inactive for 30+ days
```

## Tips for Claude Code:

When you start Claude Code, give it this prompt:
```
I want to build an anonymous JSON storage API in Go based on PROJECT.md.

Start by:
1. Setting up the project structure with proper Go packages
2. Creating the main.go entry point with a basic HTTP server
3. Implementing the database creation endpoint that generates a unique ID and two API keys
4. Use chi router for HTTP routing
5. Use SQLite for metadata storage (tracking databases, schemas, quotas)

Focus on clean architecture with these packages:
- cmd/server: main.go
- internal/api: HTTP handlers
- internal/database: SQLite operations
- internal/models: Data structures
- internal/auth: Key validation middleware
- internal/quota: Storage tracking

Let's start with the foundation and database creation endpoint first.