package bondObject_mongocc_repository

import (
	"context"

	"github.com/codecraftkit/mongocc"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Colecciones que lee el adapter de estadísticas; nunca escribe en ellas.
const (
	bondBoundCollection = "bondBound"
	profileCollection   = "profile"
)

// liveBoundStatuses son los estados de bono que cuentan para el listado de inversionistas, como en el
// legado.
var liveBoundStatuses = []string{"1", "2", "3", "6"}

// BondBoundStatsMongoccRepository implementa BondBoundStatsPort con agregaciones de solo lectura sobre
// bondBound.
type BondBoundStatsMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondObject_ports.BondBoundStatsPort = (*BondBoundStatsMongoccRepository)(nil)

// TopPurchasedObjectIDs devuelve los limit productos con más bonos no borrados de la agencia, de más a
// menos compras; limit 0 no limita.
func (repo *BondBoundStatsMongoccRepository) TopPurchasedObjectIDs(ctx context.Context, agencyID string, limit int) ([]string, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"isRemove": false, "agencyId": agencyID}},
		bson.M{"$group": bson.M{"_id": "$bondObjectId", "total": bson.M{"$sum": 1}}},
		bson.M{"$sort": bson.D{{Key: "total", Value: -1}, {Key: "_id", Value: 1}}},
	}
	if limit > 0 {
		pipeline = append(pipeline, bson.M{"$limit": limit})
	}

	cursor, err := repo.DB.Aggregate(ctx, bondBoundCollection, pipeline, nil)
	if err != nil {
		return nil, err
	}

	var rows []struct {
		ObjectID string `bson:"_id"`
	}
	if err := cursor.All(ctx, &rows); err != nil {
		return nil, err
	}
	ids := make([]string, len(rows))
	for i, r := range rows {
		ids[i] = r.ObjectID
	}
	return ids, nil
}

// InvestorsByObject agrupa por usuario los bonos vigentes del producto, con los datos públicos de su
// perfil; el usuario sin perfil no aparece. agencyID nil no filtra por agencia.
func (repo *BondBoundStatsMongoccRepository) InvestorsByObject(ctx context.Context, objectID string, agencyID *string) ([]bondObject_domain.BondObjectInvestor, error) {
	match := bson.M{
		"isRemove":     false,
		"bondObjectId": objectID,
		"statusId":     bson.M{"$in": liveBoundStatuses},
	}
	if agencyID != nil {
		match["agencyId"] = *agencyID
	}

	pipeline := bson.A{
		bson.M{"$match": match},
		bson.M{"$group": bson.M{"_id": "$userId", "numOfBounds": bson.M{"$sum": 1}}},
		// la proyección va dentro del $lookup para que el resto del perfil no salga de Mongo
		bson.M{"$lookup": bson.M{
			"from": profileCollection,
			"let":  bson.M{"userId": "$_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$userId"}}}},
				bson.M{"$project": bson.M{
					"_id": 0, "id": 1, "first_name": 1, "last_name": 1, "email": 1,
					"avatar": 1, "address": 1, "avatar_url": 1,
				}},
			},
			"as": "investor",
		}},
		bson.M{"$unwind": "$investor"},
		bson.M{"$sort": bson.D{{Key: "_id", Value: 1}}},
	}

	cursor, err := repo.DB.Aggregate(ctx, bondBoundCollection, pipeline, nil)
	if err != nil {
		return nil, err
	}

	investors := []bondObject_domain.BondObjectInvestor{}
	if err := cursor.All(ctx, &investors); err != nil {
		return nil, err
	}
	return investors, nil
}
