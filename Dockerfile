# =============================================================================
# Guest Lock PIN Manager - Multi-stage Dockerfile
# Produces a minimal distroless container for Home Assistant addon deployment
# =============================================================================

# -----------------------------------------------------------------------------
# Stage 1: Frontend Build
# -----------------------------------------------------------------------------
FROM node:20-alpine AS frontend-builder

WORKDIR /app/frontend

# Install dependencies first (better layer caching)
COPY frontend/package*.json ./
RUN npm ci --ignore-scripts

# Copy source and build
COPY frontend/ ./
RUN npm run build

# -----------------------------------------------------------------------------
# Stage 2: Backend Build (glibc-based for distroless runtime)
# -----------------------------------------------------------------------------
FROM golang:1.22-bookworm AS backend-builder

# Install build dependencies for CGO (required by go-sqlite3)
RUN apt-get update \
 && apt-get install -y --no-install-recommends build-essential ca-certificates \
 && rm -rf /var/lib/apt/lists/*

WORKDIR /app/backend

# Download dependencies first (better layer caching)
COPY backend/go.mod backend/go.sum* ./
RUN go mod download

# Copy source and build
COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /server ./cmd/server

# -----------------------------------------------------------------------------
# Stage 3: Production Runtime (distroless, glibc)
# -----------------------------------------------------------------------------
FROM gcr.io/distroless/base-debian12:nonroot

# Labels for Home Assistant addon
LABEL \
    io.hass.name="Guest Lock PIN Manager" \
    io.hass.description="Automate door lock PINs for short-term rental guests" \
    io.hass.type="addon" \
    io.hass.version="0.1.0"

WORKDIR /app

# Copy compiled backend binary
COPY --from=backend-builder /server /app/server

# Copy built frontend assets
COPY --from=frontend-builder /app/frontend/dist /app/static

# Create data directory for SQLite database
# Note: In distroless, we rely on volume mounts for /data

# Expose the ingress port
EXPOSE 8099

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/app/server", "-health-check"]

# Run as root so we can write to /data volume provided by Home Assistant
USER root

# Start the server
ENTRYPOINT ["/app/server"]
CMD ["-addr", ":8099", "-data", "/data", "-static", "/app/static"]

