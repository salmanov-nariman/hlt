package config

import (
	"fmt"
	"os"
)

type Config struct {
	DatabaseURL string
	HTTPPort    string
	WorkerAddr  string
}

func Load() *Config {
	dbUser := getEnv("POSTGRES_USER", "chat_user")
	dbPass := getEnv("POSTGRES_PASSWORD", "chat_password")
	dbHost := getEnv("DB_HOST", "chat-postgres")
	dbPort := getEnv("DB_PORT", "5432")
	dbName := getEnv("POSTGRES_DB", "chat_db")

	dsn := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPass, dbHost, dbPort, dbName)

	return &Config{
		DatabaseURL: dsn,
		HTTPPort:    getEnv("HTTP_PORT", "8080"),
		WorkerAddr:  getEnv("WORKER_SERVICE_ADDR", "worker-service:50051"),
	}
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
