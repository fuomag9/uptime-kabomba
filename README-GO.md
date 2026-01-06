# Uptime Kuma (Go + Next.js Rewrite)

A modern rewrite of Uptime Kuma using Go for the backend and Next.js for the frontend.

## Architecture

- **Backend**: Go 1.23+ with Chi router
- **Frontend**: Next.js 15 with TypeScript
- **Database**: SQLite (default) or PostgreSQL
- **Real-time**: WebSocket (custom protocol)

## Project Structure

```
├── cmd/server/                # Main application entry point
├── internal/
│   ├── api/                  # HTTP handlers
│   ├── auth/                # Authentication logic
│   ├── config/              # Configuration management
│   ├── database/            # Database connection and migrations
│   ├── jobs/                # Background job scheduler
│   ├── models/              # Data models
│   ├── monitor/             # Monitor engine (Phase 2)
│   ├── notification/        # Notification providers (Phase 3)
│   ├── uptime/              # Uptime calculator (Phase 4)
│   └── websocket/           # WebSocket hub
├── migrations/              # SQL database migrations
└── web/                     # Next.js frontend (Phase 1)
```

## Development

### Prerequisites

- Go 1.23+
- Node.js 20+
- SQLite 3 or PostgreSQL 16+

### Setup

1. Install Go dependencies:
```bash
go mod download
```

2. Create data directory:
```bash
mkdir -p data
```

3. Run the server:
```bash
go run cmd/server/main.go
```

4. The server will start on port 8080 by default.

### Environment Variables

```bash
PORT=8080
DB_TYPE=sqlite                    # or postgres
DB_DSN=./data/kuma.db            # or postgresql://user:pass@localhost/kuma
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5
JWT_SECRET=your-secret-here
ENVIRONMENT=development          # or production
```

### Database Migrations

Migrations run automatically on startup. Migration files are in `migrations/`.

To create a new migration:
```bash
# Create up and down files manually in migrations/
# Format: {version}_{description}.up.sql and {version}_{description}.down.sql
```

### API Endpoints

**Authentication:**
- `POST /api/auth/setup` - Initial setup (create first user)
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout

**Protected (require JWT):**
- `GET /api/user/me` - Get current user
- `GET /api/monitors` - List monitors (placeholder)
- `POST /api/monitors` - Create monitor (placeholder)
- `GET /api/notifications` - List notifications (placeholder)

**WebSocket:**
- `GET /ws` - WebSocket endpoint for real-time updates

**Health:**
- `GET /health` - Health check

### WebSocket Protocol

Messages are JSON with this format:
```json
{
  "type": "message_type",
  "payload": { ... }
}
```

**Client → Server:**
- `subscribe` - Subscribe to monitor updates
- `unsubscribe` - Unsubscribe from updates
- `ping` - Keep-alive ping

**Server → Client:**
- `heartbeat` - Monitor heartbeat update
- `monitorList` - List of monitors
- `pong` - Ping response

## Implementation Status

### Phase 1: Foundation ✅ (Current)
- [x] Go project setup
- [x] Chi router with middleware
- [x] Database layer (SQLite + PostgreSQL)
- [x] JWT authentication
- [x] WebSocket hub
- [x] Background job scheduler
- [ ] Next.js frontend (next step)
- [ ] 2FA implementation

### Phase 2: Monitor Types (Weeks 7-16)
- [ ] HTTP/HTTPS monitor
- [ ] TCP monitor
- [ ] Ping monitor
- [ ] DNS monitor
- [ ] Docker monitor
- [ ] Database monitors (PostgreSQL, MySQL, MongoDB, Redis, MS SQL)
- [ ] Messaging monitors (MQTT, WebSocket, gRPC, RabbitMQ)
- [ ] Specialized monitors (SNMP, GameDig, Real Browser, System Service)

### Phase 3: Notifications (Weeks 17-22)
- [ ] Notification provider interface
- [ ] Top 10 providers (Discord, Slack, Telegram, Email, etc.)
- [ ] Remaining 76 providers

### Phase 4: Advanced Features (Weeks 23-30)
- [ ] Status pages
- [ ] API endpoints
- [ ] Prometheus metrics
- [ ] Uptime calculator
- [ ] Performance optimization

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/api/...
```

## Building

```bash
# Build binary
go build -o uptime-kuma-go cmd/server/main.go

# Run binary
./uptime-kuma-go
```

## Docker (Coming Soon)

```bash
# Build image
docker build -t uptime-kuma-go .

# Run container
docker run -p 8080:8080 -v $(pwd)/data:/app/data uptime-kuma-go
```

## License

MIT (same as original Uptime Kuma)

## Credits

Original Uptime Kuma by Louis Lam: https://github.com/louislam/uptime-kuma

This is a complete rewrite using Go and Next.js for modern performance and maintainability.
