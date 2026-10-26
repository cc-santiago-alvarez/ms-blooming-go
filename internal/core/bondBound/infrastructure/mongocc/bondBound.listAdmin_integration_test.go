//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Con 25 bonos de la agencia A, la página 2 de 10 trae los bonos 11 a 20 por createdAt descendente, con
// total 25 y 3 páginas, su dueño, su producto y su tipo; los de la agencia B y los borrados no aparecen, y
// sin alcance de agencia se suman los de B. CA-06.1
func TestListAdminPageAndScope(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	var inA []bson.M
	for i := range 25 {
		doc := boundAt(nextBoundID(), i, nil)
		if i%2 == 1 {
			doc["bondObjectId"], doc["bondTypeId"], doc["userId"] = legacyObjectID, legacyTypeID, user2ID
		}
		inA = append(inA, doc)
	}
	inB := []bson.M{
		boundAt(nextBoundID(), 30, bson.M{"agencyId": agencyB}),
		boundAt(nextBoundID(), 31, bson.M{"agencyId": agencyB}),
		boundAt(nextBoundID(), 32, bson.M{"agencyId": agencyB}),
	}
	insert(t, boundCollection, inA...)
	insert(t, boundCollection, inB...)
	insert(t, boundCollection, boundAt(nextBoundID(), 40, bson.M{"isRemove": true}))

	page, err := repo.ListAdmin(context.Background(), agencyScope(agencyA), bondBound_domain.ListFilters{}, shared_domain.Pagination{Page: 2, Limit: 10}, -5)
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}

	var want []string
	for i := 14; i >= 5; i-- {
		want = append(want, inA[i]["_id"].(string))
	}
	if got := adminRowIDs(page.Items); !slices.Equal(got, want) {
		t.Errorf("bonos %v, se esperaban %v", got, want)
	}
	if page.Total != 25 || page.Page != 2 || page.Limit != 10 || page.TotalPages != 3 {
		t.Errorf("total %d, page %d, limit %d y totalPages %d, se esperaban 25, 2, 10 y 3",
			page.Total, page.Page, page.Limit, page.TotalPages)
	}
	for _, row := range page.Items {
		wantObject, wantType, wantOwner := "Bono Plus 12M", "Plus", "Carlos"
		if row.Bound.UserID == user2ID {
			wantObject, wantType, wantOwner = "Bono Clásico 6M", "Clásico", "Lucía"
		}
		if row.BondObject.Name != wantObject || row.BondType.Name != wantType || row.Profile == nil || row.Profile.FirstName != wantOwner {
			t.Errorf("bono %s con producto %q, tipo %q y dueño %+v, se esperaban %q, %q y %q",
				row.Bound.ID, row.BondObject.Name, row.BondType.Name, row.Profile, wantObject, wantType, wantOwner)
		}
	}

	all, err := repo.ListAdmin(context.Background(), nil, bondBound_domain.ListFilters{}, shared_domain.Pagination{Page: 1, Limit: 100}, -5)
	if err != nil {
		t.Fatalf("ListAdmin sin alcance: %v", err)
	}
	if all.Total != 28 || !slices.Equal(adminRowIDs(all.Items)[:3], []string{inB[2]["_id"].(string), inB[1]["_id"].(string), inB[0]["_id"].(string)}) {
		t.Errorf("sin alcance: total %d y primeros bonos %v, se esperaban 28 empezando por los de la agencia B",
			all.Total, adminRowIDs(all.Items)[:3])
	}
}

// El dueño es opcional: el bono sin perfil sale sin profile. El producto y el tipo son obligatorios: el bono
// sin alguno de los dos no sale. CA-06.1
func TestListAdminJoins(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	withProfile := boundAt(nextBoundID(), 0, nil)
	noProfile := boundAt(nextBoundID(), 1, bson.M{"userId": shared_domain.NewID()})
	noObject := boundAt(nextBoundID(), 2, bson.M{"bondObjectId": missingID})
	noType := boundAt(nextBoundID(), 3, bson.M{"bondTypeId": missingID})
	insert(t, boundCollection, withProfile, noProfile, noObject, noType)

	page, err := repo.ListAdmin(context.Background(), agencyScope(agencyA), bondBound_domain.ListFilters{}, shared_domain.Pagination{Page: 1, Limit: 10}, -5)
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}

	if got, want := adminRowIDs(page.Items), ids(noProfile, withProfile); !slices.Equal(got, want) || page.Total != 2 {
		t.Fatalf("bonos %v con total %d, se esperaban %v con total 2", got, page.Total, want)
	}
	if page.Items[0].Profile != nil {
		t.Errorf("el bono sin perfil trae profile %+v", page.Items[0].Profile)
	}
	if want := (bondBound_ports.ProfileName{FirstName: "Carlos", LastName: "Ramírez"}); page.Items[1].Profile == nil || *page.Items[1].Profile != want {
		t.Errorf("profile %+v, se esperaba %+v", page.Items[1].Profile, want)
	}
}

