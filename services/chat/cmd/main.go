package main

import (
	"chat/internal/config"
	"chat/internal/db"
	"chat/internal/grpc_client/worker"
	"chat/internal/handler"
	"chat/internal/repository"
	"chat/internal/service"
	"chat/internal/websocket"
	"log"
	"net/http"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	cfg := config.Load()

	postgresDB, err := db.NewPostgresDB(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to initialize db: %v", err)
	}
	defer postgresDB.Close()
	log.Println("Successfully connected to PostgreSQL")

	chatRepo := repository.NewChatRepository(postgresDB)

	log.Printf("Connecting to Worker Service at %s...", cfg.WorkerAddr)
	grpcConn, err := grpc.NewClient(cfg.WorkerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to worker service: %v", err)
	}
	defer grpcConn.Close()
	workerClient := worker.NewGRPCWorkerClient(grpcConn)

	wsManager := websocket.NewManager()

	chatService := service.NewChatService(wsManager, workerClient, chatRepo)

	chatHandler := handler.NewChatHandler(chatService)

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/api/chat/ws", wsManager.ServeWS)
	http.HandleFunc("/api/chat/messages", chatHandler.PostMessage)

	log.Printf("Server is starting on :%s...", cfg.HTTPPort)
	log.Fatal(http.ListenAndServe(":"+cfg.HTTPPort, nil))
}
