package bondBound_mongocc_repository

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// ListByObject devuelve los bonos no borrados del producto, por createdAt descendente. agencyID nil no filtra
// por agencia.
func (repo *BondBoundMongoccRepository) ListByObject(ctx context.Context, bondObjectID string, agencyID *string) ([]*bondBound_domain.BondBound, error) {
	filter := bson.M{"isRemove": false, "bondObjectId": bondObjectID}
	if agencyID != nil {
		filter["agencyId"] = *agencyID
	}

	cursor, err := repo.DB.Find(ctx, collectionName, filter, options.Find().SetSort(newestFirst))
	if err != nil {
		return nil, err
	}

	bounds := []*bondBound_domain.BondBound{}
	if err := cursor.All(ctx, &bounds); err != nil {
		return nil, err
	}
	return bounds, nil
}
