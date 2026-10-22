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

		got, err := f.app().Update(context.Background(), objectID, params)

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

		got, err := f.app().Update(context.Background(), objectID, params)

		if !errors.Is(err, bondObject_domain.ErrBondObjectNotFound) || got != nil {
			t.Fatalf("Update = %v, %v; se esperaba nil, ErrBondObjectNotFound", got, err)
		}
		if len(f.repo.updated) != 0 || len(f.history.added) != 0 {
			t.Fatalf("una edición de un producto borrado llamó %d veces a UpdateOne y %d a History.Add",
				len(f.repo.updated), len(f.history.added))
		}
	})
}
