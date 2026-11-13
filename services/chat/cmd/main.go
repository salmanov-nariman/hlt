package main

import (
	"chat/config"
	_ "chat/docs"
	"chat/internal/broker"
	"chat/internal/handler"
	"chat/internal/repository"
	"chat/internal/service"
	"chat/internal/websocket"
	"chat/middleware"
	"chat/migrations"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	httpSwagger "github.com/swaggo/http-swagger"
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

	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr, 
	})

	wsManager := websocket.NewManager(rdb)
	chatRepository := repository.NewChatRepository(dbPool)
	
	kafkaBrokers := strings.Split(cfg.KafkaBrokers, ",")
	wsTopic := cfg.KafkaWSTopic

	kafkaProducer := broker.NewProducer(kafkaBrokers, wsTopic)
	defer kafkaProducer.Close()

	kafkaConsumer := broker.NewConsumer(kafkaBrokers, wsTopic, wsManager)
	defer kafkaConsumer.Close()

	go kafkaConsumer.Start(ctx)

	chatService := service.NewChatService(kafkaProducer, chatRepository)
	chatHandler := handler.NewChatHandler(chatService, kafkaProducer)

	r := chi.NewRouter()

	r.Use(chiMiddleware.Recoverer)

	r.Route("/api", func(r chi.Router) {
		r.Use(middleware.AuthMiddleware)

		r.Post("/chats", middleware.REDMetricsMiddleware("/api/chats", chatHandler.CreateChat))
		r.Get("/chats", middleware.REDMetricsMiddleware("/api/chats", chatHandler.GetUserChats))
		r.Get("/chats/{chat_id}", middleware.REDMetricsMiddleware("/api/chat/{chat_id}", chatHandler.GetChat))

		r.Post("/chats/{chat_id}/messages", middleware.REDMetricsMiddleware("/api/messages", chatHandler.PostMessage))
		r.Get("/chats/{chat_id}/messages", middleware.REDMetricsMiddleware("/api/chat/messages", chatHandler.GetChatMessages))

		r.Post("/chats/{chat_id}/read", middleware.REDMetricsMiddleware("/api/chats/{chat_id}/read", chatHandler.MarkAsRead))

		r.Get("/sync/messages",  middleware.REDMetricsMiddleware("/api/sync/messages", chatHandler.SyncMessages))
		
		r.Get("/chat/ws", wsManager.ServeWS)
	})

	r.Handle("/metrics", promhttp.Handler())

	r.Get("/swagger/*", httpSwagger.WrapHandler)
	r.Handle("/*", http.FileServer(http.Dir("./frontend")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server is starting on :%s...", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
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
