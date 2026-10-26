package bondBound_mongocc_repository

import (
	"context"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// timelineFormat es el formato de las fechas de la línea de tiempo, con el año calendario.
const timelineFormat = "%Y-%m-%d %H:%M"

// Timeline devuelve las compras no borradas del usuario por fecha inicial: "<producto> (<clave>)" y la fecha
// inicial en la zona utcDiff como inicio y fin. Sin producto, el texto queda vacío.
func (repo *BondBoundMongoccRepository) Timeline(ctx context.Context, userID string, utcDiff int) ([]bondBound_ports.TimelineItem, error) {
	pipeline := bson.A{
		bson.M{"$match": bson.M{"isRemove": false, "userId": userID}},
		bson.M{"$sort": bson.D{{Key: "initialDate", Value: 1}, {Key: "_id", Value: 1}}},
	}
	pipeline = append(pipeline, lookupOne(bondObjectCollection, "bondObjectId", "bondObject",
		bson.M{"_id": 0, "name": 1}, false)...)
	date := bson.M{"$dateToString": bson.M{
		"format":   timelineFormat,
		"date":     "$initialDate",
		"timezone": timezone(utcDiff),
	}}
	pipeline = append(pipeline, bson.M{"$project": bson.M{
		"text":  bson.M{"$concat": bson.A{"$bondObject.name", " (", "$key", ")"}},
		"start": date,
		"end":   date,
	}})

	cursor, err := repo.DB.Aggregate(ctx, collectionName, pipeline, nil)
	if err != nil {
		return nil, err
	}

	items := []bondBound_ports.TimelineItem{}
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}
