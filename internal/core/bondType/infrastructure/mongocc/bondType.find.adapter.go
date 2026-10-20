package bondType_mongocc_repository

import (
	"context"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Find devuelve los tipos no borrados por name ascendente. agencyIDs vacío no filtra por agencia;
// onlyActive deja solo los activos. $in sirve para agencyId guardado como string o como arreglo.
func (repo *BondTypeMongoccRepository) Find(ctx context.Context, agencyIDs []string, onlyActive bool) ([]*bondType_domain.BondType, error) {
	filter := bson.M{"isRemove": false}
	if onlyActive {
		filter["active"] = true
	}
	if len(agencyIDs) > 0 {
		filter["agencyId"] = bson.M{"$in": agencyIDs}
	}
	opts := options.Find().SetSort(bson.D{{Key: "name", Value: 1}})

	cursor, err := repo.DB.Find(ctx, collectionName, filter, opts)
	if err != nil {
		return nil, err
	}

	types := []*bondType_domain.BondType{}
	if err := cursor.All(ctx, &types); err != nil {
		return nil, err
	}
	return types, nil
}
