package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	RedisAddr    string
	HTTPPort     string
	WorkerAddr   string
	DatabaseURL  string
	KafkaBrokers []string
	KafkaWSTopic string
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:     getEnv("HTTP_PORT", "8080"),
		WorkerAddr:   getEnv("WORKER_SERVICE_ADDR", ""),
		DatabaseURL:  getEnv("DATABASE_URL", ""),
		RedisAddr:    getEnv("REDIS_ADDR", ""),
		KafkaBrokers: parseBrokers(getEnv("KAFKA_BROKERS", "")),
		KafkaWSTopic: getEnv("KAFKA_WS_TOPIC", "websocket-events"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

func (c *Config) Validate() error {
	if err := validatePort(c.HTTPPort); err != nil {
		return fmt.Errorf("HTTP_PORT: %w", err)
	}

	if err := validateDatabaseURL(c.DatabaseURL); err != nil {
		return fmt.Errorf("DATABASE_URL: %w", err)
	}

	if err := validateAddress(c.RedisAddr); err != nil {
		return fmt.Errorf("REDIS_ADDR: %w", err)
	}

	if err := validateBrokers(c.KafkaBrokers); err != nil {
		return fmt.Errorf("KAFKA_BROKERS: %w", err)
	}

	if err := validateTopic(c.KafkaWSTopic); err != nil {
		return fmt.Errorf("KAFKA_WS_TOPIC: %w", err)
	}

	return nil
}

func validatePort(portStr string) error {
	if strings.HasPrefix(portStr, "+") {
		return errors.New("номер порта не должен начинаться со знака '+'")
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("порт должен состоять только из чисел в диапазоне от 1 до 65535")
	}

	return nil
}

func validateAddress(addr string) error {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return errors.New("некорректно передан адрес")
	}

	if strings.TrimSpace(host) == "" || strings.ContainsAny(host, " \t\n\r") {
		return errors.New("хост не должен быть пустым или содержать пробельные символы")
	}

	return validatePort(port)
}

var topicPattern = regexp.MustCompile("^[a-zA-Z0-9._-]+$")

func validateTopic(topic string) error {
	if topic == "." || topic == ".." {
		return errors.New("название топика не может быть равно '.' или '..'")
	}
	if len(topic) > 249 {
		return errors.New("название топика не должно превышать 249 символов")
	}

	if !topicPattern.MatchString(topic) {
		return errors.New("название топика содержит недопустимые символы или является пустым")
	}

	return nil
}

func parseBrokers(raw string) []string {
	brokers := strings.Split(raw, ",")

	for i, broker := range brokers {
		brokers[i] = strings.TrimSpace(broker)
	}

	return brokers
}

func validateBrokers(brokers []string) error {
	if len(brokers) == 0 {
		return errors.New("необходимо указать хотя бы одного брокера")
	}

	for i, broker := range brokers {
		if err := validateAddress(broker); err != nil {
			return fmt.Errorf("брокер №%d: %w", i+1, err)
		}
	}

	return nil
}

func validateDatabaseURL(rawURL string) error {
	if strings.TrimSpace(rawURL) == "" {
		return errors.New("url не должен быть пустым или состоять только из пробелов")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return errors.New("содержит некорректный URL")
	}

	if parsedURL.Scheme != "postgresql" && parsedURL.Scheme != "postgres" {
		return errors.New("строка должна начинаться с postgres или postgresql")
	}

	if parsedURL.User == nil || strings.TrimSpace(parsedURL.User.Username()) == "" {
		return errors.New("отсутствует имя пользователя БД")
	}

	if parsedURL.Hostname() == "" {
		return errors.New("отсутствует название хоста")
	}

	if strings.TrimSpace(strings.TrimPrefix(parsedURL.Path, "/")) == "" {
		return errors.New("отсутствует путь, содержащий название БД")
	}

	if _, err := pgxpool.ParseConfig(rawURL); err != nil {
		return errors.New("url содержит некорректную строку подключения PostgreSQL")
	}

	return nil
}
