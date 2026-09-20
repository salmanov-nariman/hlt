package grpc_server

import (
	"context" // Добавлен импорт контекста
	"log"
	pb "worker/internal/pb/worker"
)

type WorkerServer struct {
	pb.UnimplementedWorkerServiceServer
}

// Добавлен ctx context.Context первым аргументом
func (w *WorkerServer) GetDuoMembers(
	ctx context.Context,
	req *pb.GetDuoMembersRequest,
) (*pb.GetDuoMembersResponse, error) {
	duoID := req.GetDuoId()
	senderID := req.GetSenderId()

	log.Printf("Получен gRPC запрос. DuoID: %s, SenderID: %s", duoID, senderID)

	return &pb.GetDuoMembersResponse{
		UserIds: []string{senderID, "a48e7c82-4b26-4d90-bda7-8b85f3f25abf"},
	}, nil
}
