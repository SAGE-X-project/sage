# SAGE Multi-stage Dockerfile
# Optimized for production with minimal image size

# Stage 1: Builder
FROM golang:1.24.8-alpine AS builder

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

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build all binaries
RUN make build

# Build library (optional, for multi-language support)
RUN make build-lib || true

# Stage 2: Runtime
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1000 sage && \
    adduser -D -u 1000 -G sage sage

# Set working directory
WORKDIR /home/sage

# Copy binaries from builder
COPY --from=builder /app/build/bin/* /usr/local/bin/

# Copy libraries if they exist
RUN --mount=type=bind,from=builder,source=/app/build/lib,target=/tmp/lib \
    if [ -d /tmp/lib ] && [ "$(ls -A /tmp/lib)" ]; then \
        cp -r /tmp/lib/* /usr/local/lib/; \
    fi

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
