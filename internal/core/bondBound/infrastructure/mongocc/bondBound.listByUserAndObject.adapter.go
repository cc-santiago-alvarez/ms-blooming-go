package bondBound_mongocc_repository

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// userObjectStatuses son los estados de los bonos de un usuario en un producto: iniciados, madurados,
// bloqueados y canjeados.
var userObjectStatuses = []bondBound_domain.BondBoundStatus{
	bondBound_domain.StatusInitiated,
	bondBound_domain.StatusFinalized,
	bondBound_domain.StatusBlocked,
	bondBound_domain.StatusTook,
}

// ListByUserAndObject devuelve los bonos no borrados del usuario y el producto que están iniciados,
// madurados, bloqueados o canjeados, por createdAt descendente. agencyID nil no filtra por agencia.
func (repo *BondBoundMongoccRepository) ListByUserAndObject(ctx context.Context, userID, bondObjectID string, agencyID *string) ([]*bondBound_domain.BondBound, error) {
	filter := bson.M{
		"isRemove":     false,
		"userId":       userID,
		"bondObjectId": bondObjectID,
		"statusId":     bson.M{"$in": userObjectStatuses},
	}
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
