# Quickstart: Guest Lock PIN Manager

**Feature**: 001-guest-lock-pins  
**Date**: 2025-12-07

## Prerequisites

### Development Environment

- **Go**: 1.22 or later
- **Node.js**: 20 LTS or later
- **Docker**: 24.0 or later (with BuildKit enabled)
- **Make**: For running build commands (optional but recommended)

### Home Assistant (for integration testing)

- Home Assistant OS 12.0+ or Supervised installation
- At least one lock entity configured (Z-Wave, Zigbee, or WiFi)
- Supervisor API access (automatic for addons)

## Repository Setup

```bash
# Clone the repository
git clone https://github.com/[org]/guest-lock-manager.git
cd guest-lock-manager

# Switch to feature branch
git checkout 001-guest-lock-pins
```

## Backend Development

### Initial Setup

```bash
cd backend

# Download dependencies
go mod download

# Verify build
go build ./...

# Run tests
go test ./...
```

### Running Locally

```bash
# Set environment variables for local development
export DATABASE_PATH="./data/dev.db"
export HA_URL="http://homeassistant.local:8123"
export HA_TOKEN="your-long-lived-access-token"
export LOG_LEVEL="debug"

# Run the server
go run ./cmd/server

# Server starts on http://localhost:8099
```

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATABASE_PATH` | Yes | `/data/guest-lock-manager.db` | SQLite database file path |
| `HA_URL` | Yes | - | Home Assistant base URL |
| `HA_TOKEN` | Yes | - | Long-lived access token |
| `LOG_LEVEL` | No | `info` | Log level (debug, info, warn, error) |
| `PORT` | No | `8099` | HTTP server port |
| `ZWAVE_JS_URL` | No | - | Z-Wave JS UI WebSocket URL |
| `MQTT_BROKER` | No | - | MQTT broker URL for Zigbee2MQTT |

### Code Structure

```
backend/
├── cmd/server/main.go      # Entrypoint - start here
├── internal/
│   ├── api/                # HTTP handlers and routing
│   ├── calendar/           # iCal sync and parsing
│   ├── lock/               # Lock integrations
│   ├── pin/                # PIN generation logic
│   ├── storage/            # SQLite repository
│   └── websocket/          # Real-time updates
```

### Key Files to Understand First

1. `cmd/server/main.go` - Application bootstrap and dependency wiring
2. `internal/api/router.go` - All API routes defined here
3. `internal/storage/repository.go` - Data access patterns
4. `internal/pin/generator.go` - PIN generation strategies

## Frontend Development

### Initial Setup

```bash
cd frontend

# Install dependencies
npm install

# Start development server with HMR
npm run dev

# Frontend available at http://localhost:5173
```

### Connecting to Backend

Create `.env.local` for development:

```env
VITE_API_URL=http://localhost:8099/api
VITE_WS_URL=ws://localhost:8099/api/ws
```

### Code Structure

```
frontend/
├── src/
│   ├── index.html          # Single page entry
│   ├── main.ts             # Bootstrap and routing
│   ├── components/         # UI components
│   ├── services/           # API and WebSocket clients
│   └── styles/             # SCSS with Bootstrap
```

### Building for Production

```bash
npm run build
# Output in dist/ - copied to backend during Docker build
```

## Docker Development

### Building the Image

```bash
# From repository root
docker build -t guest-lock-manager:dev .

# Build for specific platform (e.g., Raspberry Pi)
docker build --platform linux/arm64 -t guest-lock-manager:dev-arm64 .
```

### Running Locally

```bash
docker run -d \
  --name glm-dev \
  -p 8099:8099 \
  -v $(pwd)/data:/data \
  -e HA_URL="http://host.docker.internal:8123" \
  -e HA_TOKEN="your-token" \
  guest-lock-manager:dev
```

### Multi-arch Build (for release)

```bash
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t ghcr.io/[org]/guest-lock-manager:latest \
  --push .
```

## Testing

### Backend Tests

```bash
cd backend

# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/pin/...

# Run with race detector
go test -race ./...
```

### Frontend Tests

```bash
cd frontend

# Run Jest tests
npm test

# Run with coverage
npm run test:coverage
```

### Integration Tests

```bash
# Requires running Home Assistant instance
cd backend
HA_URL=http://localhost:8123 HA_TOKEN=xxx go test -tags=integration ./...
```

## Home Assistant Addon Development

### Local Addon Testing

1. Copy repository to HA `/addons/` directory:
   ```bash
   scp -r . root@homeassistant.local:/addons/guest-lock-manager
   ```

2. In Home Assistant:
   - Go to Settings → Add-ons → Add-on Store
   - Click ⋮ → Repositories → Check for updates
   - Find "Guest Lock Manager" under Local add-ons
   - Install and start

3. View logs:
   ```bash
   ha addons logs local_guest-lock-manager
   ```

### Addon Configuration

The addon is configured via `config.yaml` in the repository root:

```yaml
name: "Guest Lock Manager"
version: "0.1.0"
slug: "guest-lock-manager"
description: "Manage IOT lock PINs for short-term rentals"
arch:
  - amd64
  - aarch64
ingress: true
ingress_port: 8099
options:
  log_level: info
schema:
  log_level: str
```

## Development Workflow

### Making Changes

1. Create feature branch from `001-guest-lock-pins`
2. Make changes following [Constitution](/.specify/memory/constitution.md)
3. Run `go fmt` and `go vet` before committing
4. Write/update tests for changes
5. Update API docs if endpoints changed
6. Create PR with conventional commit message

### Code Quality Checks

```bash
# Backend
cd backend
go fmt ./...
go vet ./...
golangci-lint run

# Frontend
cd frontend
npm run lint
npm run type-check
```

## Troubleshooting

### Common Issues

**"Cannot connect to Home Assistant"**
- Verify `HA_URL` is accessible from your development machine
- Ensure token has correct permissions (long-lived access token)
- Check if HA is running and API is enabled

**"Database locked"**
- Only one process can write to SQLite at a time
- Stop other instances before starting development server

**"Lock not responding"**
- Check lock is online in Home Assistant
- Verify entity_id is correct
- For Z-Wave: check Z-Wave JS UI is running

**"Frontend not connecting to WebSocket"**
- Verify CORS is configured in backend for development origin
- Check browser console for WebSocket errors
- Ensure backend is running before starting frontend

## Next Steps

After setup is complete:

1. Review [spec.md](./spec.md) for feature requirements
2. Review [plan.md](./plan.md) for architecture decisions
3. Check [data-model.md](./data-model.md) for entity definitions
4. See [contracts/openapi.yaml](./contracts/openapi.yaml) for API specification
5. Run `/speckit.tasks` to generate implementation task list



