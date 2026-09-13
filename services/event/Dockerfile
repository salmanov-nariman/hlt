FROM golang:1.27-alpine AS builder
WORKDIR /build

COPY services/event/ ./services/event/

WORKDIR /build/services/event
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o event-service ./cmd/main.go

FROM alpine:latest
WORKDIR /app

LABEL org.opencontainers.image.source="https://github.com/salmanov-nariman/hlt"

COPY --from=builder /build/services/event/event-service .

EXPOSE 8080
CMD ["./event-service"]
