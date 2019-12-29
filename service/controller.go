package service

import (
	"context"

	pb "github.com/meateam/permit-service/proto"
)

// Controller is an interface for the business logic of the permit.Service which uses a Store.
type Controller interface {
	CreatePermit(ctx context.Context, reqID string, fileID string, userID string, status string) (Permit, error)
	GetPermitsByFileID(ctx context.Context, fileID string) ([]*pb.UserStatus, error)
	HasPermit(ctx context.Context, fileID string, userID string) (bool, error)
	UpdatePermitStatus(ctx context.Context, reqID string, status string) (bool, error)
	HealthCheck(ctx context.Context) (bool, error)
}
