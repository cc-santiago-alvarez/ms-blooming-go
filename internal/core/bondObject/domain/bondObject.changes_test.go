package bondObject_domain_test

import (
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

const (
	agencyA     = "tDr6NwSsDe4Lnccp8"
	agencyB     = "0b6f8c2e-3d4a-4e5b-9c6d-7e8f9a0b1c2d"
	agencyC     = "Ar8Kd2Lp5Qw9Zx3Cv"
	userID      = "Hn4Wq8Ls2Dx6Vb9Ty"
	typeID      = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	otherTypeID = "kQ3xT9mPz2LwA7bNc"
)

var (
	// storedAt es la fecha de alta del producto guardado.
	storedAt = time.Date(2026, 1, 10, 15, 0, 0, 0, time.UTC)
	// saleInit es el inicio de la ventana de venta del producto guardado.
	saleInit = time.Date(2026, 11, 1, 5, 0, 0, 0, time.UTC)
	// saleEnd es un fin de ventana de venta que el producto guardado no tiene.
	saleEnd = time.Date(2027, 1, 31, 5, 0, 0, 0, time.UTC)
)

// stored es el producto antes de la edición; cada llamada devuelve listas y fechas propias.
func stored() bondObject_domain.BondObject {
	start := saleInit
	return bondObject_domain.BondObject{
		ID:                        "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f",
		Name:                      "Bono Plus 12M",
		Description:               "Renta fija 12 meses",
		AgencyID:                  shared_domain.AgencyIDs{agencyA, agencyB},
		Key:                       "BP12",
		BondTypeID:                typeID,
		Amount:                    1000,
		AuthorizedUserIDs:         []string{userID},
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		Active:                    true,
		Available:                 true,
		SaleInitDate:              &start,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPenaltyPercentage: 100,
		},
		CreatedAt: storedAt,
		UpdatedAt: storedAt,
	}
}

// sameValue compara las fechas por instante y el resto con DeepEqual, que distingue [] de nil y un nil
// sin tipo de un puntero nil.
func sameValue(a, b any) bool {
	if ta, ok := a.(time.Time); ok {
		tb, ok := b.(time.Time)
		return ok && ta.Equal(tb)
	}
	return reflect.DeepEqual(a, b)
}

// change es el cambio de un campo de old a updated.
func change(old, updated any) bondObject_domain.Change {
	return bondObject_domain.Change{Old: old, New: updated}
}

