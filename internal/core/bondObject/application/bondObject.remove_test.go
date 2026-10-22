package bondObject_application_test

import (
	"context"
	"errors"
	"testing"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// La baja deja isRemove true y registra el history BOND_REMOVE; una segunda baja, o la de un producto
// inexistente, es ErrBondObjectNotFound sin history, y un id inválido no llega al repositorio. CA-02.9
func TestRemove(t *testing.T) {
	t.Run("baja y segunda baja", func(t *testing.T) {
		f := newFakes(storedObject(objectID))
		app := f.app()

		if err := app.Remove(context.Background(), objectID); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if !f.repo.objects[objectID].IsRemove {
			t.Errorf("isRemove false, se esperaba true después de la baja")
		}
		if len(f.history.added) != 1 {
			t.Fatalf("History.Add se llamó %d veces, se esperaba 1", len(f.history.added))
		}
		if h := f.history.added[0]; h.TypeID != bondHistory_domain.HistoryBond || h.Text != shared_domain.TextBondRemove || h.RelatedID != objectID {
			t.Errorf("History.Add recibió %+v, se esperaba typeId %q, text %q y relatedId %s", h, bondHistory_domain.HistoryBond, shared_domain.TextBondRemove, objectID)
		}

		err := app.Remove(context.Background(), objectID)

		if !errors.Is(err, bondObject_domain.ErrBondObjectNotFound) {
			t.Fatalf("la segunda baja devolvió %v, se esperaba ErrBondObjectNotFound", err)
		}
		if len(f.history.added) != 1 {
			t.Fatalf("History.Add se llamó %d veces después de la segunda baja, se esperaba 1", len(f.history.added))
		}
	})

	tests := []struct {
		name        string
		id          string
		wantErr     error
		wantDeletes int
	}{
		{name: "inexistente", id: missingID, wantErr: bondObject_domain.ErrBondObjectNotFound, wantDeletes: 1},
		{name: "id inválido", id: "abc", wantErr: shared_domain.ErrInvalidID, wantDeletes: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(storedObject(objectID))

			err := f.app().Remove(context.Background(), tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Remove(%q) = %v, se esperaba %v", tt.id, err, tt.wantErr)
			}
			if len(f.repo.deleted) != tt.wantDeletes {
				t.Errorf("SoftDelete se llamó %d veces, se esperaban %d", len(f.repo.deleted), tt.wantDeletes)
			}
			if len(f.history.added) != 0 {
				t.Fatalf("History.Add se llamó %d veces sin baja", len(f.history.added))
			}
		})
	}
}
