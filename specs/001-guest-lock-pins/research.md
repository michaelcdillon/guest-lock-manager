# Research: Guest Lock PIN Manager

**Feature**: 001-guest-lock-pins  
**Date**: 2025-12-07  
**Status**: Complete

## Technology Decisions

### 1. Go Web Framework

**Decision**: Use standard library `net/http` with `gorilla/mux` for routing

**Rationale**: 
- Constitution requires minimal dependencies and lightweight code
- Standard library HTTP server is production-ready and well-documented
- gorilla/mux adds route parameters without heavy framework overhead
- Avoids gin/echo/fiber framework lock-in

**Alternatives Considered**:
- **Gin**: Popular but adds unnecessary abstraction for simple REST API
- **Echo**: Good performance but more dependencies than needed
- **Chi**: Lightweight but gorilla/mux has better WebSocket ecosystem integration

### 2. WebSocket Library

**Decision**: Use `gorilla/websocket`

**Rationale**:
- De facto standard for Go WebSocket implementations
- Well-maintained, production-proven
- Integrates cleanly with gorilla/mux router
- Supports ping/pong for connection health

**Alternatives Considered**:
- **nhooyr/websocket**: Modern API but less ecosystem adoption
- **gobwas/ws**: Lower-level, more complex for our needs

### 3. SQLite Driver

**Decision**: Use `mattn/go-sqlite3` (CGO-based)

**Rationale**:
- Most mature and feature-complete SQLite driver for Go
- Required for CGO-based distroless image (using static-debian12)
- Supports all SQLite features including JSON functions

**Alternatives Considered**:
- **modernc.org/sqlite**: Pure Go but less mature, larger binary
- **crawshaw/sqlite**: Good but less community adoption

**Build Implication**: Requires CGO_ENABLED=1 in Docker build. Using 
`gcr.io/distroless/static-debian12` which includes glibc for CGO compatibility.

### 4. Scheduling Library

**Decision**: Use `robfig/cron/v3`

**Rationale**:
- Standard cron expression support
- Timezone-aware scheduling (critical for check-in/out times)
- Minimal footprint, well-tested

**Alternatives Considered**:
- **go-co-op/gocron**: Higher-level but more dependencies
- **Custom ticker**: Too much boilerplate for complex schedules

### 5. Frontend Build Tool

**Decision**: Use Vite with vanilla TypeScript

**Rationale**:
- Fast development builds with HMR
- Produces optimized production bundles
- No framework overhead (React/Vue/Svelte not needed for simple UI)
- Tree-shaking removes unused Bootstrap components

**Alternatives Considered**:
- **esbuild direct**: Fast but less ecosystem tooling
- **Webpack**: Slower, more configuration
- **Parcel**: Good zero-config but larger output

### 6. Container Base Image

**Decision**: Multi-stage build with `gcr.io/distroless/static-debian12`

**Rationale**:
- Minimal attack surface (no shell, package manager)
- Small image size (<20MB for base)
- CGO compatible (includes glibc)
- Recommended by Google for Go production containers

**Alternatives Considered**:
- **Alpine**: Smaller but musl libc causes CGO issues with SQLite
- **scratch**: No glibc, can't use CGO SQLite driver
- **debian-slim**: Larger, includes unnecessary tools

### 7. Home Assistant Integration Approach

**Decision**: Supervisor API for addon lifecycle, Core API for lock control

**Rationale**:
- Supervisor API provides ingress, config, and addon management
- Core API is stable for entity discovery and service calls
- WebSocket API for real-time state subscriptions
- Follows official addon development guidelines

**Integration Points**:
1. `/api/supervisor/` - Addon configuration, health checks
2. `/api/` - Entity states, service calls (lock.set_usercode)
3. `/api/websocket` - Real-time lock state changes

### 8. Direct Protocol Integration

**Decision**: Optional Z-Wave JS UI WebSocket and Zigbee2MQTT MQTT clients

**Rationale**:
- Battery efficiency is critical requirement
- Direct integration allows PIN batching before lock communication
- Graceful fallback to HA API when direct integrations unavailable
- Z-Wave JS UI detected via `/api/hassio/addons` endpoint

**Z-Wave JS UI Integration**:
- WebSocket at `ws://[addon-host]:3000` (default port)
- Commands: `node.set_value` for usercode slots
- Supports batching multiple slot updates in single message

**Zigbee2MQTT Integration**:
- MQTT at `mqtt://[broker]:1883` 
- Topic: `zigbee2mqtt/[device]/set` with `pin_code` payload
- Native batching via single JSON message with multiple codes

## Research: iCal Parsing

**Decision**: Use Go standard library with custom parser

**Rationale**:
- iCal (RFC 5545) is text-based, simple to parse
- External libraries add dependencies for minimal gain
- Custom parser allows extraction of non-standard fields (phone numbers)

**Parsing Strategy**:
```
1. Fetch iCal URL with timeout and retry
2. Parse VEVENT blocks
3. Extract: SUMMARY, DTSTART, DTEND, DESCRIPTION
4. Apply regex patterns for phone/PIN extraction
5. Normalize dates to property timezone
```

**Phone Extraction Patterns**:
```regex
# Pattern 1: Explicit format
(?:Last 4 Digits|Last Four|Phone):\s*(\d{4})

# Pattern 2: Phone number last 4
\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?(\d{4})
```

## Research: PIN Generation Algorithms

### Date-Based Code (Fallback)

**Algorithm**: Combine check-in day and check-out day
```
pin = sprintf("%02d%02d", checkin.Day(), checkout.Day())
// Dec 15 - Dec 18 → "1518"
```

**Edge Cases**:
- Same-day checkout: Use hours instead (`1518` where 15=3pm, 18=6pm)
- Single digits: Zero-pad to ensure 4+ digits

### Description-Based Random

**Algorithm**: Deterministic hash of event description
```
hash = sha256(description)
pin = strconv.Itoa(binary.BigEndian.Uint32(hash[:4]) % 10000)
// Pad to 4 digits minimum
```

**Stability**: Same description always produces same PIN. Description 
changes trigger PIN regeneration (with user warning).

## Research: Lock Protocol Comparison

| Protocol | Wake Frequency | Batching | Latency |
|----------|----------------|----------|---------|
| Home Assistant API | Per-command | No | ~500ms |
| Z-Wave JS UI Direct | Per-batch | Yes | ~200ms |
| Zigbee2MQTT Direct | Per-batch | Yes | ~100ms |

**Recommendation**: Implement HA API first, add direct protocols as P5 
optimization. Track battery levels to measure improvement.

## Unresolved Items

None. All technical decisions resolved for Phase 1 design.

## References

- [Home Assistant Addon Development](https://developers.home-assistant.io/docs/add-ons/)
- [Z-Wave JS UI API](https://zwave-js.github.io/zwave-js-ui/#/guide/mqtt)
- [Zigbee2MQTT API](https://www.zigbee2mqtt.io/guide/usage/mqtt_topics_and_messages.html)
- [gorilla/websocket](https://github.com/gorilla/websocket)
- [Distroless Images](https://github.com/GoogleContainerTools/distroless)



