# JSONDrop

A lightweight, anonymous JSON storage API with real-time updates. Create ephemeral databases, define schemas, and store JSON documents with automatic quota management and expiration.

## Features

- **Anonymous Database Creation** - No authentication required to create a database
- **Two-Tier Authentication** - Each database gets separate read and write keys
- **Schema Validation** - Define schemas with string, number, and boolean types
- **CRUD Operations** - Full create, read, update, delete support for documents
- **Real-Time Events** - Server-Sent Events (SSE) for live data updates
- **Quota Management** - Per-database storage limits with automatic tracking
- **Auto-Expiry** - Databases automatically deleted after 30 days of inactivity
- **Filtering & Pagination** - Query documents with filters and limit/offset
- **Zero Configuration** - Works out of the box with sensible defaults

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone <repository-url>
cd jsondrop

# Start the server
docker-compose up -d

# Server is now running on http://localhost:8080
```

### Using Docker

```bash
docker build -t jsondrop .
docker run -d -p 8080:8080 -v jsondrop-data:/app/data jsondrop
```

### From Source

```bash
# Install dependencies
go mod download

# Run the server
go run cmd/server/main.go
```

## API Overview

### Create a Database

```bash
curl -X POST http://localhost:8080/api/databases
```

**Response:**
```json
{
  "database_id": "db_abc123xyz",
  "write_key": "wk_secretwritekey123",
  "read_key": "rk_secretreadkey456"
}
```

**Important:** Save these keys! They cannot be recovered.

### Define a Schema

```bash
curl -X POST http://localhost:8080/api/databases/db_abc123xyz/schemas/users \
  -H "Authorization: Bearer wk_secretwritekey123" \
  -H "Content-Type: application/json" \
  -d '{
    "fields": {
      "name": "string",
      "age": "number",
      "active": "bool"
    }
  }'
```

### Insert a Document

```bash
curl -X POST http://localhost:8080/api/databases/db_abc123xyz/users/ \
  -H "Authorization: Bearer wk_secretwritekey123" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "name": "Alice",
      "age": 25,
      "active": true
    }
  }'
```

**Response:**
```json
{
  "id": "doc_xyz789",
  "collection": "users",
  "data": {
    "name": "Alice",
    "age": 25,
    "active": true
  },
  "created_at": "2025-10-20T20:00:00Z",
  "updated_at": "2025-10-20T20:00:00Z"
}
```

### Query Documents

```bash
# Get all documents
curl -H "Authorization: Bearer rk_secretreadkey456" \
  "http://localhost:8080/api/databases/db_abc123xyz/users/"

# With pagination
curl -H "Authorization: Bearer rk_secretreadkey456" \
  "http://localhost:8080/api/databases/db_abc123xyz/users/?limit=10&offset=0"

# With filters (exact match)
curl -H "Authorization: Bearer rk_secretreadkey456" \
  "http://localhost:8080/api/databases/db_abc123xyz/users/?active=true"

# With IN list (OR logic)
curl -H "Authorization: Bearer rk_secretreadkey456" \
  "http://localhost:8080/api/databases/db_abc123xyz/users/?name=Alice&name=Bob"
```

### Update a Document

```bash
curl -X PUT http://localhost:8080/api/databases/db_abc123xyz/users/doc_xyz789 \
  -H "Authorization: Bearer wk_secretwritekey123" \
  -H "Content-Type: application/json" \
  -d '{
    "data": {
      "name": "Alice Smith",
      "age": 26,
      "active": false
    }
  }'
```

### Delete a Document

```bash
curl -X DELETE http://localhost:8080/api/databases/db_abc123xyz/users/doc_xyz789 \
  -H "Authorization: Bearer wk_secretwritekey123"
```

### Real-Time Events (SSE)

```bash
# Listen to all database events
curl -N -H "Authorization: Bearer rk_secretreadkey456" \
  http://localhost:8080/api/databases/db_abc123xyz/events

# Listen to collection-specific events
curl -N -H "Authorization: Bearer rk_secretreadkey456" \
  http://localhost:8080/api/databases/db_abc123xyz/users/events
```

**Event Types:**
- `schema_created` - New collection created
- `schema_deleted` - Collection deleted
- `insert` - Document created
- `update` - Document updated
- `delete` - Document deleted

## API Reference

### Databases

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/databases` | None | Create a new database |
| DELETE | `/api/databases/{id}` | Write | Delete database |
| GET | `/api/databases/{id}/events` | Read/Write | SSE stream (all events) |

### Schemas

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| POST | `/api/databases/{id}/schemas/{name}` | Write | Create schema |
| DELETE | `/api/databases/{id}/schemas/{name}` | Write | Delete schema |

