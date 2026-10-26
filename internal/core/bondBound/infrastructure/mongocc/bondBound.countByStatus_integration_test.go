//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// betweenDates del 2026-01-25 al 2026-02-05 en UTC-5 compara fechas completas: cuenta la compra del 30 de
// enero, la del primer instante del 25 y la del último segundo del 5, y deja fuera la de un segundo antes y la
// del 6 de febrero. CA-07.5
func TestCountByStatusBetweenDates(t *testing.T) {
	repo := setup(t)
	bound := func(minute int, at string, amount float64) bson.M {
		return boundAt(nextBoundID(), minute, bson.M{"createdAt": createdAt(t, at), "amount": amount})
	}
	insert(t, boundCollection,
		bound(0, "2026-01-30T15:00:00Z", 1000),
		bound(1, "2026-01-25T05:00:00Z", 200),
		bound(2, "2026-02-06T04:59:59Z", 30),
		bound(3, "2026-01-25T04:59:59Z", 4),
		bound(4, "2026-02-06T05:00:00Z", 5))
	f := bondBound_domain.KpiFilters{
		Date: bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: "2026-01-25", EndDate: "2026-02-05"},
	}

	count, err := repo.CountByStatus(context.Background(), agencyScope(agencyA), f, -5)
	if err != nil {
		t.Fatalf("CountByStatus: %v", err)
	}

	if want := (bondBound_ports.StatusCount{Total: 1230, Count: 3}); count != want {
		t.Errorf("CountByStatus = %+v, se esperaba %+v", count, want)
	}
}

// El KPI suma los montos, incluido el "500" guardado como string, y cuenta los bonos no borrados de la agencia
// pedida, o de todas sin agencia. El estado y el usuario filtran exacto, sea el usuario Meteor o UUID, y Status
// queda vacío. CA-07.6, CA-07.8
func TestCountByStatusScopeAndFilters(t *testing.T) {
	repo := setup(t)
	insert(t, boundCollection,
		boundAt(legacyBoundID, 0, nil),
		boundAt(boundID, 1, bson.M{"userId": user2ID, "amount": "500"}),
		boundAt(legacyBound2ID, 2, bson.M{"statusId": "2", "amount": 200.0}),
		boundAt(nextBoundID(), 3, bson.M{"agencyId": agencyB, "amount": 7000.0}),
		boundAt(nextBoundID(), 4, bson.M{"isRemove": true, "amount": 9999.0}))

	tests := []struct {
		name     string
		agencyID *string
		filters  bondBound_domain.KpiFilters
		want     bondBound_ports.StatusCount
	}{
		{name: "agencia A", agencyID: agencyScope(agencyA), want: bondBound_ports.StatusCount{Total: 1700, Count: 3}},
		{name: "iniciados", agencyID: agencyScope(agencyA), filters: bondBound_domain.KpiFilters{StatusID: bondBound_domain.StatusInitiated}, want: bondBound_ports.StatusCount{Total: 1500, Count: 2}},
		{name: "usuario Meteor", agencyID: agencyScope(agencyA), filters: bondBound_domain.KpiFilters{UserID: userID}, want: bondBound_ports.StatusCount{Total: 1200, Count: 2}},
		{name: "usuario UUID", agencyID: agencyScope(agencyA), filters: bondBound_domain.KpiFilters{UserID: user2ID}, want: bondBound_ports.StatusCount{Total: 500, Count: 1}},
		{name: "todas las agencias", want: bondBound_ports.StatusCount{Total: 8700, Count: 4}},
		{name: "usuario Meteor en todas las agencias", filters: bondBound_domain.KpiFilters{UserID: userID}, want: bondBound_ports.StatusCount{Total: 8200, Count: 3}},
		{name: "estado sin bonos", agencyID: agencyScope(agencyA), filters: bondBound_domain.KpiFilters{StatusID: bondBound_domain.StatusTook}, want: bondBound_ports.StatusCount{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := repo.CountByStatus(context.Background(), tt.agencyID, tt.filters, -5)
			if err != nil {
				t.Fatalf("CountByStatus: %v", err)
			}

			if count != tt.want {
				t.Errorf("CountByStatus = %+v, se esperaba %+v", count, tt.want)
			}
		})
	}
}

// La búsqueda del KPI no distingue mayúsculas y encuentra por la cédula, el nombre, el apellido o el email del
// dueño o por el nombre del producto; el texto se busca literal. Con búsqueda, el bono sin dueño o sin producto
// no cuenta; sin búsqueda cuentan todos. CA-07.2
func TestCountByStatusSearch(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	const (
		luisID   = "PrfLuis0000000001"
		regexObj = "ObjRegex000000001"
	)
	insert(t, profileCollection, profileDoc(luisID, "3333333333", "Luis", "Mora", "luis@futurapps.com"))
	insert(t, objectCollection, objectDoc(regexObj, "Bono a.*( Especial", "BE01", legacyTypeID, nil))
	insert(t, boundCollection,
		boundAt(nextBoundID(), 0, bson.M{"amount": 1000.0}),
		boundAt(nextBoundID(), 1, bson.M{"userId": user2ID, "bondObjectId": legacyObjectID, "amount": 100.0}),
		boundAt(nextBoundID(), 2, bson.M{"userId": shared_domain.NewID(), "amount": 10.0}),
		boundAt(nextBoundID(), 3, bson.M{"bondObjectId": missingID, "amount": 1.0}),
		boundAt(nextBoundID(), 4, bson.M{"userId": luisID, "bondObjectId": regexObj, "amount": 5.0}))

	tests := []struct {
		name   string
		search string
		want   bondBound_ports.StatusCount
	}{
		{name: "sin búsqueda", search: "", want: bondBound_ports.StatusCount{Total: 1116, Count: 5}},
		{name: "cédula", search: "1020304050", want: bondBound_ports.StatusCount{Total: 1000, Count: 1}},
		{name: "apellido", search: "ramírez", want: bondBound_ports.StatusCount{Total: 1000, Count: 1}},
		{name: "apellido en mayúsculas", search: "TORRES", want: bondBound_ports.StatusCount{Total: 100, Count: 1}},
		{name: "email", search: "lucia@", want: bondBound_ports.StatusCount{Total: 100, Count: 1}},
		{name: "producto sin el bono sin dueño", search: "plus", want: bondBound_ports.StatusCount{Total: 1000, Count: 1}},
		{name: "regex inválida", search: "a.*(", want: bondBound_ports.StatusCount{Total: 5, Count: 1}},
		{name: "sin coincidencias", search: "zzz", want: bondBound_ports.StatusCount{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count, err := repo.CountByStatus(context.Background(), agencyScope(agencyA), bondBound_domain.KpiFilters{Search: tt.search}, -5)
			if err != nil {
				t.Fatalf("CountByStatus(%q): %v", tt.search, err)
			}

			if count != tt.want {
				t.Errorf("búsqueda %q: CountByStatus = %+v, se esperaba %+v", tt.search, count, tt.want)
			}
		})
	}
}
