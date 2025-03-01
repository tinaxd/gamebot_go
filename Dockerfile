FROM golang:alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN go build -o gamebot ./cmd/gamebot

FROM alpine:latest

WORKDIR /app

# Create data directory for SQLite
RUN mkdir -p /app/data

# Copy the binary from builder
COPY --from=builder /app/gamebot /app/

# Set execution permissions
RUN chmod +x /app/gamebot

CMD ["/app/gamebot"]
