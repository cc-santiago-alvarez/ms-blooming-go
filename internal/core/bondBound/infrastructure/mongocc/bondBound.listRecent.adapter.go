package bondBound_mongocc_repository

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// recentStatuses son los estados de los bonos recientes: iniciados y madurados.
var recentStatuses = []bondBound_domain.BondBoundStatus{
	bondBound_domain.StatusInitiated,
	bondBound_domain.StatusFinalized,
}

// ListRecent devuelve los limit bonos no borrados más recientes del usuario que están iniciados o madurados,
// de cualquier agencia, con el nombre y los términos de retiro de su producto. limit 0 no limita.
func (repo *BondBoundMongoccRepository) ListRecent(ctx context.Context, userID string, limit int) ([]bondBound_ports.BondBoundWithObject, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"isRemove": false, "userId": userID, "statusId": bson.M{"$in": recentStatuses}}},
		bson.M{"$sort": newestFirst},
	}
	if limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": limit})
	}
	pipeline = append(pipeline, nestBound)
	pipeline = append(pipeline, lookupOne(bondObjectCollection, "bound.bondObjectId", "bondObject",
		bson.M{"_id": 0, "name": 1, "earlyCashTerm": 1}, false)...)

	cursor, err := repo.DB.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return nil, err
	}

	rows := []bondBound_ports.BondBoundWithObject{}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
