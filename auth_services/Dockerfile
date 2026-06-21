# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o auth_services ./cmd/main.go

# Stage 2: Create a minimal runtime image
FROM alpine:latest

WORKDIR /app

# Install CA certificates for HTTPS requests (e.g., to Supabase or other external APIs)
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from the builder stage
COPY --from=builder /app/auth_services .
COPY --from=builder /app/migrations ./migrations

# Set default environment variables
ENV DB_TYPE=postgres
ENV APP_ENV=production

# Expose port
EXPOSE 8000

# Run the binary
CMD ["./auth_services"]
