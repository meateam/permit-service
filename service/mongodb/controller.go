package mongodb

import (
	"context"

	pb "github.com/meateam/permit-service/proto"
	"github.com/meateam/permit-service/service"
	"go.mongodb.org/mongo-driver/mongo"
)

// Controller is the permisison service business logic implementation using MongoStore.
type Controller struct {
	store MongoStore
}

// NewMongoController returns a new controller.
func NewMongoController(db *mongo.Database) (Controller, error) {
	store, err := newMongoStore(db)
	if err != nil {
		return Controller{}, err
	}

	return Controller{store: store}, nil
}

// HealthCheck runs store's healthcheck and returns true if healthy, otherwise returns false
// and any error if occured.
func (c Controller) HealthCheck(ctx context.Context) (bool, error) {
	return true, nil
}

// CreatePermit creates a permit in store and returns its unique ID.
func (c Controller) CreatePermit(ctx context.Context, reqID string, fileID string, userID string, status pb.Status) (service.Permit, error) {
	return nil, nil
}

// GetPermitByFileID returns the statuses of the permits of each user associated with the fileID.
func (c Controller) GetPermitByFileID(ctx context.Context, fileID string) ([]pb.UserStatus, error) {
	return nil, nil
}

// UpdatePermitStatus todo
func (c Controller) UpdatePermitStatus(ctx context.Context, req *pb.UpdatePermitStatusRequest) (*pb.UpdatePermitStatusResponse, error) {
	return nil, nil
}
