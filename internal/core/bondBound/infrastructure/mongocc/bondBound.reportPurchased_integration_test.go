//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Filtros ya normalizados del reporte: los iniciados, que es el estado por defecto, y todos los estados.
var (
	reportInitiated = bondBound_domain.ReportFilters{StatusID: bondBound_domain.StatusInitiated}
	reportAll       = bondBound_domain.ReportFilters{StatusID: bondBound_domain.ReportAllStatuses}
)

// El reporte trae solo los bonos no borrados de la agencia pedida. Por defecto son los iniciados y con '0'
// todos los estados, por createdAt descendente y con el estado por nombre; los _id Meteor y UUID salen por
// igual. Cada fila trae lo que recibe ms-exporter: la clave, los montos, las fechas, el nombre del dueño y los
// nombres del producto y del tipo. CA-07.1, CA-07.8
func TestReportPurchasedRowsStatusAndScope(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	initiatedUUID := boundAt(boundID, 0, nil)
	initiatedLegacy := boundAt(legacyBoundID, 1, bson.M{"userId": user2ID, "bondObjectId": legacyObjectID, "bondTypeId": legacyTypeID})
	matured := boundAt(legacyBound2ID, 2, bson.M{"statusId": "2"})
	insert(t, boundCollection, initiatedUUID, initiatedLegacy, matured,
		boundAt(nextBoundID(), 3, bson.M{"agencyId": agencyB}),
		boundAt(nextBoundID(), 4, bson.M{"isRemove": true}))

	t.Run("iniciados", func(t *testing.T) {
		rows, err := repo.ReportPurchasedRows(context.Background(), agencyA, reportInitiated, -5)
		if err != nil {
			t.Fatalf("ReportPurchasedRows: %v", err)
		}

		if got, want := reportKeys(rows), keys(initiatedLegacy, initiatedUUID); !slices.Equal(got, want) {
			t.Fatalf("bonos %v, se esperaban %v", got, want)
		}
		assertJSON(t, rows[1], bondBound_ports.PurchasedReportRow{
			Key:                       "BP12-0",
			Status:                    "Iniciado",
			Amount:                    1000,
			MaturityTimeInDays:        365,
			InitialDate:               fixtureAt,
			MaturityDate:              fixtureAt.AddDate(0, 0, 365),
			MaturityRewardsPercentage: 15,
			Profile:                   &bondBound_ports.ProfileName{FirstName: "Carlos", LastName: "Ramírez"},
			BondType:                  bondBound_ports.BondTypeRef{Name: "Plus"},
			BondObject:                bondBound_ports.BondObjectRef{Name: "Bono Plus 12M"},
			CreatedAt:                 fixtureAt,
		})
		if rows[0].Status != "Iniciado" || rows[0].BondObject.Name != "Bono Clásico 6M" || rows[0].BondType.Name != "Clásico" ||
			rows[0].Profile == nil || rows[0].Profile.FirstName != "Lucía" {
			t.Errorf("fila legacy %+v, se esperaba el bono iniciado de Lucía del producto y el tipo Clásico", rows[0])
		}
	})

	t.Run("todos los estados", func(t *testing.T) {
		rows, err := repo.ReportPurchasedRows(context.Background(), agencyA, reportAll, -5)
		if err != nil {
			t.Fatalf("ReportPurchasedRows: %v", err)
		}

		if got, want := reportKeys(rows), keys(matured, initiatedLegacy, initiatedUUID); !slices.Equal(got, want) {
			t.Fatalf("bonos %v, se esperaban %v", got, want)
		}
		statuses := []string{rows[0].Status, rows[1].Status, rows[2].Status}
		if want := []string{"Madurado", "Iniciado", "Iniciado"}; !slices.Equal(statuses, want) {
			t.Errorf("estados %v, se esperaban %v", statuses, want)
		}
	})
}

// El dueño es opcional: el bono sin perfil sale sin profile en el JSON. El producto y el tipo son
// obligatorios: el bono sin alguno de los dos no sale. CA-07.1
func TestReportPurchasedRowsJoins(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	withProfile := boundAt(nextBoundID(), 0, nil)
	noProfile := boundAt(nextBoundID(), 1, bson.M{"userId": shared_domain.NewID()})
	insert(t, boundCollection, withProfile, noProfile,
		boundAt(nextBoundID(), 2, bson.M{"bondObjectId": missingID}),
		boundAt(nextBoundID(), 3, bson.M{"bondTypeId": missingID}))

	rows, err := repo.ReportPurchasedRows(context.Background(), agencyA, reportInitiated, -5)
	if err != nil {
		t.Fatalf("ReportPurchasedRows: %v", err)
	}

	if got, want := reportKeys(rows), keys(noProfile, withProfile); !slices.Equal(got, want) {
		t.Fatalf("bonos %v, se esperaban %v", got, want)
	}
	raw, err := json.Marshal(rows[0])
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var row map[string]any
	if err := json.Unmarshal(raw, &row); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if _, ok := row["profile"]; ok || rows[0].Profile != nil {
		t.Errorf("el bono sin perfil trae profile: %s", raw)
	}
}

