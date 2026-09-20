package main

import (
	"log"
	"net"
	"net/http"

	"worker/internal/grpc_server"
	pb "worker/internal/pb/worker"

	"google.golang.org/grpc"
)

func healthCheck(writer http.ResponseWriter, request *http.Request) {
	log.Println("Пришел запрос на healthCheck")
	writer.WriteHeader(http.StatusOK)
}

func main() {
	log.Println("Сервис worker запускается...")

	go func() {
		grpcServer := grpc.NewServer()
	    pb.RegisterWorkerServiceServer(grpcServer, &grpc_server.WorkerServer{})

		lis, err := net.Listen("tcp", ":50051")
		if err != nil {
			log.Fatalf("Ошибка при прослушивании порта gRPC: %v", err)
		}

		log.Println("gRPC сервер слушает порт :50051")
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("Ошибка при запуске gRPC сервера: %v", err)
		}
	}()

	http.HandleFunc("/api/event/healthCheck", healthCheck)

	log.Println("HTTP сервер (healthCheck) слушает порт :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Ошибка при запуске HTTP сервера: %v", err)
	}
}
