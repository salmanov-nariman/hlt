package config

import "os"

type Config struct {
	HTTPPort   string
	WorkerAddr string
}

func Load() *Config {
	return &Config{
		HTTPPort:   getEnv("HTTP_PORT", "8080"),
		WorkerAddr: getEnv("WORKER_SERVICE_ADDR", "worker-service:50051"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
