# Implementation Plan: Guest Lock PIN Manager

**Branch**: `001-guest-lock-pins` | **Date**: 2025-12-07 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/specs/001-guest-lock-pins/spec.md`

## Summary

A Home Assistant addon that automates door lock PIN management for short-term rental 
properties. The system subscribes to rental calendars (Airbnb, VRBO, etc.), extracts 
guest information, and automatically provisions temporary PINs on IOT locks. It also 
supports static recurring PINs for service personnel with day/time restrictions.

**Technical Approach**: Go backend API with SQLite for metadata persistence, TypeScript 
frontend with Bootstrap UI served from the same container. WebSocket communication 
enables real-time updates between backend and frontend. Multi-stage distroless Docker 
build for minimal container footprint. Integrates with Home Assistant Supervisor API, 
with optional direct Z-Wave JS UI and Zigbee2MQTT connections for battery efficiency.

## Technical Context

**Language/Version**: Go 1.22+ (backend), TypeScript 5.x (frontend)  
**Primary Dependencies**:
- Backend: `gorilla/websocket`, `mattn/go-sqlite3`, `robfig/cron` (scheduling)
- Frontend: Bootstrap 5.x, native WebSocket API
- Build: Multi-stage Docker with `gcr.io/distroless/static-debian12`

**Storage**: SQLite 3 (embedded, file-based at `/data/guest-lock-manager.db`)  
**Testing**: `go test` with table-driven tests, Jest for frontend  
**Target Platform**: Home Assistant OS / Supervised (linux/amd64, linux/arm64)  
**Project Type**: Web application (Go API backend + TypeScript SPA frontend)  
**Performance Goals**: 
- API response <100ms p95 for all endpoints
- Calendar sync processing <5s for 100 events
- Lock command batching within 30s window

**Constraints**: 
- Container image <50MB (distroless requirement)
- Memory <128MB runtime
- Battery-aware: minimize lock wake-ups

**Scale/Scope**: 
- 10 calendars, 20 locks, 100 concurrent reservations
- Single-user addon (property owner)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Gate | Status |
|-----------|------|--------|
| I. Code Quality First | DRY abstractions, readable code, meaningful comments | ✅ Pass |
| II. YAGNI & Simplicity | No speculative features, minimal dependencies | ✅ Pass |
| III. Continuous Refactoring | Small batches, visible tech debt | ✅ Pass |
| IV. Performance Excellence | <100ms p95, efficient algorithms, benchmarks | ✅ Pass |
| V. Documentation Standards | README, API docs, GoDoc comments | ✅ Pass |

**Go-Specific Compliance**:
- [x] Follow Effective Go conventions
- [x] Use gofmt and go vet
- [x] Explicit error handling (no panics in handlers)
- [x] Context propagation for cancellation
- [x] Domain-structured packages

**API Design Compliance**:
- [x] Resource-oriented URLs
- [x] Appropriate HTTP methods
- [x] Consistent error format
- [x] Idempotent operations
- [x] Pagination for lists

## Project Structure

### Documentation (this feature)

```text
specs/001-guest-lock-pins/
├── plan.md              # This file
├── research.md          # Phase 0: Technology decisions
├── data-model.md        # Phase 1: Entity definitions
├── quickstart.md        # Phase 1: Developer setup guide
├── contracts/           # Phase 1: API specifications
│   ├── openapi.yaml     # REST API contract
│   └── websocket.md     # WebSocket message schemas
└── tasks.md             # Phase 2: Implementation tasks
```

### Source Code (repository root)

```text
backend/
├── cmd/
│   └── server/
│       └── main.go           # Application entrypoint
├── internal/
│   ├── api/
│   │   ├── handlers/         # HTTP request handlers
│   │   ├── middleware/       # Auth, logging, error handling
│   │   └── router.go         # Route definitions
│   ├── calendar/
│   │   ├── ical.go           # iCal parsing
│   │   ├── sync.go           # Calendar sync logic
│   │   └── extractor.go      # PIN extraction from events
│   ├── lock/
│   │   ├── manager.go        # Lock operations orchestration
│   │   ├── homeassistant.go  # HA API integration
│   │   ├── zwave.go          # Z-Wave JS UI direct integration
│   │   └── zigbee.go         # Zigbee2MQTT direct integration
│   ├── pin/
│   │   ├── generator.go      # PIN generation strategies
│   │   ├── scheduler.go      # PIN activation/deactivation
│   │   └── conflict.go       # Collision detection
│   ├── storage/
│   │   ├── sqlite.go         # Database connection
│   │   ├── migrations/       # Schema migrations
│   │   └── repository.go     # Data access layer
│   └── websocket/
│       ├── hub.go            # WebSocket connection manager
│       └── messages.go       # Message type definitions
├── go.mod
├── go.sum
└── Dockerfile

frontend/
├── src/
│   ├── index.html
│   ├── main.ts               # Application bootstrap
│   ├── components/
│   │   ├── calendar-list.ts  # Calendar subscription UI
│   │   ├── lock-list.ts      # Lock management UI
│   │   ├── pin-table.ts      # PIN status display
│   │   └── static-pin-form.ts# Static PIN configuration
│   ├── services/
│   │   ├── api.ts            # REST API client
│   │   └── websocket.ts      # WebSocket client
│   └── styles/
│       └── main.scss         # Bootstrap customization
├── package.json
├── tsconfig.json
└── vite.config.ts

# Root level
├── Dockerfile                # Multi-stage build
├── config.yaml               # Home Assistant addon config
├── README.md
└── docs/
    └── adr/                  # Architecture Decision Records
```

**Structure Decision**: Web application pattern selected. Go backend serves both the 
REST API and static frontend assets. Single container deployment aligns with Home 
Assistant addon architecture. Frontend is a lightweight TypeScript SPA (no framework) 
with Bootstrap for rapid UI development.

## Complexity Tracking

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| WebSocket + REST hybrid | Real-time PIN status updates | Polling would increase API load and latency |
| Three lock integrations (HA, Z-Wave, Zigbee) | Battery efficiency requirement | HA-only would work but drain batteries faster |
| SQLite embedded DB | Persistence across addon restarts | In-memory would lose all configuration on restart |

## Dependencies

### External Services
- **Home Assistant Supervisor API**: Addon lifecycle, ingress routing
- **Home Assistant Core API**: Lock entity discovery and control
- **Z-Wave JS UI WebSocket** (optional): Direct Z-Wave communication
- **Zigbee2MQTT MQTT** (optional): Direct Zigbee communication
- **External Calendar URLs**: iCal feeds from rental platforms

### Go Packages (minimal set)
- `github.com/gorilla/mux` - HTTP router with path parameters
- `github.com/gorilla/websocket` - WebSocket server
- `github.com/mattn/go-sqlite3` - SQLite driver (CGO required)
- `github.com/robfig/cron/v3` - Cron-style scheduling
- Standard library for HTTP server, JSON encoding, iCal parsing

### Frontend Packages (minimal set)
- `bootstrap` - UI components and grid
- `vite` - Build tooling (dev dependency only)
- `typescript` - Type checking (dev dependency only)
