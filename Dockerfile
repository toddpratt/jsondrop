# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

# Install build dependencies (gcc and musl-dev for SQLite CGO)
RUN apk add --no-cache gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary with CGO enabled for SQLite
# Use static linking for a more portable binary
RUN CGO_ENABLED=1 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags '-extldflags "-static"' \
    -o jsondrop \
    ./cmd/server/main.go

# Stage 2: Create minimal runtime image
FROM alpine:latest

# Install ca-certificates for HTTPS and sqlite runtime
RUN apk --no-cache add ca-certificates sqlite-libs

# Create a non-root user
RUN addgroup -g 1000 jsondrop && \
    adduser -D -u 1000 -G jsondrop jsondrop

# Set working directory
WORKDIR /app

# Create data directory with proper permissions
RUN mkdir -p /app/data && \
    chown -R jsondrop:jsondrop /app

# Copy binary from builder
COPY --from=builder /build/jsondrop /app/jsondrop

# Switch to non-root user
USER jsondrop

# Expose port
EXPOSE 8080

# Set default environment variables
ENV PORT=8080 \
    DB_BASE_DIR=/app/data \
    CATALOG_DB_PATH=/app/data/catalog.db \
    CORS_ORIGINS=* \
    DEFAULT_QUOTA_MB=100 \
    EXPIRY_DAYS=30 \
    EXPIRY_CHECK_INTERVAL=24h

# Run the binary
CMD ["/app/jsondrop"]
