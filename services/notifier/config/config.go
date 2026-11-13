package config

import "os"

type Config struct {
	RedisAddr    string
	KafkaBrokers string
	KafkaTopic   string
}

func Load() *Config {
	return &Config{
		RedisAddr:    getEnv("REDIS_ADDR", "localhost:6379"),
		KafkaBrokers: getEnv("KAFKA_BROKERS", "localhost:9092"),
		KafkaTopic:   getEnv("KAFKA_WS_TOPIC", "chat-ws-events"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
