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

// La baja deja isRemove true y registra el history BOND_REMOVE; una segunda baja, o la de un producto
// inexistente, es ErrBondObjectNotFound sin borrar ni registrar history, y un id inválido no llega al
// repositorio. CA-02.9
func TestRemove(t *testing.T) {
	t.Run("baja y segunda baja", func(t *testing.T) {
		f := newFakes(storedObject(objectID))
		app := f.app()

		if err := app.Remove(withAdmin(agencyMain, true), objectID); err != nil {
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

		err := app.Remove(withAdmin(agencyMain, true), objectID)

		if !errors.Is(err, bondObject_domain.ErrBondObjectNotFound) {
			t.Fatalf("la segunda baja devolvió %v, se esperaba ErrBondObjectNotFound", err)
		}
		if len(f.history.added) != 1 {
			t.Fatalf("History.Add se llamó %d veces después de la segunda baja, se esperaba 1", len(f.history.added))
		}
	})

	tests := []struct {
		name    string
		id      string
		wantErr error
	}{
		{name: "inexistente", id: missingID, wantErr: bondObject_domain.ErrBondObjectNotFound},
		{name: "id inválido", id: "abc", wantErr: shared_domain.ErrInvalidID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(storedObject(objectID))

			err := f.app().Remove(withAdmin(agencyMain, true), tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Remove(%q) = %v, se esperaba %v", tt.id, err, tt.wantErr)
			}
			if len(f.repo.deleted) != 0 {
				t.Errorf("SoftDelete se llamó %d veces, se esperaba 0", len(f.repo.deleted))
			}
			if len(f.history.added) != 0 {
				t.Fatalf("History.Add se llamó %d veces sin baja", len(f.history.added))
			}
		})
	}
}

// Un admin de una agencia no principal solo borra productos únicamente de su agencia: uno de otra agencia
// es ErrBondObjectNotFound, y uno compartido, ErrForbidden. La agencia principal borra todos. Ningún error
// borra el producto ni registra history. CA-02.19
func TestRemoveAgencyScope(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		id      string
		wantErr error
	}{
		{name: "admin de A, producto de B", ctx: withAdmin(agencyA, false), id: otherObjectID, wantErr: bondObject_domain.ErrBondObjectNotFound},
		{name: "admin de A, producto compartido", ctx: withAdmin(agencyA, false), id: sharedObjectID, wantErr: shared_domain.ErrForbidden},
		{name: "admin de A, producto de A", ctx: withAdmin(agencyA, false), id: objectID},
		{name: "principal, producto de B", ctx: withAdmin(agencyMain, true), id: otherObjectID},
		{name: "principal, producto compartido", ctx: withAdmin(agencyMain, true), id: sharedObjectID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(agencyObjects()...)

			err := f.app().Remove(tt.ctx, tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Remove(%q) = %v, se esperaba %v", tt.id, err, tt.wantErr)
				}
				if len(f.repo.deleted) != 0 || len(f.history.added) != 0 || f.repo.objects[tt.id].IsRemove {
					t.Fatalf("una baja rechazada llamó %d veces a SoftDelete y %d a History.Add", len(f.repo.deleted), len(f.history.added))
				}
				return
			}
			if err != nil {
				t.Fatalf("Remove(%q): %v", tt.id, err)
			}
			if !slices.Equal(f.repo.deleted, []string{tt.id}) || !f.repo.objects[tt.id].IsRemove || len(f.history.added) != 1 {
				t.Fatalf("SoftDelete recibió %v y History.Add se llamó %d veces, se esperaba [%s], isRemove true y 1",
					f.repo.deleted, len(f.history.added), tt.id)
			}
		})
	}
}
