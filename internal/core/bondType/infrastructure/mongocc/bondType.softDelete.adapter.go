package bondType_mongocc_repository

import (
	"context"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// SoftDelete marca el tipo con isRemove true y actualiza updatedAt.
func (repo *BondTypeMongoccRepository) SoftDelete(ctx context.Context, id string) error {
	filter := bson.M{"_id": id, "isRemove": false}
	update := bson.M{"$set": bson.M{"isRemove": true, "updatedAt": shared_domain.Now()}}

	_, err := repo.DB.UpdateOne(ctx, collectionName, filter, update, nil)
	return err
}
