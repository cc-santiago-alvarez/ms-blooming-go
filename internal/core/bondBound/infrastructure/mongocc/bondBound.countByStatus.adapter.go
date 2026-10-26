package bondBound_mongocc_repository

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// kpiSearchFields son los campos donde busca el texto del KPI de bonos por estado.
var kpiSearchFields = []string{"profile.id", "profile.first_name", "profile.last_name", "profile.email", "bondObject.name"}

// CountByStatus devuelve la suma de los montos y la cantidad de los bonos no borrados que pasan f; Status queda
// vacío. El filtro de fecha usa la zona utcDiff, y agencyID nil no filtra por agencia. Con búsqueda, los bonos
// sin dueño o sin producto no cuentan.
func (repo *BondBoundMongoccRepository) CountByStatus(ctx context.Context, agencyID *string, f bondBound_domain.KpiFilters, utcDiff int) (bondBound_ports.StatusCount, error) {
	match := bson.M{"isRemove": false}
	if agencyID != nil {
		match["agencyId"] = *agencyID
	}
	if f.StatusID != "" {
		match["statusId"] = f.StatusID
	}
	if f.UserID != "" {
		match["userId"] = f.UserID
	}

	pipeline := bson.A{bson.M{"$match": match}}
	dateStage, err := DateRangeStage(f.Date, "createdAt", utcDiff)
	if err != nil {
		return bondBound_ports.StatusCount{}, err
	}
	if dateStage != nil {
		pipeline = append(pipeline, dateStage)
	}
	if search := searchStage(f.Search, kpiSearchFields...); search != nil {
		pipeline = append(pipeline, lookupOne(profileCollection, "userId", "profile",
			bson.M{"_id": 0, "id": 1, "first_name": 1, "last_name": 1, "email": 1}, true)...)
		pipeline = append(pipeline, lookupOne(bondObjectCollection, "bondObjectId", "bondObject",
			bson.M{"_id": 0, "name": 1}, true)...)
		pipeline = append(pipeline, search)
	}
	pipeline = append(pipeline, bson.M{"$group": bson.M{
		"_id":   nil,
		"total": bson.M{"$sum": numericAmount},
		"count": bson.M{"$sum": 1},
	}})

	cursor, err := repo.DB.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return bondBound_ports.StatusCount{}, err
	}
	var result []bondBound_ports.StatusCount
	if err := cursor.All(ctx, &result); err != nil {
		return bondBound_ports.StatusCount{}, err
	}
	if len(result) == 0 {
		return bondBound_ports.StatusCount{}, nil
	}
	return result[0], nil
}
