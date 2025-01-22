# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Set working directory
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /go/bin/quantai cmd/server/main.go

# Final stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata curl

# Set working directory
WORKDIR /app

# Create non-root user
RUN adduser -D -g '' quantai
USER quantai

# Copy binary from builder
COPY --from=builder /go/bin/quantai .

# Copy necessary files
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/config ./config

# Environment variables
ENV PORT=8080 \
    GIN_MODE=release \
    TZ=UTC

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Set entry point
ENTRYPOINT ["./quantai"]