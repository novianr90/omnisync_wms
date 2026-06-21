# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy go.mod and go.sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o wms_dashboard ./cmd/main.go

# Stage 2: Create a minimal runtime image
FROM alpine:latest

WORKDIR /app

# Install CA certificates
RUN apk --no-cache add ca-certificates tzdata

# Copy the binary from the builder stage
COPY --from=builder /app/wms_dashboard .

# Copy static assets and templates required for runtime
COPY --from=builder /app/web/static ./web/static
COPY --from=builder /app/web/templates ./web/templates
COPY --from=builder /app/migrations ./migrations

# Set default environment variables
ENV DB_TYPE=postgres
ENV APP_ENV=production

# Expose port
EXPOSE 9901

# Run the binary
CMD ["./wms_dashboard"]
