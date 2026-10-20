package bondObject_mongocc_repository

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// UpdateKeys cambia key y updatedAt de los productos no borrados de ids. Sin ids no consulta.
func (repo *BondObjectMongoccRepository) UpdateKeys(ctx context.Context, ids []string, key string) (bondObject_domain.UpdateKeysResult, error) {
	if len(ids) == 0 {
		return bondObject_domain.UpdateKeysResult{}, nil
	}

	filter := bson.M{"_id": bson.M{"$in": ids}, "isRemove": false}
	update := bson.M{"$set": bson.M{"key": key, "updatedAt": shared_domain.Now()}}

	res, err := repo.DB.UpdateMany(ctx, collectionName, filter, update, nil)
	if err != nil {
		return bondObject_domain.UpdateKeysResult{}, err
	}
	return bondObject_domain.UpdateKeysResult{Matched: res.MatchedCount, Modified: res.ModifiedCount}, nil
}
