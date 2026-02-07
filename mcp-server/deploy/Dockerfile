# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git for fetching dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the server
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o server ./cmd/server

# Final stage - distroless
FROM gcr.io/distroless/static:nonroot

# Non-root user
USER 65532:65532

# Copy binary
COPY --from=builder --chown=65532:65532 /app/server /server

# Expose ports
EXPOSE 8080 8081

# Health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/server"] || exit 1

ENTRYPOINT ["/server"]
