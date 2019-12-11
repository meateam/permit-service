package mongodb

import (
	"context"
	"fmt"

	"github.com/meateam/permit-service/service"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

const (
	// PermitCollectionName is the name of the permits collection.
	PermitCollectionName = "permits"

	// MongoObjectIDField is the default mongodb unique key.
	MongoObjectIDField = "_id"

	// PermitBSONFileIDField is the name of the fileID field in BSON.
	PermitBSONFileIDField = "fileID"

	// PermitBSONUserIDField is the name of the userID field in BSON.
	PermitBSONUserIDField = "userID"

	// PermitBSONReqIDField is the name of the reqID field in BSON.
	PermitBSONReqIDField = "reqID"

	// PermitBSONStatusField is the name of the reqID field in BSON.
	PermitBSONStatusField = "status"
)

// MongoStore holds the mongodb database and implements Store interface.
type MongoStore struct {
	DB *mongo.Database
}

// newMongoStore returns a new store.
func newMongoStore(db *mongo.Database) (MongoStore, error) {
	collection := db.Collection(PermitCollectionName)
	indexes := collection.Indexes()
	// TODO: check ion indices
	indexModel := mongo.IndexModel{
		Keys: bson.D{
			bson.E{
				Key:   PermitBSONFileIDField,
				Value: 1,
			},
			bson.E{
				Key:   PermitBSONUserIDField,
				Value: 1,
			},
		},
		Options: options.Index().SetUnique(true),
	}

	_, err := indexes.CreateOne(context.Background(), indexModel)
	if err != nil {
		return MongoStore{}, err
	}

	return MongoStore{DB: db}, nil
}

// HealthCheck checks the health of the service, returns true if healthy, or false otherwise.
func (s MongoStore) HealthCheck(ctx context.Context) (bool, error) {
	if err := s.DB.Client().Ping(ctx, readpref.Primary()); err != nil {
		return false, err
	}

	return true, nil
}

// Create creates a permit of a file to a user,
// If permit already exists then its updated to have the permit values,
// If successful returns the permit and a nil error,
// otherwise returns empty string and non-nil error if any occured.
func (s MongoStore) Create(ctx context.Context, permit service.Permit) (service.Permit, error) {
	collection := s.DB.Collection(PermitCollectionName)
	fileID := permit.GetFileID()
	if fileID == "" {
		return nil, fmt.Errorf("fileID is required")
	}

	userID := permit.GetUserID()
	if userID == "" {
		return nil, fmt.Errorf("userID is required")
	}

	reqID := permit.GetReqID()
	if reqID == "" {
		return nil, fmt.Errorf("reqID is required")
	}

	status := permit.GetStatus()
	if reqID == "" {
		return nil, fmt.Errorf("reqID is required")
	}

	filter := bson.D{
		bson.E{
			Key:   PermitBSONFileIDField,
			Value: fileID,
		},
		bson.E{
			Key:   PermitBSONUserIDField,
			Value: userID,
		},
	}

	permitUpdate := bson.D{
		bson.E{
			Key:   PermitBSONFileIDField,
			Value: fileID,
		},
		bson.E{
			Key:   PermitBSONUserIDField,
			Value: userID,
		},
		bson.E{
			Key:   PermitBSONReqIDField,
			Value: reqID,
		},
		bson.E{
			Key:   PermitBSONStatusField,
			Value: status,
		},
	}

	update := bson.D{
		bson.E{
			Key:   "$set",
			Value: permitUpdate,
		},
	}

	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)
	result := collection.FindOneAndUpdate(ctx, filter, update, opts)
	newPermit := &BSON{}
	err := result.Decode(newPermit)
	if err != nil {
		return nil, err
	}

	return newPermit, nil
}
