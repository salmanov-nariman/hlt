FROM golang:1.27-alpine AS builder
WORKDIR /build

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o worker-service ./cmd/main.go

FROM alpine:latest
WORKDIR /app

LABEL org.opencontainers.image.source="https://github.com/salmanov-nariman/hlt"

COPY --from=builder /build/worker-service .

EXPOSE 8080
CMD ["./worker-service"]
