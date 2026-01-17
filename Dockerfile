# Backend build stage
FROM golang:1.25-alpine AS backend-builder

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

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment variables
ENV DATABASE_TYPE=postgres \
    POSTGRES_HOST=postgres \
    POSTGRES_PORT=5432 \
    POSTGRES_DB=uptime \
    POSTGRES_USER=uptime \
    POSTGRES_PASSWORD=secret \
    POSTGRES_SSLMODE=disable \
    PORT=8080

# Run the application
CMD ["./uptime-kabomba-server"]
