package worker

import (
	"context"
	"fmt"

	pb "chat/internal/pb/worker"

	"google.golang.org/grpc"
)

type WorkerClient interface {
	GetDuoMembers(ctx context.Context, duoID, senderID string) ([]string, error)
}

type grpcWorkerClient struct {
	api pb.WorkerServiceClient
}

func NewGRPCWorkerClient(conn grpc.ClientConnInterface) WorkerClient {
	return &grpcWorkerClient{
		api: pb.NewWorkerServiceClient(conn),
	}
}

func (c *grpcWorkerClient) GetDuoMembers(ctx context.Context, duoID, senderID string) ([]string, error) {
	req := &pb.GetDuoMembersRequest{
		DuoId:    duoID,
		SenderId: senderID,
	}

	resp, err := c.api.GetDuoMembers(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get duo members: %w", err)
	}

	return resp.UserIds, nil
}