// El estado y el usuario filtran por igualdad exacta. CA-06.1
func TestListAdminFilters(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	initiated := boundAt(nextBoundID(), 0, nil)
	matured := boundAt(nextBoundID(), 1, bson.M{"statusId": "2"})
	otherMatured := boundAt(nextBoundID(), 2, bson.M{"statusId": "2", "userId": user2ID})
	insert(t, boundCollection, initiated, matured, otherMatured)

	tests := []struct {
		name    string
		filters bondBound_domain.ListFilters
		want    []string
	}{
		{name: "estado", filters: bondBound_domain.ListFilters{StatusID: bondBound_domain.StatusFinalized}, want: ids(otherMatured, matured)},
		{name: "usuario", filters: bondBound_domain.ListFilters{UserID: userID}, want: ids(matured, initiated)},
		{name: "estado y usuario", filters: bondBound_domain.ListFilters{StatusID: bondBound_domain.StatusFinalized, UserID: user2ID}, want: ids(otherMatured)},
		{name: "estado sin bonos", filters: bondBound_domain.ListFilters{StatusID: bondBound_domain.StatusSold}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := repo.ListAdmin(context.Background(), agencyScope(agencyA), tt.filters, shared_domain.Pagination{Page: 1, Limit: 10}, -5)
			if err != nil {
				t.Fatalf("ListAdmin: %v", err)
			}

			if got := adminRowIDs(page.Items); !slices.Equal(got, tt.want) || page.Total != int64(len(tt.want)) {
				t.Errorf("bonos %v con total %d, se esperaban %v", got, page.Total, tt.want)
			}
		})
	}
}

// La búsqueda no distingue mayúsculas y encuentra por la cédula, el nombre, el apellido y el email del dueño,
// la clave, el nombre y la descripción del producto y el nombre del tipo. El texto se busca literal: "a.*("
// no rompe la agregación y no se interpreta como regex. CA-06.2
func TestListAdminSearch(t *testing.T) {
	repo := setup(t)
	const (
		juanID   = "PrfJuan0000000001"
		plainTyp = "TypPlain000000001"
		plainObj = "ObjPlain000000001"
		regexObj = "ObjRegex000000001"
	)
	mailID, noneID := shared_domain.NewID(), shared_domain.NewID()
	juanTyp, keyObj := shared_domain.NewID(), shared_domain.NewID()
	insert(t, profileCollection,
		profileDoc(juanID, "1111111111", "Juan", "Pereira", "jp@futurapps.com"),
		profileDoc(mailID, "2222222222", "Ana", "Gómez", "ana.juanes@futurapps.com"),
		profileDoc(noneID, "3333333333", "Luis", "Mora", "luis@futurapps.com"))
	insert(t, typeCollection, typeDoc(plainTyp, "Plus", nil), typeDoc(juanTyp, "Juanita Premium", nil))
	insert(t, objectCollection,
		objectDoc(plainObj, "Bono Plus 12M", "BP12", plainTyp, bson.M{"description": "Renta fija 12 meses"}),
		objectDoc(keyObj, "Bono Mensual", "JUAN24", plainTyp, bson.M{"description": "Producto mensual"}),
		objectDoc(regexObj, "Bono Especial", "BE01", plainTyp, bson.M{"description": "Plan a.*( especial"}))
	bound := func(minute int, owner, object, bondType string) bson.M {
		return boundAt(nextBoundID(), minute, bson.M{"userId": owner, "bondObjectId": object, "bondTypeId": bondType})
	}
	byFirstName := bound(0, juanID, plainObj, plainTyp)
	byEmail := bound(1, mailID, plainObj, plainTyp)
	byKey := bound(2, noneID, keyObj, plainTyp)
	byType := bound(3, noneID, plainObj, juanTyp)
	byRegex := bound(4, noneID, regexObj, plainTyp)
	insert(t, boundCollection, byFirstName, byEmail, byKey, byType, byRegex)

	tests := []struct {
		name   string
		search string
		want   []string
	}{
		{name: "JUAN en nombre, email, clave y tipo", search: "JUAN", want: ids(byType, byKey, byEmail, byFirstName)},
		{name: "juan en minúsculas", search: "juan", want: ids(byType, byKey, byEmail, byFirstName)},
		{name: "cédula", search: "1111111111", want: ids(byFirstName)},
		{name: "apellido", search: "PEREIRA", want: ids(byFirstName)},
		{name: "nombre del producto", search: "bono mensual", want: ids(byKey)},
		{name: "descripción del producto", search: "renta FIJA", want: ids(byType, byEmail, byFirstName)},
		{name: "regex inválida", search: "a.*(", want: ids(byRegex)},
		{name: "regex que coincide con todo", search: "a.*", want: ids(byRegex)},
		{name: "paréntesis suelto", search: "(", want: ids(byRegex)},
		{name: "sin coincidencias", search: "zzz", want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page, err := repo.ListAdmin(context.Background(), nil, bondBound_domain.ListFilters{Search: tt.search}, shared_domain.Pagination{Page: 1, Limit: 10}, -5)
			if err != nil {
				t.Fatalf("ListAdmin(%q): %v", tt.search, err)
			}

			if got := adminRowIDs(page.Items); !slices.Equal(got, tt.want) || page.Total != int64(len(tt.want)) {
				t.Errorf("búsqueda %q: bonos %v con total %d, se esperaban %v", tt.search, got, page.Total, tt.want)
			}
		})
	}
}

