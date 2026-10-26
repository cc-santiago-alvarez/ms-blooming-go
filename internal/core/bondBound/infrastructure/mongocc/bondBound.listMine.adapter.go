package bondBound_mongocc_repository

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ListMine devuelve la página p, ya normalizada, de los bonos no borrados del usuario en la agencia, por
// createdAt descendente, con el nombre de su producto; los bonos sin producto no aparecen. statusIDs vacío no
// filtra por estado.
func (repo *BondBoundMongoccRepository) ListMine(ctx context.Context, userID, agencyID string, statusIDs []bondBound_domain.BondBoundStatus, p shared_domain.Pagination) (shared_domain.Paginated[bondBound_ports.BondBoundWithObject], error) {
	match := bson.M{"isRemove": false, "userId": userID, "agencyId": agencyID}
	if len(statusIDs) > 0 {
		match["statusId"] = bson.M{"$in": statusIDs}
	}

	pipeline := bson.A{bson.M{"$match": match}, bson.M{"$sort": newestFirst}, nestBound}
	pipeline = append(pipeline, lookupOne(bondObjectCollection, "bound.bondObjectId", "bondObject",
		bson.M{"_id": 0, "name": 1}, true)...)

	return aggregatePage[bondBound_ports.BondBoundWithObject](ctx, repo.DB, pipeline, p)
}
