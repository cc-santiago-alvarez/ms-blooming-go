package bondObject_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Una edición que cambia amount 1000→2000 y earlyCashTerm.inflectionPoint 0→30 registra el history
// BOND_UPDATED con changes y editedProperties solo en esas dos claves; sobre un producto borrado es
// ErrBondObjectNotFound y no se registra nada. CA-02.6
func TestUpdate(t *testing.T) {
	t.Run("cambia amount e inflectionPoint", func(t *testing.T) {
		stored := storedObject(objectID)
		f := newFakes(stored)
		params := paramsFrom(stored)
		params.Amount = 2000
		params.EarlyCashTerm.InflectionPoint = 30

		got, err := f.app().Update(withAdmin(agencyMain, true), objectID, params)

		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if got.Amount != 2000 || got.EarlyCashTerm.InflectionPoint != 30 {
			t.Errorf("Update devolvió amount %v e inflectionPoint %d, se esperaban 2000 y 30", got.Amount, got.EarlyCashTerm.InflectionPoint)
		}
		if got.ID != objectID || !got.CreatedAt.Equal(storedAt) || !got.UpdatedAt.After(storedAt) {
			t.Errorf("Update devolvió _id %s, createdAt %v y updatedAt %v; se esperaba el mismo _id y createdAt y un updatedAt posterior",
				got.ID, got.CreatedAt, got.UpdatedAt)
		}
		if len(f.repo.updated) != 1 {
			t.Fatalf("UpdateOne se llamó %d veces, se esperaba 1", len(f.repo.updated))
		}

		if len(f.history.added) != 1 {
			t.Fatalf("History.Add se llamó %d veces, se esperaba 1", len(f.history.added))
		}
		h := f.history.added[0]
		if h.TypeID != bondHistory_domain.HistoryBond || h.Text != shared_domain.TextBondUpdated || h.RelatedID != objectID {
			t.Errorf("History.Add recibió typeId %q, text %q y relatedId %q; se esperaba %q, %q y %s",
				h.TypeID, h.Text, h.RelatedID, bondHistory_domain.HistoryBond, shared_domain.TextBondUpdated, objectID)
		}
		wantChanges := map[string]bondHistory_domain.ChangeEntry{
			"amount":                        {Old: 1000.0, New: 2000.0},
			"earlyCashTerm.inflectionPoint": {Old: 0, New: 30},
		}
		if len(h.Changes) != len(wantChanges) {
			t.Errorf("changes %v, se esperaba %v", h.Changes, wantChanges)
		}
		for field, w := range wantChanges {
			if h.Changes[field] != w {
				t.Errorf("changes[%s] = %+v, se esperaba %+v", field, h.Changes[field], w)
			}
		}
		if want := []string{"amount", "earlyCashTerm.inflectionPoint"}; !slices.Equal(h.EditedProperties, want) {
			t.Errorf("editedProperties %v, se esperaba %v", h.EditedProperties, want)
		}
	})

	t.Run("borrado", func(t *testing.T) {
		removed := storedObject(objectID)
		removed.IsRemove = true
		f := newFakes(removed)
		params := paramsFrom(removed)
		params.Amount = 2000

		got, err := f.app().Update(withAdmin(agencyMain, true), objectID, params)

		if !errors.Is(err, bondObject_domain.ErrBondObjectNotFound) || got != nil {
			t.Fatalf("Update = %v, %v; se esperaba nil, ErrBondObjectNotFound", got, err)
		}
		if len(f.repo.updated) != 0 || len(f.history.added) != 0 {
			t.Fatalf("una edición de un producto borrado llamó %d veces a UpdateOne y %d a History.Add",
				len(f.repo.updated), len(f.history.added))
		}
	})
}

