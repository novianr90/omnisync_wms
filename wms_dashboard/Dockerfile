FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o wms_dashboard ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/wms_dashboard .
COPY migrations/ ./migrations/
COPY web/ ./web/
EXPOSE 9901
ENV DB_TYPE=postgres
CMD ["./wms_dashboard"]
