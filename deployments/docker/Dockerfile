# SAGE Multi-stage Dockerfile
# Optimized for production with minimal image size

# Stage 1: Builder
# Base images are pinned by digest (multi-arch index); Dependabot keeps the tag and digest in step.
FROM golang:1.26.9-alpine@sha256:cdfd4fe2da6b225d8b40c6b7a105736e548e83ff56d5d8f9394446eeb5eb84e0 AS builder

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

# Stage 2: Runtime
FROM alpine:3.24.2@sha256:294b683cb724975bec92580e1e685676bd4b50bda910ddb8c51d4cabeaec77e6

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
