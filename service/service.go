package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	pb "github.com/meateam/permit-service/proto"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
)

// Service is the structure used for handling
type Service struct {
	controller Controller
	logger     *logrus.Logger
}

// ApprovalReqType is the struct sent as json to the approval service
type ApprovalReqType struct {
	reqID          string
	fileID         string
	sharerID       string
	users          []string
	classification string
	info           string
}

// HealthCheck checks the health of the service, and returns a boolean accordingly.
func (s *Service) HealthCheck(mongoClientPingTimeout time.Duration) bool {
	timeoutCtx, cancel := context.WithTimeout(context.TODO(), mongoClientPingTimeout)
	defer cancel()
	healthy, err := s.controller.HealthCheck(timeoutCtx)
	if err != nil {
		s.logger.Errorf("%v", err)
		return false
	}

	return healthy
}

// NewService creates a Service and returns it.
func NewService(controller Controller, logger *logrus.Logger) Service {
	return Service{controller: controller, logger: logger}
}

// CreatePermit is the request handler for creating a permit of a file to user.
func (s Service) CreatePermit(ctx context.Context, req *pb.CreatePermitRequest) (*pb.CreatePermitResponse, error) {
	fileID := req.GetFileID()
	sharerID := req.GetSharerID()
	users := req.GetUsers()
	classification := req.GetClassification()
	info := req.GetInfo()

	usersNum := len(users)

	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}

	if sharerID == "" {
		return nil, fmt.Errorf("sharerID is required")
	}

	if usersNum == 0 {
		return nil, fmt.Errorf("at least one user is required")
	}

	reqID, err := ksuid.NewRandomWithTime(time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed creating reqID")
	}

	// Add the permits to mongo
	var wg sync.WaitGroup
	wg.Add(usersNum)

	var userIDs []string
	for i := 0; i < usersNum; i++ {
		userIDs = append(userIDs, users[i].GetId())
	}

	for i := 0; i < usersNum; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := s.controller.CreatePermit(ctx, reqID.String(), fileID, userIDs[i], pb.Status_NONE)
			if err != nil {
				_ = fmt.Errorf("failed creating permit %s %s %v", fileID, users[i].GetId(), err)
			}
		}(i)
	}

	// TODO: get spike token. add header of authorization bearer

	// Call Approval service with the required parameters.
	requestBody, err := json.Marshal(
		&ApprovalReqType{
			reqID:          reqID.String(),
			fileID:         fileID,
			sharerID:       sharerID,
			users:          userIDs,
			classification: classification,
			info:           info,
		})

	if err != nil {
		return nil, fmt.Errorf("failed creating json object, %v", err)
	}

	resp, err := http.Post("https://todo.com/bliblu", "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error while requesting from approval service %v", err)
	}

	defer resp.Body.Close()

	return &pb.CreatePermitResponse{}, nil
}

// GetPermitByFileID is the request handler for getting a permit (user, status) by file id.
func (s Service) GetPermitByFileID(ctx context.Context, req *pb.GetPermitByFileIDRequest) (*pb.GetPermitByFileIDResponse, error) {
	fileID := req.GetFileID()
	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}

	userStatuses, err := s.controller.GetPermitByFileID(ctx, fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve permit %v", err)
	}

	return &pb.GetPermitByFileIDResponse{UserStatus: userStatuses}, nil
}

// UpdatePermitStatus is the request handler for updating the status of a given permit.
func (s Service) UpdatePermitStatus(ctx context.Context, req *pb.UpdatePermitStatusRequest) (*pb.UpdatePermitStatusResponse, error) {
	reqID := req.GetReqID()
	status := req.GetStatus()

	if reqID == "" {
		return nil, fmt.Errorf("reqID is required")
	}

	ok, err := s.controller.UpdatePermitStatus(ctx, reqID, status)
	if err != nil {
		return nil, fmt.Errorf("update permit status failed %v", err)
	}

	if !ok {
		_ = fmt.Errorf("error updating permit status")
	}

	return &pb.UpdatePermitStatusResponse{}, nil
}
