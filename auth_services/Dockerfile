FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o auth_services ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/auth_services .
COPY migrations/ ./migrations/
EXPOSE 8000
ENV DB_TYPE=postgres
CMD ["./auth_services"]
