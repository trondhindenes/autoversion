# Build stage
FROM golang:1.25-alpine AS builder

# Install git (needed for go-git)
RUN apk add --no-cache git

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o autoversion ./cmd/autoversion

# Final stage
FROM alpine:latest AS autoversion

# Install git (required for autoversion to work with repositories)
RUN apk add --no-cache git

# Copy the binary from builder
COPY --from=builder /build/autoversion /usr/local/bin/autoversion

# Set working directory
WORKDIR /repo

# Default command
ENTRYPOINT ["/usr/local/bin/autoversion"]

FROM autoversion AS autoversion-action

RUN apk add --no-cache bash
RUN git config --global --add safe.directory '*'

# Copy entrypoint script
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Set working directory
WORKDIR /github/workspace

# Set entrypoint
ENTRYPOINT ["/entrypoint.sh"]