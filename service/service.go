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
	spb "github.com/meateam/spike-service/proto/spike-service"
	"github.com/segmentio/ksuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	// StatusPending is the status of a pending request
	StatusPending = "pending"
)

// Service is the structure used for handling
type Service struct {
	spikeClient spb.SpikeClient
	controller  Controller
	logger      *logrus.Logger
	grantType   string
	audience    string
	approvalURL string
}

// ApprovalReqType is the struct sent as json to the approval service
type ApprovalReqType struct {
	ID             string     `json:"id"`
	From           string     `json:"from"`
	Approvers      []string   `json:"approvers"`
	To             []UserType `json:"to"`
	FileID         string     `json:"fileId"`
	Info           string     `json:"info"`
	Classification string     `json:"classification"`
}

// UserType is the struct that contains id and fullname of a user
type UserType struct {
	ID   string `json:"id"`
	Name string `json:"name"`
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
func NewService(controller Controller, logger *logrus.Logger, spikeConn *grpc.ClientConn, grantType string, audience string, approvalURL string) Service {
	s := Service{controller: controller, logger: logger, grantType: grantType, audience: audience, approvalURL: approvalURL}
	s.spikeClient = spb.NewSpikeClient(spikeConn)
	return s
}

// CreatePermit is the request handler for creating a permit of a file to user.
func (s Service) CreatePermit(ctx context.Context, req *pb.CreatePermitRequest) (*pb.CreatePermitResponse, error) {
	fileID := req.GetFileID()
	sharerID := req.GetSharerID()
	users := req.GetUsers()
	classification := req.GetClassification()
	info := req.GetInfo()
	approvers := req.GetApprovers()

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

	var userIDs []UserType
	for i := 0; i < usersNum; i++ {
		user := UserType{
			ID:   users[i].GetId(),
			Name: users[i].GetFullName(),
		}
		userIDs = append(userIDs, user)
	}

	for i := 0; i < usersNum; i++ {
		go func(i int) {
			defer wg.Done()
			_, err := s.controller.CreatePermit(ctx, reqID.String(), fileID, userIDs[i].ID, StatusPending)
			if err != nil {
				_ = fmt.Errorf("failed creating permit %s %s %v", fileID, users[i].GetId(), err)
			}
		}(i)
	}

	// TODO: get spike token. add header of authorization bearer
	getSpikeTokenRequest := &spb.GetSpikeTokenRequest{
		GrantType: s.grantType,
		Audience:  s.audience,
	}

	tokenRes, err := s.spikeClient.GetSpikeToken(ctx, getSpikeTokenRequest)
	if err != nil {
		return nil, fmt.Errorf("failed getting spike token %v", err)
	}

	token := tokenRes.GetToken()

	request := ApprovalReqType{
		ID:             reqID.String(),
		From:           sharerID,
		Approvers:      approvers,
		To:             userIDs,
		FileID:         fileID,
		Info:           info,
		Classification: classification,
	}

	requestBody, err := json.Marshal(request)

	if err != nil {
		return nil, fmt.Errorf("failed creating json object, %v", err)
	}

	client := &http.Client{}
	httpReq, err := http.NewRequest("POST", s.approvalURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("error while creating http request to approval, %v", err)
	}
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(httpReq)
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

	userStatuses, err := s.controller.GetPermitsByFileID(ctx, fileID)
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
