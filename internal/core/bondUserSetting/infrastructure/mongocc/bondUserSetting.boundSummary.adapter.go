package bondUserSetting_mongocc_repository

import (
	"context"

	"github.com/codecraftkit/mongocc"
	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Colección de los bonos comprados; solo se lee.
const bondBoundCollection = "bondBound"

// BoundSummaryMongoccRepository implementa BoundSummaryPort con una agregación de solo lectura sobre
// bondBound.
type BoundSummaryMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondUserSetting_ports.BoundSummaryPort = (*BoundSummaryMongoccRepository)(nil)

// BoundSummary suma amount de los bonos no borrados de userID en agencyID, por estado, y arma el resumen.
// Como en el legado, un amount no numérico no suma.
func (repo *BoundSummaryMongoccRepository) BoundSummary(ctx context.Context, userID, agencyID string) (bondUserSetting_domain.BoundSummary, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"isRemove": false, "agencyId": agencyID, "userId": userID}},
		bson.M{"$group": bson.M{"_id": "$statusId", "total": bson.M{"$sum": "$amount"}}},
	}

	cursor, err := repo.DB.Aggregate(ctx, bondBoundCollection, pipeline, nil)
	if err != nil {
		return bondUserSetting_domain.BoundSummary{}, err
	}

	var rows []struct {
		StatusID string  `bson:"_id"`
		Total    float64 `bson:"total"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return bondUserSetting_domain.BoundSummary{}, err
	}

	totals := make(map[string]float64, len(rows))
	for _, r := range rows {
		totals[r.StatusID] = r.Total
	}
	return bondUserSetting_domain.NewBoundSummary(totals), nil
}
