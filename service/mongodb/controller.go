package mongodb

import (
	"context"
	"fmt"

	pb "github.com/meateam/permit-service/proto"
	"github.com/meateam/permit-service/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	fmt.Println("Todo: complete healthcheck")
	return true, nil
}

// CreatePermit creates a permit in store and returns its unique ID.
func (c Controller) CreatePermit(ctx context.Context, reqID string, fileID string, userID string, status pb.Status) (service.Permit, error) {
	fmt.Println("Todo: complete CreatePermit")
	permit := &BSON{FileID: fileID, ReqID: reqID, UserID: userID, Status: status}
	createdPermission, err := c.store.Create(ctx, permit)
	if err != nil {
		return nil, fmt.Errorf("failed creating permit %v", err)
	}
	return createdPermission, nil
}

// GetPermitsByFileID returns the statuses of the permits of each user associated with the fileID.
func (c Controller) GetPermitsByFileID(ctx context.Context, fileID string) ([]*pb.UserStatus, error) {
	fmt.Println("Todo: complete GetPermitsByFileID")
	filter := bson.D{
		bson.E{
			Key:   PermitBSONFileIDField,
			Value: fileID,
		},
	}

	permits, err := c.store.GetAll(ctx, filter)
	if err != nil && err != mongo.ErrNoDocuments {
		return nil, err
	}

	if err == mongo.ErrNoDocuments {
		return nil, status.Error(codes.NotFound, "permit not found")
	}

	userStatuses := make([]*pb.UserStatus, 0, len(permits))

	for _, permit := range permits {
		userStatus := &pb.UserStatus{UserId: permit.GetUserID(), Status: permit.GetStatus().String()}
		userStatuses = append(userStatuses, userStatus)
	}

	return userStatuses, nil
}

// UpdatePermitStatus todo
func (c Controller) UpdatePermitStatus(ctx context.Context, reqID string, status pb.Status) (bool, error) {
	fmt.Println("Todo: complete UpdatePermitStatus")

	return true, nil
}
