# Guest Lock PIN Manager

A Home Assistant addon that automates door lock PIN management for short-term rental properties.

## Features

- **Calendar Integration**: Subscribe to Airbnb, VRBO, or any iCal calendar to auto-provision guest PINs
- **Smart PIN Generation**: Generate PINs from phone numbers, dates, or custom values
- **Static PINs**: Create recurring access codes for cleaners and maintenance with day/time restrictions
- **Multi-Lock Support**: Manage multiple locks across properties with independent calendar routing
- **Battery Efficient**: Direct Z-Wave JS UI and Zigbee2MQTT integration to minimize lock wake-ups
- **Real-Time Updates**: WebSocket-powered UI for instant status changes

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                     Home Assistant Supervisor                     │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────┐    │
│  │              Guest Lock PIN Manager Addon                │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │    │
│  │  │  Frontend   │  │   Go API    │  │    SQLite DB    │  │    │
│  │  │ (Bootstrap) │◄─┤ (REST + WS) │◄─┤   (Metadata)    │  │    │
│  │  └─────────────┘  └──────┬──────┘  └─────────────────┘  │    │
│  └──────────────────────────┼──────────────────────────────┘    │
│                             │                                    │
│  ┌──────────────────────────▼──────────────────────────────┐    │
│  │                   Home Assistant Core                    │    │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │    │
│  │  │ Lock Entity │  │ Lock Entity │  │  Lock Entity    │  │    │
│  │  │  (Z-Wave)   │  │  (Zigbee)   │  │    (WiFi)       │  │    │
│  │  └─────────────┘  └─────────────┘  └─────────────────┘  │    │
│  └─────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────┘
```

## Quick Start

### Installation

1. Add the repository to your Home Assistant addon store
2. Install "Guest Lock PIN Manager"
3. Start the addon
4. Open the web UI via the sidebar

### Configure a Calendar

1. Navigate to **Calendars** in the addon UI
2. Click **Add Calendar**
3. Paste your iCal URL from Airbnb/VRBO
4. Select which locks should receive guest PINs
5. Save and sync

### PIN Generation Priority

The addon generates PINs using this priority order:

1. **Custom Override**: Manually specified PIN for a reservation
2. **Phone Last-4**: Extracted from `(Last 4 Digits): XXXX` in event description
3. **Description Hash**: Deterministic code from event description
4. **Date-Based**: Check-in + checkout days (always succeeds)

## Development

### Prerequisites

- Go 1.22+
- Node.js 20+
- Docker (for building container)

### Local Development

```bash
# Backend
cd backend
go mod download
go run ./cmd/server

# Frontend (separate terminal)
cd frontend
npm install
npm run dev
```

### Building Docker Image

```bash
docker build -t guest-lock-manager:dev .
```

### Project Structure

```
├── backend/
│   ├── cmd/server/          # Application entrypoint
│   └── internal/
│       ├── api/             # REST handlers and middleware
│       ├── calendar/        # iCal parsing and sync
│       ├── lock/            # HA integration, Z-Wave, Zigbee
│       ├── pin/             # PIN generation and scheduling
│       ├── storage/         # SQLite repositories
│       └── websocket/       # Real-time updates
├── frontend/
│   └── src/
│       ├── components/      # UI components
│       ├── services/        # API and WebSocket clients
│       └── styles/          # Bootstrap customization
├── config.yaml              # Home Assistant addon config
├── Dockerfile               # Multi-stage distroless build
└── specs/                   # Feature specifications
```

## API Documentation

See [contracts/openapi.yaml](specs/001-guest-lock-pins/contracts/openapi.yaml) for the complete REST API specification.

## License

MIT

