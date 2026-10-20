package bondObject_mongocc_repository

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Find devuelve los productos no borrados por name ascendente; agencyIDs vacío no filtra por agencia.
func (repo *BondObjectMongoccRepository) Find(ctx context.Context, agencyIDs []string) ([]*bondObject_domain.BondObject, error) {
	filter := bson.M{"isRemove": false}
	if len(agencyIDs) > 0 {
		filter["agencyId"] = bson.M{"$in": agencyIDs}
	}
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := repo.DB.Find(ctx, collectionName, filter, opts)
	if err != nil {
		return nil, err
	}

	objects := []*bondObject_domain.BondObject{}
	if err := cursor.All(ctx, &objects); err != nil {
		return nil, err
	}
	return objects, nil
}
