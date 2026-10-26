package bondBound_mongocc_repository_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/bondBound/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// El $match filtra el campo recibido por el rango [desde, hasta) del filtro en la zona de la sesión, sin
// $month ni $year. CA-06.3
func TestDateRangeStage(t *testing.T) {
	tests := []struct {
		name    string
		filter  bondBound_domain.DateFilter
		field   string
		utcDiff int
		want    bson.M
	}{
		{
			name:    "aDate sobre createdAt",
			filter:  bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterADate, ADate: "2026-10-13"},
			field:   "createdAt",
			utcDiff: -5,
			want: bson.M{"$match": bson.M{"createdAt": bson.M{
				"$gte": time.Date(2026, 10, 13, 5, 0, 0, 0, time.UTC),
				"$lt":  time.Date(2026, 10, 14, 5, 0, 0, 0, time.UTC),
			}}},
		},
		{
			name:    "aMonth sobre initialDate",
			filter:  bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAMonth, Month: 10, Year: 2026},
			field:   "initialDate",
			utcDiff: -5,
			want: bson.M{"$match": bson.M{"initialDate": bson.M{
				"$gte": time.Date(2026, 10, 1, 5, 0, 0, 0, time.UTC),
				"$lt":  time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC),
			}}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bondBound_mongocc_repository.DateRangeStage(tt.filter, tt.field, tt.utcDiff)
			if err != nil {
				t.Fatalf("error %v, se esperaba la etapa", err)
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("etapa %v, se esperaba %v", got, tt.want)
			}
		})
	}
}

// Sin tipo de filtro no hay etapa. CA-06.3
func TestDateRangeStageWithoutFilter(t *testing.T) {
	got, err := bondBound_mongocc_repository.DateRangeStage(bondBound_domain.DateFilter{}, "createdAt", -5)

	if got != nil || err != nil {
		t.Errorf("etapa %v y error %v, se esperaban nil", got, err)
	}
}

// Un betweenDates con startDate posterior a endDate es ERR_INVALID_DATE_FILTER y no arma la etapa. CA-06.3
func TestDateRangeStageInvalid(t *testing.T) {
	f := bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: "2026-10-14", EndDate: "2026-10-13"}

	got, err := bondBound_mongocc_repository.DateRangeStage(f, "createdAt", -5)

	if got != nil || !errors.Is(err, bondBound_domain.ErrInvalidDateFilter) {
		t.Errorf("etapa %v y error %v, se esperaba ERR_INVALID_DATE_FILTER", got, err)
	}
}
