package bondBound_mongocc_repository

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ReportSummaryRows devuelve el total comprado por cada usuario en los bonos no borrados de la agencia, del
// mayor al menor, con su perfil. De f solo usa el estado.
func (repo *BondBoundMongoccRepository) ReportSummaryRows(ctx context.Context, agencyID string, f bondBound_domain.ReportFilters) ([]bondBound_ports.SummaryReportRow, error) {
	match := bson.M{"isRemove": false, "agencyId": agencyID}
	if !f.AllStatuses() {
		match["statusId"] = f.StatusID
	}

	pipeline := bson.A{
		bson.M{"$match": match},
		bson.M{"$group": bson.M{"_id": "$userId", "total": bson.M{"$sum": numericAmount}}},
	}
	pipeline = append(pipeline, lookupOne(profileCollection, "_id", "profile",
		bson.M{"_id": 0, "first_name": 1, "last_name": 1}, false)...)
	pipeline = append(pipeline, bson.M{"$sort": bson.D{{Key: "total", Value: -1}, {Key: "_id", Value: 1}}})

	cursor, err := repo.DB.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return nil, err
	}
	rows := []bondBound_ports.SummaryReportRow{}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	return rows, nil
}
