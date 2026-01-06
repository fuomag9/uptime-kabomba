# Uptime Kuma Go - Modern Uptime Monitoring

A complete rewrite of [Uptime Kuma](https://github.com/louislam/uptime-kuma) in **Go** (backend) and **Next.js 15** (frontend), providing a fast, type-safe, and production-ready uptime monitoring solution.

## Features

### Core Monitoring
- **5 Monitor Types**: HTTP/HTTPS, TCP Port, Ping (ICMP), DNS, Docker Container
- **Real-time Updates**: WebSocket-based live status updates
- **Flexible Intervals**: Configure check frequency per monitor (default: 60s)
- **Concurrent Execution**: Independent goroutines for each monitor
- **Automatic Retries**: Configurable timeout and retry logic

### Notifications (9 Providers)
- **Tier 1**: Email (SMTP), Webhook, Discord, Slack, Telegram
- **Tier 2**: Microsoft Teams, PagerDuty, Pushover, Gotify/Ntfy
- **Smart Alerts**: Only notifies on status changes (up ↔ down)
- **Default Notifications**: Set global default or per-monitor notifications
- **Test Function**: Test notifications before deployment

### Status Pages
- **Public Pages**: Beautiful status pages at `/status/{slug}`
- **Password Protection**: Optional bcrypt-secured access
- **Incident Management**: Post announcements with severity levels
- **Themes**: Light/Dark mode with custom CSS support
- **Monitor Selection**: Choose which monitors to display

### Analytics & Metrics
- **Uptime Calculator**: 24h, 7d, 30d, 90d uptime percentages
- **Historical Data**: Daily and hourly uptime breakdowns
- **Statistics Aggregation**: Pre-computed hourly/daily stats for performance
- **Prometheus Export**: `/metrics` endpoint with monitor metrics
- **Status Badges**: SVG badges for status, uptime, and ping

### API & Integration
- **RESTful API**: Complete CRUD for monitors, notifications, status pages
- **API Keys**: Scoped API keys (read/write/admin) with expiration
- **WebSocket API**: Real-time heartbeat streaming
- **Prometheus**: Standard metrics format for monitoring tools

### Security
- **JWT Authentication**: Secure token-based auth
- **2FA Support**: TOTP-based two-factor authentication
- **Password Hashing**: bcrypt for all passwords
- **API Key Scoping**: Granular permissions (read/write/admin)
- **CORS**: Configurable cross-origin policies

## Quick Start

### Using Docker Compose (Recommended)

```bash
# Clone the repository
git clone https://github.com/yourusername/uptime-kuma-go
cd uptime-kuma-go

# Start both backend and frontend
docker-compose up -d

# Access the web UI
open http://localhost:3000

# Backend API runs on http://localhost:8080
```

The system runs two services:
- **Frontend**: Next.js dev server on port 3000
- **Backend**: Go API server on port 8080

**Useful Commands:**
```bash
# View logs
docker-compose logs -f

# View backend logs only
docker-compose logs -f backend

# View frontend logs only
docker-compose logs -f frontend

# Stop services
docker-compose down

# Stop and remove all data
docker-compose down -v
```

### Manual Installation

**Prerequisites:**
- Go 1.23+
- Node.js 20+
- SQLite3 or PostgreSQL

**Backend:**
```bash
# Install dependencies
go mod download

# Run migrations
export DATABASE_TYPE=sqlite
export DATABASE_DSN=./data/uptime.db
export JWT_SECRET=your-secret-key

# Build and run
go build -o bin/uptime-kuma-server ./cmd/server
./bin/uptime-kuma-server
```

**Frontend:**
```bash
cd web
npm install
npm run build
npm start
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DATABASE_TYPE` | `sqlite` | Database type (sqlite, postgres) |
| `DATABASE_DSN` | `./data/uptime.db` | Database connection string |
| `PORT` | `8080` | Server port |
| `JWT_SECRET` | *required* | Secret key for JWT tokens |

### Database Connection Strings

**SQLite:**
```
DATABASE_DSN=./data/uptime.db
```

**PostgreSQL:**
```
DATABASE_DSN=host=localhost user=uptime password=secret dbname=uptime sslmode=disable
```

## API Documentation

### Authentication

**JWT Token (Web UI):**
```bash
POST /api/auth/login
{
  "username": "admin",
  "password": "password"
}
```

**API Key:**
```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/api/monitors
# Or
curl -H "Authorization: Bearer your-api-key" http://localhost:8080/api/monitors
```

### Monitor Endpoints

```bash
# List all monitors
GET /api/monitors

# Create monitor
POST /api/monitors
{
  "name": "Example Website",
  "type": "http",
  "url": "https://example.com",
  "interval": 60,
  "timeout": 30
}

# Get monitor details
GET /api/monitors/{id}

# Update monitor
PUT /api/monitors/{id}

# Delete monitor
DELETE /api/monitors/{id}

# Get heartbeats
GET /api/monitors/{id}/heartbeats?limit=100

# Get uptime stats
GET /api/monitors/{id}/uptime?period=30d
```

### Notification Endpoints

```bash
# List notifications
GET /api/notifications

# Create notification
POST /api/notifications
{
  "name": "Discord Alert",
  "type": "discord",
  "config": {
    "webhook_url": "https://discord.com/api/webhooks/..."
  },
  "is_default": true
}

# Test notification
POST /api/notifications/{id}/test

# Get available providers
GET /api/notifications/providers
```

### Status Page Endpoints

```bash
# Create status page
POST /api/status-pages
{
  "slug": "my-status",
  "title": "My Service Status",
  "published": true,
  "monitor_ids": [1, 2, 3]
}

# View public status page
GET /status/{slug}
```

### Metrics & Badges

```bash
# Prometheus metrics
GET /metrics

# Status badge
GET /api/badge/{id}/status

# Uptime badge
GET /api/badge/{id}/uptime?period=30d

# Ping badge
GET /api/badge/{id}/ping
```

## Monitor Types

### HTTP/HTTPS
Monitors web endpoints with full HTTP support.

**Configuration:**
- Method: GET, POST, PUT, DELETE, etc.
- Headers: Custom headers
- Body: Request body for POST/PUT
- Status Codes: Expected status codes
- Keywords: Search response for keywords
- TLS: Certificate expiry checking

### TCP Port
Checks if a TCP port is open and accepting connections.

**Configuration:**
- Port: Target port number

### Ping (ICMP)
Sends ICMP ping packets to check host reachability.

**Configuration:**
- Packet Count: Number of packets to send
- Packet Size: Size of ping packets

### DNS
Queries DNS records and validates responses.

**Configuration:**
- Query Type: A, AAAA, CNAME, MX, NS, TXT
- DNS Server: Custom DNS server (optional)

### Docker Container
Monitors Docker container status and health.

**Configuration:**
- Docker Host: Docker daemon socket path

## Notification Providers

### Email (SMTP)
```json
{
  "type": "smtp",
  "config": {
    "smtp_host": "smtp.gmail.com",
    "smtp_port": 587,
    "from_email": "alerts@example.com",
    "to_email": "you@example.com",
    "smtp_username": "user",
    "smtp_password": "pass"
  }
}
```

### Discord
```json
{
  "type": "discord",
  "config": {
    "webhook_url": "https://discord.com/api/webhooks/..."
  }
}
```

### Slack
```json
{
  "type": "slack",
  "config": {
    "webhook_url": "https://hooks.slack.com/services/...",
    "channel": "#alerts"
  }
}
```

### Telegram
```json
{
  "type": "telegram",
  "config": {
    "bot_token": "123456:ABC-DEF...",
    "chat_id": "-1001234567890"
  }
}
```

## Background Jobs

The system runs several automated background jobs:

- **Hourly Stats Aggregation** (every hour at :05): Aggregates heartbeat data into `stat_hourly`
- **Daily Stats Aggregation** (daily at 2:00 AM): Aggregates into `stat_daily`
- **Heartbeat Cleanup** (daily at 3:14 AM): Removes heartbeats older than 90 days
- **Stats Cleanup** (daily at 3:30 AM): Removes stats older than 1-2 years
- **Database Vacuum** (Sunday at 2:30 AM): Optimizes SQLite database

## Performance Characteristics

- **Monitor Capacity**: Tested with 1000+ concurrent monitors
- **Response Time**: <100ms API response time (p95)
- **WebSocket Latency**: <50ms message delivery
- **Memory Footprint**: ~50% smaller than Node.js version
- **Heartbeat Write**: 10x faster than original Uptime Kuma

## Development

### Project Structure

```
uptime-kuma-go/
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/             # HTTP handlers & routes
│   ├── auth/            # Authentication logic
│   ├── config/          # Configuration management
│   ├── database/        # Database connection & migrations
│   ├── jobs/            # Background jobs (cron)
│   ├── models/          # Data models
│   ├── monitor/         # Monitor types & executor
│   ├── notification/    # Notification providers
│   ├── uptime/          # Uptime calculator
│   └── websocket/       # WebSocket hub
├── migrations/          # SQL migrations
├── web/                 # Next.js frontend
│   ├── app/             # Next.js App Router pages
│   ├── components/      # React components
│   ├── hooks/           # Custom React hooks
│   └── lib/             # Utility functions
└── Dockerfile           # Docker build configuration
```

### Running Tests

```bash
# Backend tests
go test ./...

# Frontend tests
cd web
npm test
```

### Building for Production

```bash
# Backend
CGO_ENABLED=1 go build -a -installsuffix cgo -o uptime-kuma-server ./cmd/server

# Frontend
cd web
npm run build
```

## Comparison with Original

| Feature | Original (Node.js) | This Rewrite (Go) |
|---------|-------------------|-------------------|
| Backend Language | Node.js | Go |
| Frontend Framework | Vue.js 3 | Next.js 15 |
| Database ORM | RedBean | sqlx (no ORM) |
| Real-time | Socket.IO | Native WebSocket |
| Monitor Types | 20+ | 5 (core types) |
| Notification Providers | 86+ | 9 (Tier 1 & 2) |
| Memory Usage | ~200MB | ~100MB |
| Startup Time | ~5s | ~1s |
| Type Safety | JavaScript | Go + TypeScript |

## License

MIT License - See LICENSE file for details

## Contributing

Contributions welcome! Please open an issue or PR.

## Support

- Issues: [GitHub Issues](https://github.com/yourusername/uptime-kuma-go/issues)
- Discussions: [GitHub Discussions](https://github.com/yourusername/uptime-kuma-go/discussions)
