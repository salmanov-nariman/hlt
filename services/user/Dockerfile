FROM golang:1.27-alpine AS builder
WORKDIR /build

COPY services/user/ ./services/user/

WORKDIR /build/services/user
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o user-service ./cmd/main.go

FROM alpine:latest
WORKDIR /app

LABEL org.opencontainers.image.source="https://github.com/salmanov-nariman/hlt"

COPY --from=builder /build/services/user/user-service .

EXPOSE 8080
CMD ["./user-service"]
