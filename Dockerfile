# Stage 1: Build the Go binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy dependency files and download
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the application
# We use -ldflags="-s -w" to shrink the binary and CGO_ENABLED=0 for a static binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/server ./cmd/server/main.go

# Stage 2: Final lightweight image
FROM alpine:latest

# Install ca-certificates to allow HTTPS requests to external sites
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the binary from the builder stage
COPY --from=builder /app/server .

# Expose the default port
EXPOSE 8080

# Run the binary
CMD ["./server"]