### Documents

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| GET | `/api/databases/{id}/{collection}/` | Read/Write | Query documents |
| POST | `/api/databases/{id}/{collection}/` | Write | Insert document |
| PUT | `/api/databases/{id}/{collection}/{docId}` | Write | Update document |
| DELETE | `/api/databases/{id}/{collection}/{docId}` | Write | Delete document |
| GET | `/api/databases/{id}/{collection}/events` | Read/Write | SSE stream (collection) |

## Configuration

Configure via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP server port |
| `DB_BASE_DIR` | `./data` | Base directory for database files |
| `CATALOG_DB_PATH` | `./data/catalog.db` | Catalog database path |
| `CORS_ORIGINS` | `*` | Allowed CORS origins (comma-separated) |
| `DEFAULT_QUOTA_MB` | `100` | Default quota per database (MB) |
| `EXPIRY_DAYS` | `30` | Days before inactive database expires |
| `EXPIRY_CHECK_INTERVAL` | `24h` | How often to check for expired databases |

**Example:**

```bash
PORT=3000 \
CORS_ORIGINS="https://example.com,https://app.example.com" \
DEFAULT_QUOTA_MB=250 \
go run cmd/server/main.go
```

## Architecture

- **Language:** Go 1.24
- **Router:** Chi v5
- **Database:** SQLite (one file per database + catalog)
- **Authentication:** API keys (read-only and read-write)
- **Real-Time:** Server-Sent Events (SSE)

### Project Structure

```
jsondrop/
├── cmd/server/          # Main entry point
├── internal/
│   ├── api/            # HTTP handlers and routing
│   ├── config/         # Configuration management
│   ├── database/       # SQLite operations
│   ├── events/         # SSE broadcasting
│   └── models/         # Data structures
├── Dockerfile          # Multi-stage Docker build
├── docker-compose.yml  # Docker Compose configuration
└── CLAUDE.md          # Development guidelines
```

## Development

### Prerequisites

- Go 1.24 or later
- SQLite3
- Docker (optional)

### Running Tests

```bash
go test ./...
```

### Building from Source

```bash
go build -o bin/jsondrop cmd/server/main.go
./bin/jsondrop
```

## Security Considerations

- **API Keys:** Treat write keys as secrets. They provide full database access.
- **Read Keys:** Can query data and listen to events, but cannot modify.
- **CORS:** Configure `CORS_ORIGINS` properly for production (don't use `*`).
- **Rate Limiting:** Handle externally (e.g., via reverse proxy like Traefik).
- **Quota Enforcement:** Prevents abuse through storage limits.
- **Auto-Expiry:** Automatically cleans up inactive databases.

## Production Deployment

### With Docker Compose

1. Update `CORS_ORIGINS` in `docker-compose.yml`
2. Configure quotas and expiry as needed
3. Run: `docker-compose up -d`

### With Reverse Proxy

Use Traefik, Nginx, or Caddy for:
- SSL/TLS termination
- Rate limiting
- Load balancing
- Access logs

### Example Nginx Configuration

```nginx
upstream jsondrop {
    server localhost:8080;
}

server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://jsondrop;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;

        # For SSE support
        proxy_buffering off;
        proxy_cache off;
        proxy_set_header Connection '';
        proxy_http_version 1.1;
        chunked_transfer_encoding off;
    }
}
```

## Use Cases

- **Prototyping:** Quick backend for frontend development
- **Temporary Data Storage:** Forms, surveys, event registration
- **Real-Time Dashboards:** Live data updates via SSE
- **Testing:** Mock data for integration tests
- **Webhooks:** Store webhook payloads temporarily
- **IoT Data Collection:** Sensor data with automatic cleanup

## Limitations

- **Storage:** Limited by quota (default 100MB per database)
- **Filtering:** In-memory filtering (not optimized for large datasets)
- **No Indexes:** No custom indexes on JSON fields
- **Single Server:** No built-in clustering or replication
- **Temporary:** Databases expire after inactivity

## Roadmap

- [ ] Custom indexes on JSON fields
- [ ] Advanced query operators ($gt, $lt, $regex)
- [ ] Webhooks for change notifications
- [ ] Database backup/restore API
- [ ] Multi-region support
- [ ] GraphQL endpoint

## Contributing

Contributions welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

[Your chosen license]

## Support

- **Issues:** [GitHub Issues](https://github.com/yourusername/jsondrop/issues)
- **Documentation:** See `CLAUDE.md` for development guidelines
- **Questions:** Open a discussion or issue

