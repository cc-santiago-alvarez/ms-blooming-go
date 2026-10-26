package bondBound_mongocc_repository

import (
	"context"
	"fmt"

	"github.com/codecraftkit/mongocc"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Colección de los bonos comprados; la comparte con Meteor.
const collectionName = "bondBound"

// Colecciones que leen las consultas con lookups; nunca escriben en ellas.
const (
	profileCollection    = "profile"
	bondObjectCollection = "bondObject"
	bondTypeCollection   = "bondType"
)

// BondBoundMongoccRepository implementa BondBoundMongoccRepositoryPorts sobre bondBound.
type BondBoundMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondBound_ports.BondBoundMongoccRepositoryPorts = (*BondBoundMongoccRepository)(nil)

// nestBound deja el bono en el campo bound, para decodificarlo con su lector legacy junto a lo que traen los
// lookups.
var nestBound = bson.M{"$replaceWith": bson.M{"bound": "$$ROOT"}}

// newestFirst es el orden de los listados: del más reciente al más antiguo.
var newestFirst = bson.D{{Key: "createdAt", Value: -1}, {Key: "_id", Value: 1}}

// numericAmount es el monto del bono como número para sumarlo: los legacy pueden guardarlo como string, y uno
// que no es número cuenta 0.
var numericAmount = bson.M{"$convert": bson.M{"input": "$amount", "to": "double", "onError": 0, "onNull": 0}}

// lookupOne trae en as el documento de from cuyo _id es localField, con la proyección project si no es nil.
// Con required, el documento sin coincidencia no sigue; si no, sigue sin as.
func lookupOne(from, localField, as string, project bson.M, required bool) bson.A {
	pipeline := bson.A{bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$_id", "$$id"}}}}}
	if project != nil {
		pipeline = append(pipeline, bson.M{"$project": project})
	}
	return bson.A{
		bson.M{"$lookup": bson.M{
			"from":     from,
			"let":      bson.M{"id": "$" + localField},
			"pipeline": pipeline,
			"as":       as,
		}},
		bson.M{"$unwind": bson.M{"path": "$" + as, "preserveNullAndEmptyArrays": !required}},
	}
}

// timezone devuelve la zona utcDiff (horas) en el formato de Mongo, por ejemplo "-05:00".
func timezone(utcDiff int) string {
	return fmt.Sprintf("%+03d:00", utcDiff)
}

// aggregatePage corre el pipeline sobre bondBound y devuelve la página p, ya normalizada, con el total de
// documentos que salen del pipeline, en una sola agregación.
func aggregatePage[T any](ctx context.Context, db *mongocc.MongoQueries, pipeline bson.A, p shared_domain.Pagination) (shared_domain.Paginated[T], error) {
	var page shared_domain.Paginated[T]

	pipeline = append(pipeline, bson.M{"$facet": bson.M{
		"items": bson.A{bson.M{"$skip": p.Skip()}, bson.M{"$limit": p.Limit}},
		"total": bson.A{bson.M{"$count": "n"}},
	}})
	cursor, err := db.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return page, err
	}

	var result []struct {
		Items []T `bson:"items"`
		Total []struct {
			N int64 `bson:"n"`
		} `bson:"total"`
	}
	if err := cursor.All(ctx, &result); err != nil {
		return page, err
	}

	var items []T
	var total int64
	if len(result) > 0 {
		items = result[0].Items
		if len(result[0].Total) > 0 {
			total = result[0].Total[0].N
		}
	}
	return shared_domain.NewPaginated(items, total, p), nil
}
