package main

import (
	"chat/config"
	"chat/internal/websocket"
	"chat/migrations"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	cfg := config.Load()

	if err := runMigrations(cfg.DatabaseURL); err != nil {
		log.Fatalf("Ошибка при выполнении миграций: %v", err)
	}

	dbPool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Не удалось подключиться к БД: %v", err)
	}
	defer dbPool.Close()

	if err := dbPool.Ping(ctx); err != nil {
		log.Fatalf("База данных не отвечает: %v", err)
	}
	log.Println("Успешное подключение к PostgreSQL!")

	wsManager := websocket.NewManager()

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/api/chat/ws", wsManager.ServeWS)

	log.Println("Server is starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func runMigrations(dbUrl string) error {
	d, err := iofs.New(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("ошибка загрузки файлов миграций: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dbUrl)
	if err != nil {
		return fmt.Errorf("ошибка инициализации мигратора: %w", err)
	}
	defer func() {
		sourceErr, dbErr := m.Close()
		if sourceErr != nil {
			log.Printf("Ошибка закрытия источника миграций: %v", sourceErr)
		}
		if dbErr != nil {
			log.Printf("Ошибка закрытия соединения с БД в миграторе: %v", dbErr)
		}
	}()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("не удалось применить миграции: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("Миграции не требуются, база в актуальном состоянии.")
	} else {
		log.Println("Миграции успешно применены!")
	}

	return nil
}