// El filtro aMonth toma el mes en la zona de la sesión: en UTC-5, una compra del 31 de octubre a las 22:00
// (1 de noviembre en UTC) es de octubre y una del 30 de septiembre a las 22:00 no. El mes y el año de cada
// fila también salen en la zona de la sesión. Un filtro inválido es ErrInvalidDateFilter. CA-06.3, CA-06.1
func TestListAdminDateFilterAndMonth(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	at := func(s string) bson.M {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("fecha de prueba inválida %q: %v", s, err)
		}
		return boundAt(nextBoundID(), 0, bson.M{"createdAt": v})
	}
	lateOctober := at("2026-11-01T03:00:00Z")
	lateSeptember := at("2026-10-01T03:00:00Z")
	midOctober := at("2026-10-15T12:00:00Z")
	newYearEve := at("2027-01-01T03:00:00Z")
	insert(t, boundCollection, lateOctober, lateSeptember, midOctober, newYearEve)
	october := bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAMonth, Month: 10, Year: 2026}

	page, err := repo.ListAdmin(context.Background(), nil, bondBound_domain.ListFilters{Date: october}, shared_domain.Pagination{Page: 1, Limit: 10}, -5)
	if err != nil {
		t.Fatalf("ListAdmin: %v", err)
	}
	if got, want := adminRowIDs(page.Items), ids(lateOctober, midOctober); !slices.Equal(got, want) {
		t.Errorf("octubre en UTC-5: bonos %v, se esperaban %v", got, want)
	}

	type monthYear struct{ month, year int }
	zones := []struct {
		name    string
		utcDiff int
		want    map[string]monthYear
	}{
		{name: "UTC-5", utcDiff: -5, want: map[string]monthYear{
			lateOctober["_id"].(string):   {10, 2026},
			lateSeptember["_id"].(string): {9, 2026},
			midOctober["_id"].(string):    {10, 2026},
			newYearEve["_id"].(string):    {12, 2026},
		}},
		{name: "UTC", utcDiff: 0, want: map[string]monthYear{
			lateOctober["_id"].(string):   {11, 2026},
			lateSeptember["_id"].(string): {10, 2026},
			midOctober["_id"].(string):    {10, 2026},
			newYearEve["_id"].(string):    {1, 2027},
		}},
	}
	for _, z := range zones {
		t.Run(z.name, func(t *testing.T) {
			page, err := repo.ListAdmin(context.Background(), nil, bondBound_domain.ListFilters{}, shared_domain.Pagination{Page: 1, Limit: 10}, z.utcDiff)
			if err != nil {
				t.Fatalf("ListAdmin: %v", err)
			}

			if len(page.Items) != len(z.want) {
				t.Fatalf("%d bonos, se esperaban %d", len(page.Items), len(z.want))
			}
			for _, row := range page.Items {
				if got := (monthYear{row.Month, row.Year}); got != z.want[row.Bound.ID] {
					t.Errorf("bono creado %v: mes y año %v, se esperaban %v", row.Bound.CreatedAt, got, z.want[row.Bound.ID])
				}
			}
		})
	}

	t.Run("filtro inválido", func(t *testing.T) {
		invalid := bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: "2026-10-14", EndDate: "2026-10-13"}

		_, err := repo.ListAdmin(context.Background(), nil, bondBound_domain.ListFilters{Date: invalid}, shared_domain.Pagination{Page: 1, Limit: 10}, -5)

		if !errors.Is(err, bondBound_domain.ErrInvalidDateFilter) {
			t.Errorf("ListAdmin = %v, se esperaba ErrInvalidDateFilter", err)
		}
	})
}
