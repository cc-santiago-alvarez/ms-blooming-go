package bondObject_mongocc_repository

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// FindVisible devuelve los productos que ve el inversionista, del más reciente al más antiguo: no
// borrados, de su agencia y, si son privados, solo donde está autorizado.
func (repo *BondObjectMongoccRepository) FindVisible(ctx context.Context, f bondObject_domain.VisibleFilter) ([]*bondObject_domain.BondObject, error) {
	filter := bson.M{
		"isRemove": false,
		"agencyId": f.AgencyID,
		"$or":      visibleTo(f.UserID),
	}
	if f.BondTypeID != "" {
		filter["bondTypeId"] = f.BondTypeID
	}
	if len(f.IDs) > 0 {
		filter["_id"] = bson.M{"$in": f.IDs}
	}
	if f.OnlyActive {
		filter["active"] = true
	}
	opts := options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: 1}})
	if f.Limit > 0 {
		opts.SetLimit(f.Limit)
	}

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
