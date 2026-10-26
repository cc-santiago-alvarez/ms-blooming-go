package bondBound_mongocc_repository

import (
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// DateRangeStage devuelve el $match que deja los documentos con field dentro del rango del filtro en la zona
// utcDiff, desde el inicio incluido hasta el final excluido, o nil si el filtro no filtra. Un filtro inválido
// es ErrInvalidDateFilter.
func DateRangeStage(f bondBound_domain.DateFilter, field string, utcDiff int) (bson.M, error) {
	r, err := f.Range(utcDiff)
	if err != nil || r == nil {
		return nil, err
	}
	return bson.M{"$match": bson.M{field: bson.M{"$gte": r.From, "$lt": r.To}}}, nil
}
