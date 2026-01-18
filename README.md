# Uptime Kabomba - Modern Uptime Monitoring

A fast, type-safe, and production-ready uptime monitoring solution built with **Go** (backend) and **Next.js 15** (frontend).

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
git clone https://github.com/fuomag9/uptime-kabomba
cd uptime-kabomba

# Start both backend and frontend
docker-compose up -d

# Access the web UI
open http://localhost:3000
```

The system runs two services:
- **Frontend**: Next.js server on port 3000 (publicly accessible)
- **Backend**: Go API server (only accessible via Docker internal network)

**Architecture**: The backend is NOT exposed to the host - all API requests are proxied through the Next.js frontend. This ensures the backend is never directly accessible to users, improving security.

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
- PostgreSQL

**Backend:**
```bash
# Install dependencies
go mod download

# Run migrations
export DATABASE_TYPE=postgres
export POSTGRES_HOST=localhost
export POSTGRES_PORT=5432
export POSTGRES_DB=uptime
export POSTGRES_USER=uptime
export POSTGRES_PASSWORD=secret
export POSTGRES_SSLMODE=disable
export APP_URL=http://localhost:3000
export JWT_SECRET=your-secret-key

# Build and run
go build -o bin/uptime-kabomba-server ./cmd/server
./bin/uptime-kabomba-server
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
| `DATABASE_TYPE` | `postgres` | Database type (postgres) |
| `DATABASE_DSN` | *(optional)* | Database connection string (overrides POSTGRES_* vars) |
| `POSTGRES_HOST` | `localhost` | Postgres host |
| `POSTGRES_PORT` | `5432` | Postgres port |
| `POSTGRES_DB` | `uptime` | Postgres database name |
| `POSTGRES_USER` | `uptime` | Postgres user |
| `POSTGRES_PASSWORD` | `secret` | Postgres password |
| `POSTGRES_SSLMODE` | `disable` | Postgres SSL mode |
| `PORT` | `8080` | Backend internal port (not exposed to host) |
| `JWT_SECRET` | *required* | Secret key for JWT tokens |
| `APP_URL` | *(optional)* | Base URL for deriving CORS origins and OAuth redirect |
| `METRICS_TOKEN` | *required* | Token required to access `/metrics` |
| `HEALTH_TOKEN` | *required* | Token required to access `/health` |

### Database Connection Strings

**PostgreSQL (via env vars):**
```
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=uptime
POSTGRES_USER=uptime
POSTGRES_PASSWORD=secret
POSTGRES_SSLMODE=disable
```

**PostgreSQL (single DSN override):**
```
DATABASE_DSN=postgresql://uptime:secret@localhost:5432/uptime?sslmode=disable
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
curl -H "X-API-Key: your-api-key" http://localhost:3000/api/monitors
# Or
curl -H "Authorization: Bearer your-api-key" http://localhost:3000/api/monitors
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

## Performance Characteristics

- **Monitor Capacity**: Tested with 1000+ concurrent monitors
- **Response Time**: <100ms API response time (p95)
- **WebSocket Latency**: <50ms message delivery
- **Memory Footprint**: Efficient Go runtime (~100MB)
- **Heartbeat Write**: Optimized for high-throughput writes

## Development

### Project Structure

```
uptime-kabomba/
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
CGO_ENABLED=1 go build -a -installsuffix cgo -o uptime-kabomba-server ./cmd/server

# Frontend
cd web
npm run build
```

## License

MIT License - See LICENSE file for details

## Contributing

Contributions welcome! Please open an issue or PR.

## Support

- Issues: [GitHub Issues](https://github.com/fuomag9/uptime-kabomba/issues)
- Discussions: [GitHub Discussions](https://github.com/fuomag9/uptime-kabomba/discussions)
