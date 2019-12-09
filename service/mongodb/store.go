package mongodb

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// MongoObjectIDField is the default mongodb unique key.
	MongoObjectIDField = "_id"

	// PermitCollectionName is the name of the permits collection.
	PermitCollectionName = "permits"

	// PermitBSONFileIDField is the name of the fileID field in BSON.
	PermitBSONFileIDField = "fileID"

	// PermitBSONUserIDField is the name of the userID field in BSON.
	PermitBSONUserIDField = "userID"

	// PermitBSONRoleField is the name of the role field in BSON.
	PermitBSONRoleField = "role"
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
