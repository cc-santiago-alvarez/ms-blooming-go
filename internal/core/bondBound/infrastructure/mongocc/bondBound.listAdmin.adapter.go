package bondBound_mongocc_repository

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// adminSearchFields son los campos donde busca el texto del listado de administración.
var adminSearchFields = []string{
	"profile.id", "profile.first_name", "profile.last_name", "profile.email",
	"bondObject.key", "bondObject.name", "bondObject.description", "bondType.name",
}

// ListAdmin devuelve la página p, ya normalizada, de los bonos no borrados que pasan f, por createdAt
// descendente, con su dueño, su producto y su tipo; los bonos sin producto o sin tipo no aparecen. El filtro
// de fecha, el mes y el año usan la zona utcDiff. agencyID nil no filtra por agencia.
func (repo *BondBoundMongoccRepository) ListAdmin(ctx context.Context, agencyID *string, f bondBound_domain.ListFilters, p shared_domain.Pagination, utcDiff int) (shared_domain.Paginated[bondBound_ports.BondBoundAdminRow], error) {
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
		return shared_domain.Paginated[bondBound_ports.BondBoundAdminRow]{}, err
	}
	if dateStage != nil {
		pipeline = append(pipeline, dateStage)
	}
	pipeline = append(pipeline, bson.M{"$sort": newestFirst}, nestBound)
	// el perfil sale solo con los campos de la búsqueda y del listado
	pipeline = append(pipeline, lookupOne(profileCollection, "bound.userId", "profile",
		bson.M{"_id": 0, "id": 1, "first_name": 1, "last_name": 1, "email": 1}, false)...)
	pipeline = append(pipeline, lookupOne(bondObjectCollection, "bound.bondObjectId", "bondObject",
		bson.M{"_id": 0, "name": 1, "key": 1, "description": 1}, true)...)
	pipeline = append(pipeline, lookupOne(bondTypeCollection, "bound.bondTypeId", "bondType",
		bson.M{"_id": 0, "name": 1}, true)...)
	if search := searchStage(f.Search, adminSearchFields...); search != nil {
		pipeline = append(pipeline, search)
	}
	zone := timezone(utcDiff)
	pipeline = append(pipeline, bson.M{"$set": bson.M{
		"month": bson.M{"$month": bson.M{"date": "$bound.createdAt", "timezone": zone}},
		"year":  bson.M{"$year": bson.M{"date": "$bound.createdAt", "timezone": zone}},
	}})

	return aggregatePage[bondBound_ports.BondBoundAdminRow](ctx, repo.DB, pipeline, p)
}
