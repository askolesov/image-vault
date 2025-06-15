# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN make build

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache \
    exiftool \
    ca-certificates \
    tzdata

# Create non-root user
RUN addgroup -g 1001 -S imv && \
    adduser -u 1001 -S imv -G imv

# Copy binary from builder stage
COPY --from=builder /app/build/imv /usr/local/bin/imv

# Set ownership and permissions
RUN chown imv:imv /usr/local/bin/imv && \
    chmod +x /usr/local/bin/imv

# Switch to non-root user
USER imv

# Set working directory for the application
WORKDIR /workspace

# Default command
ENTRYPOINT ["imv"]
CMD ["--help"]
