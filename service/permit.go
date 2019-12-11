package service

import (
	pb "github.com/meateam/permit-service/proto"
)

// Permit is an interface of a permit object.
type Permit interface {
	MarshalProto(permission *pb.PermitObject) error
}
