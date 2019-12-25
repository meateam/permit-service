package service

import (
	pb "github.com/meateam/permit-service/proto"
)

// Permit is an interface of a permit object.
type Permit interface {
	GetID() string
	SetID(id string) error

	GetReqID() string
	SetReqID(reqID string) error

	GetFileID() string
	SetFileID(fileID string) error

	GetUserID() string
	SetUserID(userID string) error

	GetStatus() string
	SetStatus(status string) error

	MarshalProto(permit *pb.PermitObject) error
}
