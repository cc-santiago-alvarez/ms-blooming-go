package bondUserSetting_mongocc_repository

import (
	"context"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// SoftDelete marca el setting con isRemove true y actualiza updatedAt; devuelve false si no existe o ya
// estaba borrado.
func (repo *BondUserSettingMongoccRepository) SoftDelete(ctx context.Context, id string) (bool, error) {
	filter := bson.M{"_id": id, "isRemove": false}
	update := bson.M{"$set": bson.M{"isRemove": true, "updatedAt": shared_domain.Now()}}

	res, err := repo.DB.UpdateOne(ctx, collectionName, filter, update, nil)
	if err != nil {
		return false, err
	}
	return res.MatchedCount == 1, nil
}
