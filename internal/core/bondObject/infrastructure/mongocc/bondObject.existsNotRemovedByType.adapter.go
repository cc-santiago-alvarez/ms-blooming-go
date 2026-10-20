package bondObject_mongocc_repository

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ExistsNotRemovedByType indica si algún producto no borrado usa el tipo.
func (repo *BondObjectMongoccRepository) ExistsNotRemovedByType(ctx context.Context, bondTypeID string) (bool, error) {
	filter := bson.M{"bondTypeId": bondTypeID, "isRemove": false}
	opts := options.FindOne().SetProjection(bson.M{"_id": 1})

	var found bson.M
	if err := repo.DB.FindOne(ctx, collectionName, filter, opts).Decode(&found); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
