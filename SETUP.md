# Uptime Kabomba (Go + Next.js) - Setup Guide

Complete setup guide for the Go + Next.js rewrite of Uptime Kabomba.

## Prerequisites

- **Go** 1.23 or higher ([download](https://go.dev/dl/))
- **Node.js** 20 or higher ([download](https://nodejs.org/))
- **Git**
- **Make** (optional, for convenience commands)
- **Docker** (optional, for containerized development)

## Quick Start (Local Development)

### 1. Clone and Setup

```bash
# Clone the repository
cd /Users/fuomag9/Documents/Dev/uptime-kabomba

# Install dependencies
make setup

# Or manually:
go mod download
cd web && npm install
```

### 2. Configure Environment

```bash
# Copy environment template
cd web
cp .env.local.example .env.local

# Edit .env.local if needed (defaults should work)
```

### 3. Run Development Servers

**Option A: Using Make (Recommended)**
```bash
make dev
```

**Option B: Manual (Two terminals)**

Terminal 1 - Backend:
```bash
go run cmd/server/main.go
```

Terminal 2 - Frontend:
```bash
cd web
npm run dev
```

### 4. Access the Application

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080/api
- **Health Check**: http://localhost:8080/health
- **WebSocket**: ws://localhost:8080/ws

### 5. Initial Setup

1. Visit http://localhost:3000
2. Click "Get Started" or navigate to http://localhost:3000/setup
3. Create your admin account
4. You'll be redirected to the dashboard

## Docker Development

### Using Docker Compose

```bash
# Start all services (PostgreSQL + Backend + Frontend)
make docker-up

# Or manually:
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
make docker-down
```

Services:
- **Frontend**: http://localhost:3000
- **Backend**: http://localhost:8080
- **PostgreSQL**: localhost:5432

## Project Structure

```
uptime-kabomba/
├── cmd/server/              # Go backend entry point
├── internal/                # Go backend code
│   ├── api/                # HTTP handlers & routing
│   ├── auth/               # Authentication logic
│   ├── config/             # Configuration
│   ├── database/           # Database layer
│   ├── jobs/               # Background jobs
│   ├── models/             # Data models
│   └── websocket/          # WebSocket hub
├── migrations/             # Database migrations
├── web/                    # Next.js frontend
│   ├── app/               # Next.js App Router
│   ├── components/        # React components
│   ├── hooks/             # Custom React hooks
│   └── lib/               # Utilities & API client
├── go.mod                  # Go dependencies
├── Makefile               # Build commands
└── docker-compose.yml     # Docker setup
```

## Environment Variables

### Backend (Go)

Create a `.env` file in the root directory:

```bash
# Server
PORT=8080
ENVIRONMENT=development

# Database (PostgreSQL)
DB_TYPE=postgres
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=uptime
POSTGRES_USER=uptime
POSTGRES_PASSWORD=secret
POSTGRES_SSLMODE=disable

# Database Connection Pool
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# Security
JWT_SECRET=change-this-secret-in-production
```

### Frontend (Next.js)

Create `web/.env.local`:

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
NEXT_PUBLIC_WS_URL=ws://localhost:8080/ws
```

## Database Setup

1. **Install PostgreSQL**
   ```bash
   # macOS
   brew install postgresql@16
   brew services start postgresql@16

   # Or use Docker
   docker run -d \
     --name postgres \
     -e POSTGRES_USER=kuma \
     -e POSTGRES_PASSWORD=kuma \
     -e POSTGRES_DB=kuma \
     -p 5432:5432 \
     postgres:16-alpine
   ```

2. **Update environment variables**
   ```bash
   DB_TYPE=postgres
POSTGRES_HOST=localhost
POSTGRES_PORT=5432
POSTGRES_DB=uptime
POSTGRES_USER=uptime
POSTGRES_PASSWORD=secret
POSTGRES_SSLMODE=disable
   ```

3. **Restart backend**
   ```bash
   make dev-backend
   ```

## API Endpoints

### Authentication
- `POST /api/auth/setup` - Initial setup (create first user)
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout

### Protected Endpoints (require JWT)
- `GET /api/user/me` - Get current user
- `GET /api/monitors` - List monitors
- `POST /api/monitors` - Create monitor
- `GET /api/notifications` - List notifications

### WebSocket
- `GET /ws` - WebSocket connection for real-time updates

### Health
- `GET /health` - Health check

## WebSocket Protocol

Connect to `ws://localhost:8080/ws` with optional JWT token.

**Message Format:**
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
- `connected` - Connection established
- `disconnected` - Connection lost

## Development Commands

```bash
# Setup dependencies
make setup

# Run development servers
make dev

# Build production binaries
make build

# Run tests
make test

# Lint code
make lint

# Clean build artifacts
make clean

# Docker commands
make docker-up      # Start Docker services
make docker-down    # Stop Docker services
make docker-build   # Build Docker image
```

## Testing

### Backend Tests
```bash
make test-backend
# Or
go test -v ./...
```

### Frontend Tests
```bash
cd web
npm test
```

### E2E Tests
```bash
cd web
npm run test:e2e
```

## Building for Production

### Backend
```bash
go build -o uptime-kabomba-go cmd/server/main.go
./uptime-kabomba-go
```

### Frontend
```bash
cd web
npm run build
npm start
```

### Docker Production Build
```bash
docker build -t uptime-kabomba-go:latest .
docker run -p 8080:8080 -v $(pwd)/data:/app/data uptime-kabomba-go:latest
```

## Troubleshooting

### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080
# Kill the process
kill -9 <PID>
```

### Database Migration Issues
```bash
# Reset database (WARNING: deletes all data)
rm -f data/kuma.db*
# Restart backend to recreate
make dev-backend
```

### WebSocket Connection Issues
- Check that backend is running on port 8080
- Verify `NEXT_PUBLIC_WS_URL` in `web/.env.local`
- Check browser console for WebSocket errors

### Go Module Issues
```bash
go clean -modcache
go mod download
```

### Node Module Issues
```bash
cd web
rm -rf node_modules package-lock.json
npm install
```

## What's Working (Phase 1 Complete ✅)

- ✅ Go backend with Chi router
- ✅ JWT authentication (login, logout, setup)
- ✅ WebSocket real-time communication
- ✅ Database support (PostgreSQL)
- ✅ Automated migrations
- ✅ Background job scheduler
- ✅ Next.js 15 frontend
- ✅ Tailwind CSS + dark mode
- ✅ TanStack Query for data fetching
- ✅ WebSocket client hooks
- ✅ Authentication pages (login, setup)
- ✅ Dashboard layout

## What's Next (Phase 2 - Monitor Types)

- [ ] HTTP/HTTPS monitor
- [ ] TCP monitor
- [ ] Ping monitor
- [ ] DNS monitor
- [ ] Docker monitor
- [ ] Database monitors
- [ ] More monitor types...

## Contributing

This is a complete rewrite of the original Uptime Kabomba. See the main plan at:
`/Users/fuomag9/.claude/plans/recursive-wobbling-flame.md`

## License

MIT (same as original Uptime Kabomba)

## Credits

Original Uptime Kabomba by Louis Lam: https://github.com/louislam/uptime-kabomba

This rewrite uses Go and Next.js for improved performance and modern development experience.
