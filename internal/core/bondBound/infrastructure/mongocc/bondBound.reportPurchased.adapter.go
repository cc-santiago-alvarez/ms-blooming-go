package bondBound_mongocc_repository

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// reportSearchFields son los campos donde busca el texto del reporte de comprados.
var reportSearchFields = []string{"profile.first_name", "bondObject.name", "bondType.name"}

// reportRow es un bono del reporte con lo que traen los lookups.
type reportRow struct {
	Bound      bondBound_domain.BondBound    `bson:"bound"`
	Profile    *bondBound_ports.ProfileName  `bson:"profile"`
	BondObject bondBound_ports.BondObjectRef `bson:"bondObject"`
	BondType   bondBound_ports.BondTypeRef   `bson:"bondType"`
}

// ReportPurchasedRows devuelve los bonos no borrados de la agencia que pasan f, por createdAt descendente, con
// su dueño, su producto, su tipo y el nombre de su estado; los bonos sin producto o sin tipo no aparecen. El
// filtro de fecha usa la zona utcDiff.
func (repo *BondBoundMongoccRepository) ReportPurchasedRows(ctx context.Context, agencyID string, f bondBound_domain.ReportFilters, utcDiff int) ([]bondBound_ports.PurchasedReportRow, error) {
	match := bson.M{"isRemove": false, "agencyId": agencyID}
	if !f.AllStatuses() {
		match["statusId"] = f.StatusID
	}
	if f.UserID != "" {
		match["userId"] = f.UserID
	}

	pipeline := bson.A{bson.M{"$match": match}}
	dateStage, err := DateRangeStage(f.Date, "createdAt", utcDiff)
	if err != nil {
		return nil, err
	}
	if dateStage != nil {
		pipeline = append(pipeline, dateStage)
	}
	pipeline = append(pipeline, bson.M{"$sort": newestFirst}, nestBound)
	pipeline = append(pipeline, lookupOne(profileCollection, "bound.userId", "profile",
		bson.M{"_id": 0, "first_name": 1, "last_name": 1}, false)...)
	pipeline = append(pipeline, lookupOne(bondObjectCollection, "bound.bondObjectId", "bondObject",
		bson.M{"_id": 0, "name": 1}, true)...)
	pipeline = append(pipeline, lookupOne(bondTypeCollection, "bound.bondTypeId", "bondType",
		bson.M{"_id": 0, "name": 1}, true)...)
	if search := searchStage(f.Search, reportSearchFields...); search != nil {
		pipeline = append(pipeline, search)
	}

	cursor, err := repo.DB.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return nil, err
	}
	var rows []reportRow
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}

	out := make([]bondBound_ports.PurchasedReportRow, len(rows))
	for i, row := range rows {
		b := row.Bound
		out[i] = bondBound_ports.PurchasedReportRow{
			Key:                       b.Key,
			Status:                    b.StatusID.Name(),
			Amount:                    b.Amount,
			MaturityTimeInDays:        b.MaturityTimeInDays,
			InitialDate:               b.InitialDate,
			MaturityDate:              b.MaturityDate,
			MaturityRewardsPercentage: b.MaturityRewardsPercentage,
			Profile:                   row.Profile,
			BondType:                  row.BondType,
			BondObject:                row.BondObject,
			CreatedAt:                 b.CreatedAt,
		}
	}
	return out, nil
}
