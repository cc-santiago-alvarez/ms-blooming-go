package bondBound_mongocc_repository

import (
	"regexp"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// searchStage devuelve el $match que busca text como texto literal, sin distinguir mayúsculas, en alguno de
// los campos, o nil si text está vacío.
func searchStage(text string, fields ...string) bson.M {
	if text == "" {
		return nil
	}
	like := bson.M{"$regex": regexp.QuoteMeta(text), "$options": "i"}
	or := make(bson.A, len(fields))
	for i, field := range fields {
		or[i] = bson.M{field: like}
	}
	return bson.M{"$match": bson.M{"$or": or}}
}
