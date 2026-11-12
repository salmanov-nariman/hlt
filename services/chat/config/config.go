package config

import "os"

type Config struct {
	HTTPPort     string
	WorkerAddr   string
	DatabaseURL  string
	KafkaBrokers string
	KafkaWSTopic string
}

func Load() *Config {
	return &Config{
		HTTPPort: getEnv("HTTP_PORT", "8080"),
		// WorkerAddr:  getEnv("WORKER_SERVICE_ADDR", "worker-service:50051"),
		WorkerAddr:  getEnv("WORKER_SERVICE_ADDR", "localhost:50051"),
		DatabaseURL: getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/chat_db?sslmode=disable"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaWSTopic: getEnv("KAFKA_WS_TOPIC", "chat-ws-events"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