// userId filtra por igualdad exacta, sea un id Meteor o un UUID, y trae los bonos del usuario con _id de
// los dos formatos. CA-07.8
func TestReportPurchasedRowsUserFilter(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	legacyUserLegacyBound := boundAt(legacyBoundID, 0, nil)
	legacyUserUUIDBound := boundAt(boundID, 1, nil)
	uuidUserLegacyBound := boundAt(legacyBound2ID, 2, bson.M{"userId": user2ID})
	uuidUserUUIDBound := boundAt(shared_domain.NewID(), 3, bson.M{"userId": user2ID})
	insert(t, boundCollection, legacyUserLegacyBound, legacyUserUUIDBound, uuidUserLegacyBound, uuidUserUUIDBound)

	tests := []struct {
		name   string
		userID string
		want   []string
	}{
		{name: "usuario Meteor", userID: userID, want: keys(legacyUserUUIDBound, legacyUserLegacyBound)},
		{name: "usuario UUID", userID: user2ID, want: keys(uuidUserUUIDBound, uuidUserLegacyBound)},
		{name: "usuario sin bonos", userID: missingID, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := reportInitiated
			f.UserID = tt.userID

			rows, err := repo.ReportPurchasedRows(context.Background(), agencyA, f, -5)
			if err != nil {
				t.Fatalf("ReportPurchasedRows: %v", err)
			}

			if got := reportKeys(rows); !slices.Equal(got, tt.want) {
				t.Errorf("bonos %v, se esperaban %v", got, tt.want)
			}
		})
	}
}

// La búsqueda no distingue mayúsculas y encuentra por el nombre del dueño, el nombre del producto y el nombre
// del tipo; por paridad no busca en el apellido, el email ni las claves. El texto se busca literal: "a.*(" no
// rompe la agregación y no se interpreta como regex. CA-07.2
func TestReportPurchasedRowsSearch(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	const regexObj = "ObjRegex000000001"
	insert(t, objectCollection, objectDoc(regexObj, "Bono a.*( Especial", "BE01", legacyTypeID, nil))
	bound := func(minute int, owner, object, bondType string) bson.M {
		return boundAt(nextBoundID(), minute, bson.M{"userId": owner, "bondObjectId": object, "bondTypeId": bondType})
	}
	byObject := bound(0, user2ID, objectID, legacyTypeID)
	byType := bound(1, user2ID, legacyObjectID, typeID)
	byOwner := bound(2, userID, legacyObjectID, legacyTypeID)
	byRegex := bound(3, user2ID, regexObj, legacyTypeID)
	insert(t, boundCollection, byObject, byType, byOwner, byRegex)

	tests := []struct {
		name   string
		search string
		want   []string
	}{
		{name: "plus en el producto y en el tipo", search: "plus", want: keys(byType, byObject)},
		{name: "PLUS en mayúsculas", search: "PLUS", want: keys(byType, byObject)},
		{name: "nombre del dueño", search: "carlos", want: keys(byOwner)},
		{name: "nombre del producto", search: "clásico 6m", want: keys(byOwner, byType)},
		{name: "apellido", search: "ramírez", want: []string{}},
		{name: "email", search: "lucia@futurapps.com", want: []string{}},
		{name: "clave", search: "BP12", want: []string{}},
		{name: "regex inválida", search: "a.*(", want: keys(byRegex)},
		{name: "regex que coincide con todo", search: "a.*", want: keys(byRegex)},
		{name: "sin coincidencias", search: "zzz", want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := reportInitiated
			f.Search = tt.search

			rows, err := repo.ReportPurchasedRows(context.Background(), agencyA, f, -5)
			if err != nil {
				t.Fatalf("ReportPurchasedRows(%q): %v", tt.search, err)
			}

			if got := reportKeys(rows); !slices.Equal(got, tt.want) {
				t.Errorf("búsqueda %q: bonos %v, se esperaban %v", tt.search, got, tt.want)
			}
		})
	}
}

// El filtro aMonth toma el mes en la zona pedida: en UTC-5, una compra del 2026-11-01T03:00Z es de octubre y
// una del 2026-10-01T03:00Z no; en UTC es al revés. Un filtro inválido es ErrInvalidDateFilter. CA-07.2
func TestReportPurchasedRowsDateFilter(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	lateOctober := boundAt(nextBoundID(), 0, bson.M{"createdAt": createdAt(t, "2026-11-01T03:00:00Z")})
	lateSeptember := boundAt(nextBoundID(), 1, bson.M{"createdAt": createdAt(t, "2026-10-01T03:00:00Z")})
	midOctober := boundAt(nextBoundID(), 2, bson.M{"createdAt": createdAt(t, "2026-10-15T12:00:00Z")})
	insert(t, boundCollection, lateOctober, lateSeptember, midOctober)
	october := reportInitiated
	october.Date = bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAMonth, Month: 10, Year: 2026}

	tests := []struct {
		name    string
		utcDiff int
		want    []string
	}{
		{name: "UTC-5", utcDiff: -5, want: keys(lateOctober, midOctober)},
		{name: "UTC", utcDiff: 0, want: keys(midOctober, lateSeptember)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := repo.ReportPurchasedRows(context.Background(), agencyA, october, tt.utcDiff)
			if err != nil {
				t.Fatalf("ReportPurchasedRows: %v", err)
			}

			if got := reportKeys(rows); !slices.Equal(got, tt.want) {
				t.Errorf("octubre en %s: bonos %v, se esperaban %v", tt.name, got, tt.want)
			}
		})
	}

	t.Run("filtro inválido", func(t *testing.T) {
		invalid := reportInitiated
		invalid.Date = bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: "2026-10-14", EndDate: "2026-10-13"}

		_, err := repo.ReportPurchasedRows(context.Background(), agencyA, invalid, -5)

		if !errors.Is(err, bondBound_domain.ErrInvalidDateFilter) {
			t.Errorf("ReportPurchasedRows = %v, se esperaba ErrInvalidDateFilter", err)
		}
	})
}
