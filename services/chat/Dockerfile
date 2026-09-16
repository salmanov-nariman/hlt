FROM golang:1.27-alpine AS builder
WORKDIR /build

COPY services/booking/ ./services/booking/

WORKDIR /build/services/booking
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o booking-service ./cmd/main.go

FROM alpine:latest
WORKDIR /app

LABEL org.opencontainers.image.source="https://github.com/salmanov-nariman/hlt"

COPY --from=builder /build/services/booking/booking-service .

EXPOSE 8080
CMD ["./booking-service"]
