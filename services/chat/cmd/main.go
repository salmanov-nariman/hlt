package main

import (
	"chat/internal/handler"
	"chat/internal/service"
	"chat/internal/websocket"
	"log"
	"net/http"
)

func main() {
	wsManager := websocket.NewManager()

	chatService := service.NewChatService(wsManager)

	chatHandler := handler.NewChatHandler(chatService)

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/api/chat/ws", wsManager.ServeWS)
	http.HandleFunc("/api/chat/messages", chatHandler.PostMessage)

	log.Println("Server is starting on :8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
