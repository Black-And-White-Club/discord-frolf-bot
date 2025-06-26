# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies and security updates
RUN apk add --no-cache --update git ca-certificates tzdata \
    && apk upgrade \
    && rm -rf /var/cache/apk/*

WORKDIR /app

# Copy go mod and sum files first for better caching
COPY go.mod go.sum ./
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build the application with optimizations and reproducible builds
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags='-w -s -extldflags "-static" -buildid=' \
    -trimpath \
    -a -installsuffix cgo \
    -o discord-frolf-bot .

# Runtime stage - use distroless for minimal attack surface
FROM gcr.io/distroless/static:nonroot

# Add metadata labels for better container management
LABEL org.opencontainers.image.title="Discord Frolf Bot" \
      org.opencontainers.image.description="Discord bot for Disc Golf event management" \
      org.opencontainers.image.source="https://github.com/Black-And-White-Club/discord-frolf-bot" \
      org.opencontainers.image.vendor="Black And White Club" \
      org.opencontainers.image.licenses="MIT"

# Copy timezone data and ca-certificates from builder
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy the binary from builder stage
COPY --from=builder /app/discord-frolf-bot /discord-frolf-bot

# Copy migrations for database setup (only if migrations directory exists)
COPY --from=builder /app/migrations /migrations

# Don't copy config.yaml - use environment variables or volume mounts instead
# COPY --from=builder /app/config.yaml /config.yaml

# Use nonroot user from distroless (UID 65532)
USER nonroot:nonroot

# Expose port for health checks and webhooks
EXPOSE 8080

# Use exec form for better signal handling
ENTRYPOINT ["/discord-frolf-bot"]
