package bondType_mongocc_repository

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// FindManyByIDs devuelve los tipos no borrados que existan entre ids; los demás no aparecen. Sin ids no
// consulta.
func (repo *BondTypeMongoccRepository) FindManyByIDs(ctx context.Context, ids []string) ([]*bondType_domain.BondType, error) {
	types := []*bondType_domain.BondType{}
	if len(ids) == 0 {
		return types, nil
	}

	filter := bson.M{"_id": bson.M{"$in": ids}, "isRemove": false}
	cursor, err := repo.DB.Find(ctx, collectionName, filter, nil)
	if err != nil {
		return nil, err
	}
	if err := cursor.All(ctx, &types); err != nil {
		return nil, err
	}
	return types, nil
}
