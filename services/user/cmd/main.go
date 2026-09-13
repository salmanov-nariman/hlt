package main

import (
	"log"
	"net/http"
)

func healthCheck(writer http.ResponseWriter, request *http.Request) {
	log.Println("Пришел запрос на healthCheck")
	writer.WriteHeader(http.StatusOK)
}

func main() {
	log.Println("Сервис user запущен...")

	http.HandleFunc("/api/user/healthCheck", healthCheck)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Println("Ошибка при запуске user сервиса", err)
	}
}
