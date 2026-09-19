package main

import (
	"chat/internal/websocket"
	"log"
	"net/http"
)

func main() {
	setupAPI()

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func setupAPI() {
	manager := websocket.NewManager()

	http.Handle("/", http.FileServer(http.Dir("./frontend")))
	http.HandleFunc("/ws", manager.ServeWS)
}
