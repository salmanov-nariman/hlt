FROM golang:1.27-alpine AS builder
WORKDIR /build

COPY services/auth/ ./services/auth/

WORKDIR /build/services/auth
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o auth-service ./cmd/main.go

FROM alpine:latest
WORKDIR /app

LABEL org.opencontainers.image.source="https://github.com/salmanov-nariman/hlt"

COPY --from=builder /build/services/auth/auth-service .

EXPOSE 8080
CMD ["./auth-service"]
