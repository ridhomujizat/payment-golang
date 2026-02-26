FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install git and ca-certificates
RUN apk add --no-cache git ca-certificates

# Copy go files first
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy all source code
COPY . .

# Build the application
RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux go build -v -o api ./cmd/api

# Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates wget
WORKDIR /root/

# Copy binary and required files
COPY --from=builder /app/api .
COPY --from=builder /app/configs ./configs

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD wget --quiet --tries=1 --spider http://localhost:8080/health || exit 1

CMD ["./api"]
