# Backend build stage
FROM golang:1.24-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o uptime-kabomba-server ./cmd/server

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=backend-builder /app/uptime-kabomba-server .

# Copy migrations from builder
COPY --from=backend-builder /app/migrations ./migrations

# Create data directory for SQLite
RUN mkdir -p /data

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run as non-root user (disabled for now to avoid permissions issues)
# RUN addgroup -g 1000 uptime && \
#     adduser -D -u 1000 -G uptime uptime && \
#     chown -R uptime:uptime /app /data
# USER uptime

# Set environment variables
ENV DATABASE_TYPE=sqlite \
    DATABASE_DSN=/data/uptime.db \
    PORT=8080

# Run the application
CMD ["./uptime-kabomba-server"]
