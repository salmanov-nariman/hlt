package main

import (
	"chat/internal/grpc_client/worker"
	"chat/internal/handler"
	"chat/internal/service"
	"chat/internal/websocket"
	"log"
	"net/http"
)

func main() {
	//cfg := config.Load()
	//
	//log.Printf("Connecting to Worker Service at %s...", cfg.WorkerAddr)
	/*
		КОГДА ВОРКЕР БУДЕТ ГОТОВ В KUBERNETES, РАСКОММЕНТИРУЙ ЭТОТ БЛОК:

		workerAddr := "worker-service:50051"
		grpcConn, err := grpc.NewClient(workerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Failed to connect to worker service: %v", err)
		}
		defer grpcConn.Close()
		workerClient := worker.NewGRPCWorkerClient(grpcConn)
	*/

	workerClient := worker.NewMockWorkerClient()

	wsManager := websocket.NewManager()
	chatService := service.NewChatService(wsManager, workerClient)
	chatHandler := handler.NewChatHandler(chatService)

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/api/chat/ws", wsManager.ServeWS)
	http.HandleFunc("/api/chat/messages", chatHandler.PostMessage)

	log.Println("Server is starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
