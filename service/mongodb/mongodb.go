package mongodb

import (
	"fmt"

	pb "github.com/meateam/permit-service/proto"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// BSON is the struct that represents a permit as it's stored.
type BSON struct {
	ID     primitive.ObjectID `bson:"_id,omitempty"`
	FileID string             `bson:"fileID, omitempty"`
	Status string             `bson:"status,omitempty"`
	UserID string             `bson:"userID,omitempty"`
	ReqID  string             `bson:"reqID,omitempty"`
}

// GetID returns the string value of the b.ID.
func (b BSON) GetID() string {
	if b.ID.IsZero() {
		return ""
	}

	return b.ID.Hex()
}

// SetID sets the b.ID ObjectID's string value to id.
func (b *BSON) SetID(id string) error {
	if b == nil {
		panic("b == nil")
	}

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	b.ID = objectID
	return nil
}

// GetFileID returns b.FileID.
func (b BSON) GetFileID() string {
	return b.FileID
}

// SetFileID sets b.FileID to fileID.
func (b *BSON) SetFileID(fileID string) error {
	if b == nil {
		panic("b == nil")
	}

	if fileID == "" {
		return fmt.Errorf("FileID is required")
	}

	b.FileID = fileID
	return nil
}

// GetStatus returns b.Status.
func (b BSON) GetStatus() string {
	return b.Status
}

// SetStatus sets b.SharerID to status.
func (b *BSON) SetStatus(status string) error {
	if b == nil {
		panic("b == nil")
	}

	b.Status = status
	return nil
}

// GetUserID returns b.UserID.
func (b BSON) GetUserID() string {
	return b.UserID
}

// SetUserID sets b.UserID to userID.
func (b *BSON) SetUserID(userID string) error {
	if b == nil {
		panic("b == nil")
	}

	if userID == "" {
		return fmt.Errorf("UserID is required")
	}

	b.UserID = userID
	return nil
}

// GetReqID returns b.reqID.
func (b BSON) GetReqID() string {
	return b.ReqID
}

// SetReqID sets b.ReqID to reqID.
func (b *BSON) SetReqID(reqID string) error {
	if b == nil {
		panic("b == nil")
	}

	if reqID == "" {
		return fmt.Errorf("reqID is required")
	}

	b.ReqID = reqID
	return nil
}

// MarshalProto marshals b into a permission.
func (b BSON) MarshalProto(permit *pb.PermitObject) error {
	permit.ReqID = b.GetReqID()
	permit.FileID = b.GetFileID()
	permit.UserID = b.GetUserID()
	permit.Status = b.GetStatus()

	return nil
}