// BuildChanges registra cada campo editado con su valor anterior y el nuevo: los escalares, las fechas por
// instante, agencyId y authorizedUserIds como conjuntos y earlyCashTerm campo por campo; editedProperties
// son esas claves ordenadas. Un cambio de amount 1000→2000 y de earlyCashTerm.inflectionPoint 0→30 da
// exactamente esas dos claves. CA-02.6
func TestBuildChanges(t *testing.T) {
	tests := []struct {
		name string
		// prepare, si está, se aplica al producto antes y después de la edición.
		prepare func(o *bondObject_domain.BondObject)
		edit    func(o *bondObject_domain.BondObject)
		want    map[string]bondObject_domain.Change
		// edited es el orden esperado de editedProperties; si falta, son las claves de want ordenadas.
		edited []string
	}{
		{
			name: "amount e inflectionPoint",
			edit: func(o *bondObject_domain.BondObject) {
				o.Amount = 2000
				o.EarlyCashTerm.InflectionPoint = 30
			},
			want: map[string]bondObject_domain.Change{
				"amount":                        change(1000.0, 2000.0),
				"earlyCashTerm.inflectionPoint": change(0, 30),
			},
			edited: []string{"amount", "earlyCashTerm.inflectionPoint"},
		},
		{
			name: "sin cambios",
			edit: func(*bondObject_domain.BondObject) {},
			want: map[string]bondObject_domain.Change{},
		},
		{
			name: "_id, createdAt, updatedAt e isRemove no cuentan",
			edit: func(o *bondObject_domain.BondObject) {
				o.ID = "1b2c3d4e-5f6a-4b7c-9d8e-0f1a2b3c4d5e"
				o.CreatedAt = storedAt.AddDate(0, 0, 1)
				o.UpdatedAt = storedAt.AddDate(0, 1, 0)
				o.IsRemove = true
			},
			want: map[string]bondObject_domain.Change{},
		},

		// escalares
		{
			name: "name",
			edit: func(o *bondObject_domain.BondObject) { o.Name = "Bono Max" },
			want: map[string]bondObject_domain.Change{"name": change("Bono Plus 12M", "Bono Max")},
		},
		{
			name: "description",
			edit: func(o *bondObject_domain.BondObject) { o.Description = "Renta fija 24 meses" },
			want: map[string]bondObject_domain.Change{"description": change("Renta fija 12 meses", "Renta fija 24 meses")},
		},
		{
			name: "key",
			edit: func(o *bondObject_domain.BondObject) { o.Key = "BP24" },
			want: map[string]bondObject_domain.Change{"key": change("BP12", "BP24")},
		},
		{
			name: "bondTypeId",
			edit: func(o *bondObject_domain.BondObject) { o.BondTypeID = otherTypeID },
			want: map[string]bondObject_domain.Change{"bondTypeId": change(typeID, otherTypeID)},
		},
		{
			name: "isPrivateBond",
			edit: func(o *bondObject_domain.BondObject) { o.IsPrivateBond = true },
			want: map[string]bondObject_domain.Change{"isPrivateBond": change(false, true)},
		},
		{
			name: "maturityTimeInDays",
			edit: func(o *bondObject_domain.BondObject) { o.MaturityTimeInDays = 180 },
			want: map[string]bondObject_domain.Change{"maturityTimeInDays": change(365, 180)},
		},
		{
			name: "maturityRewardsPercentage",
			edit: func(o *bondObject_domain.BondObject) { o.MaturityRewardsPercentage = 12.5 },
			want: map[string]bondObject_domain.Change{"maturityRewardsPercentage": change(15.0, 12.5)},
		},
		{
			name: "active",
			edit: func(o *bondObject_domain.BondObject) { o.Active = false },
			want: map[string]bondObject_domain.Change{"active": change(true, false)},
		},
		{
			name: "available",
			edit: func(o *bondObject_domain.BondObject) { o.Available = false },
			want: map[string]bondObject_domain.Change{"available": change(true, false)},
		},

		// earlyCashTerm
		{
			name: "earlyCashTerm.initialPenaltyPercentage",
			edit: func(o *bondObject_domain.BondObject) { o.EarlyCashTerm.InitialPenaltyPercentage = 80 },
			want: map[string]bondObject_domain.Change{"earlyCashTerm.initialPenaltyPercentage": change(100.0, 80.0)},
		},
		{
			name: "earlyCashTerm.inflectionPenaltyPercentage",
			edit: func(o *bondObject_domain.BondObject) { o.EarlyCashTerm.InflectionPenaltyPercentage = 75 },
			want: map[string]bondObject_domain.Change{"earlyCashTerm.inflectionPenaltyPercentage": change(100.0, 75.0)},
		},
		{
			name: "earlyCashTerm.blockWithdrawals",
			edit: func(o *bondObject_domain.BondObject) { o.EarlyCashTerm.BlockWithdrawals = true },
			want: map[string]bondObject_domain.Change{"earlyCashTerm.blockWithdrawals": change(false, true)},
		},
		{
			name: "earlyCashTerm.initialDayForWithdraw",
			edit: func(o *bondObject_domain.BondObject) { o.EarlyCashTerm.InitialDayForWithdraw = 30 },
			want: map[string]bondObject_domain.Change{"earlyCashTerm.initialDayForWithdraw": change(0, 30)},
		},

		// fechas
		{
			name: "mismo instante en otra zona",
			edit: func(o *bondObject_domain.BondObject) {
				local := saleInit.In(time.FixedZone("", -5*3600))
				o.SaleInitDate = &local
			},
			want: map[string]bondObject_domain.Change{},
		},
		{
			name: "otro instante",
			edit: func(o *bondObject_domain.BondObject) {
				later := saleInit.Add(24 * time.Hour)
				o.SaleInitDate = &later
			},
			want: map[string]bondObject_domain.Change{"saleInitDate": change(saleInit, saleInit.Add(24*time.Hour))},
		},
		{
			name: "fecha que se agrega",
			edit: func(o *bondObject_domain.BondObject) {
				end := saleEnd
				o.SaleEndDate = &end
			},
			want: map[string]bondObject_domain.Change{"saleEndDate": change(nil, saleEnd)},
		},
		{
			name: "fecha que se quita",
			edit: func(o *bondObject_domain.BondObject) { o.SaleInitDate = nil },
			want: map[string]bondObject_domain.Change{"saleInitDate": change(saleInit, nil)},
		},

		// listas
		{
			name: "agencyId reordenado y con repetidos",
			edit: func(o *bondObject_domain.BondObject) {
				o.AgencyID = shared_domain.AgencyIDs{agencyB, agencyA, agencyB}
			},
			want: map[string]bondObject_domain.Change{},
		},
		{
			name: "agencyId con una agencia más",
			edit: func(o *bondObject_domain.BondObject) {
				o.AgencyID = shared_domain.AgencyIDs{agencyA, agencyB, agencyC}
			},
			want: map[string]bondObject_domain.Change{
				"agencyId": change([]string{agencyA, agencyB}, []string{agencyA, agencyB, agencyC}),
			},
		},
		{
			name:    "authorizedUserIds nil y vacío son iguales",
			prepare: func(o *bondObject_domain.BondObject) { o.AuthorizedUserIDs = nil },
			edit:    func(o *bondObject_domain.BondObject) { o.AuthorizedUserIDs = []string{} },
			want:    map[string]bondObject_domain.Change{},
		},
		{
			name: "authorizedUserIds que se vacía",
			edit: func(o *bondObject_domain.BondObject) { o.AuthorizedUserIDs = nil },
			want: map[string]bondObject_domain.Change{"authorizedUserIds": change([]string{userID}, []string{})},
		},

		{
			name: "varios cambios a la vez",
			edit: func(o *bondObject_domain.BondObject) {
				end := saleEnd
				o.Name = "Bono Max"
				o.Amount = 1500
				o.SaleEndDate = &end
				o.AgencyID = shared_domain.AgencyIDs{agencyA, agencyB, agencyC}
				o.EarlyCashTerm.BlockWithdrawals = true
			},
			want: map[string]bondObject_domain.Change{
				"name":                           change("Bono Plus 12M", "Bono Max"),
				"amount":                         change(1000.0, 1500.0),
				"saleEndDate":                    change(nil, saleEnd),
				"agencyId":                       change([]string{agencyA, agencyB}, []string{agencyA, agencyB, agencyC}),
				"earlyCashTerm.blockWithdrawals": change(false, true),
			},
			edited: []string{"agencyId", "amount", "earlyCashTerm.blockWithdrawals", "name", "saleEndDate"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			old, updated := stored(), stored()
			if tt.prepare != nil {
				tt.prepare(&old)
				tt.prepare(&updated)
			}
			tt.edit(&updated)

			changes, edited := bondObject_domain.BuildChanges(old, updated)

			if changes == nil {
				t.Fatalf("changes es nil, se esperaba un mapa")
			}
			if got, want := slices.Sorted(maps.Keys(changes)), slices.Sorted(maps.Keys(tt.want)); !slices.Equal(got, want) {
				t.Fatalf("changes en %v, se esperaba en %v", got, want)
			}
			for field, w := range tt.want {
				c := changes[field]
				if !sameValue(c.Old, w.Old) || !sameValue(c.New, w.New) {
					t.Errorf("changes[%s] = {%#v, %#v}, se esperaba {%#v, %#v}", field, c.Old, c.New, w.Old, w.New)
				}
			}

			wantEdited := tt.edited
			if wantEdited == nil {
				wantEdited = slices.Sorted(maps.Keys(tt.want))
			}
			if !slices.Equal(edited, wantEdited) {
				t.Errorf("editedProperties %v, se esperaba %v", edited, wantEdited)
			}
		})
	}
}
