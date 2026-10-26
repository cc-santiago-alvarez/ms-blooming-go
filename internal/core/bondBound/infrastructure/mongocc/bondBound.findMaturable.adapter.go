package bondBound_mongocc_repository

import (
	"context"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// byID es el orden del cursor de los lotes: por _id ascendente.
var byID = bson.D{{Key: "_id", Value: 1}}

// FindMaturable devuelve hasta limit bonos no borrados e iniciados, sin soldDate ni tookDate y con
// maturityDate hasta now, por _id ascendente y después de afterID. afterID vacío empieza por el primero y
// limit 0 no limita.
func (repo *BondBoundMongoccRepository) FindMaturable(ctx context.Context, now time.Time, limit int, afterID string) ([]*bondBound_domain.BondBound, error) {
	filter := bson.M{
		"isRemove":     false,
		"statusId":     bondBound_domain.StatusInitiated,
		"maturityDate": bson.M{"$lte": now},
		"soldDate":     nil,
		"tookDate":     nil,
	}
	if afterID != "" {
		filter["_id"] = bson.M{"$gt": afterID}
	}

	opts := options.Find().SetSort(byID)
	if limit > 0 {
		opts.SetLimit(int64(limit))
	}
	cursor, err := repo.DB.Find(ctx, collectionName, filter, opts)
	if err != nil {
		return nil, err
	}

	bounds := []*bondBound_domain.BondBound{}
	if err := cursor.All(ctx, &bounds); err != nil {
		return nil, err
	}
	return bounds, nil
}
