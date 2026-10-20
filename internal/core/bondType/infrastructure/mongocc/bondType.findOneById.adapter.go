package bondType_mongocc_repository

import (
	"context"
	"errors"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// FindOneByID devuelve el tipo no borrado, o nil, nil si no existe o está borrado.
func (repo *BondTypeMongoccRepository) FindOneByID(ctx context.Context, id string) (*bondType_domain.BondType, error) {
	filter := bson.M{"_id": id, "isRemove": false}

	var t bondType_domain.BondType
	if err := repo.DB.FindOne(ctx, collectionName, filter, nil).Decode(&t); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}
