# SAGE Multi-stage Dockerfile
# Optimized for production with minimal image size

# Stage 1: Builder
# Base images are pinned by digest (multi-arch index); Dependabot keeps the tag and digest in step.
FROM golang:1.27.1-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download and verify dependencies against go.sum; never modify go.mod during the build
ENV GOFLAGS=-mod=readonly
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build all binaries
RUN make build

# Stage 2: Runtime
FROM alpine:3.24.1@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# Upgrade the base packages first so the image does not ship OS-level CVEs
# already fixed in the Alpine repositories (e.g. OpenSSL), then install
# runtime dependencies.
RUN apk upgrade --no-cache && \
    apk add --no-cache \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 sage && \
    adduser -D -u 1000 -G sage sage

# Set working directory
WORKDIR /home/sage

# Copy binaries from builder
COPY --from=builder /app/build/bin/* /usr/local/bin/

# Copy configuration templates if they exist
RUN --mount=type=bind,from=builder,source=/app,target=/tmp/app \
    if [ -f /tmp/app/config.yaml.example ]; then \
        cp /tmp/app/config.yaml.example /home/sage/config.yaml.example; \
    fi

# Set ownership
RUN chown -R sage:sage /home/sage

# Switch to non-root user
USER sage

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD sage-crypto help >/dev/null 2>&1 || exit 1

# Expose ports (adjust as needed)
EXPOSE 8080

# Default command
CMD ["sage-crypto", "help"]
