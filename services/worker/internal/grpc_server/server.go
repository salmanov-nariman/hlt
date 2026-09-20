package grpc_server

import (
	"log"
	pb "worker/internal/pb/worker"
)

type WorkerServer struct {
	pb.UnimplementedWorkerServiceServer
}

func (w *WorkerServer) GetDuoMembers(
	req *pb.GetDuoMembersRequest,
) (*pb.GetDuoMembersResponse, error) {
	duoID := req.GetDuoId()
	senderID := req.GetSenderId()

	log.Printf("Получен gRPC запрос. DuoID: %s, SenderID: %s", duoID, senderID)

	return &pb.GetDuoMembersResponse{
		UserIds: []string{req.SenderId, "a48e7c82-4b26-4d90-bda7-8b85f3f25abf"},
	}, nil
}