// Un admin de una agencia no principal solo edita productos únicamente de su agencia, los deja solo en ella
// y con un tipo de su alcance: un producto de otra agencia es ErrBondObjectNotFound; uno compartido, o una
// edición con otra agencia, ErrForbidden sin consultar el tipo; y un tipo de otra agencia,
// ErrBondObjectInvalidBondType. La agencia principal edita todos. Ningún error guarda el producto ni
// registra history. CA-02.19
func TestUpdateAgencyScope(t *testing.T) {
	edit := func(bondTypeID string, agencyIDs ...string) bondObject_domain.UpdateParams {
		p := paramsFrom(storedObject(objectID))
		p.Amount = 2000
		p.BondTypeID = bondTypeID
		p.AgencyID = agencyIDs
		return p
	}

	tests := []struct {
		name          string
		ctx           context.Context
		id            string
		params        bondObject_domain.UpdateParams
		wantErr       error
		wantTypeReads int
	}{
		{name: "admin de A, producto de B", ctx: withAdmin(agencyA, false), id: otherObjectID, params: edit(typeID, agencyA), wantErr: bondObject_domain.ErrBondObjectNotFound},
		{name: "admin de A, producto compartido", ctx: withAdmin(agencyA, false), id: sharedObjectID, params: edit(typeID, agencyA), wantErr: shared_domain.ErrForbidden},
		{name: "admin de A, producto compartido con un tipo de B", ctx: withAdmin(agencyA, false), id: sharedObjectID, params: edit(otherTypeID, agencyA), wantErr: shared_domain.ErrForbidden},
		{name: "admin de A, producto de A con A y B", ctx: withAdmin(agencyA, false), id: objectID, params: edit(typeID, agencyA, agencyB), wantErr: shared_domain.ErrForbidden},
		{name: "admin de A, producto de A con un tipo de B", ctx: withAdmin(agencyA, false), id: objectID, params: edit(otherTypeID, agencyA), wantErr: bondObject_domain.ErrBondObjectInvalidBondType, wantTypeReads: 1},
		{name: "admin de A, producto de A con A", ctx: withAdmin(agencyA, false), id: objectID, params: edit(typeID, agencyA), wantTypeReads: 1},
		{name: "principal, producto de B con A y B", ctx: withAdmin(agencyMain, true), id: otherObjectID, params: edit(typeID, agencyA, agencyB), wantTypeReads: 1},
		{name: "principal, producto compartido con B", ctx: withAdmin(agencyMain, true), id: sharedObjectID, params: edit(otherTypeID, agencyB), wantTypeReads: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(agencyObjects()...)
			f.types.existing[otherTypeID] = []string{agencyB}
			before := *f.repo.objects[tt.id]

			got, err := f.app().Update(tt.ctx, tt.id, tt.params)

			if len(f.types.reads) != tt.wantTypeReads {
				t.Errorf("el tipo se consultó %d veces, se esperaban %d", len(f.types.reads), tt.wantTypeReads)
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || got != nil {
					t.Fatalf("Update = %v, %v; se esperaba nil, %v", got, err, tt.wantErr)
				}
				if len(f.repo.updated) != 0 || len(f.history.added) != 0 {
					t.Fatalf("una edición rechazada llamó %d veces a UpdateOne y %d a History.Add", len(f.repo.updated), len(f.history.added))
				}
				if after := f.repo.objects[tt.id]; after.Amount != before.Amount || !slices.Equal(after.AgencyID, before.AgencyID) {
					t.Fatalf("el producto guardado cambió: %+v, se esperaba %+v", *after, before)
				}
				return
			}
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if got.Amount != 2000 || !slices.Equal(got.AgencyID, tt.params.AgencyID) || got.BondTypeID != tt.params.BondTypeID {
				t.Errorf("Update devolvió %+v, se esperaban los campos de la edición", got)
			}
			if len(f.repo.updated) != 1 || len(f.history.added) != 1 {
				t.Fatalf("la edición llamó %d veces a UpdateOne y %d a History.Add, se esperaba 1 y 1", len(f.repo.updated), len(f.history.added))
			}
		})
	}
}
