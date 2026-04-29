# ============================================================
# Stage 1: builder — compile the Go binary
# ============================================================
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

# Copy module files first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source
COPY . .

# Build a static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /app/server ./cmd/api

# ============================================================
# Stage 2: runtime — minimal image with just the binary
# ============================================================
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

# Copy only the compiled binary from builder
COPY --from=builder /app/server /app/server

# Run as non-root for security
RUN adduser -D -u 1000 appuser
USER appuser

EXPOSE 3000

ENTRYPOINT ["/app/server"]