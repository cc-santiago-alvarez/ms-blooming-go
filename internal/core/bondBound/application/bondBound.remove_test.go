package bondBound_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// La baja marca el bono como borrado si está en el alcance de la sesión: su agencia, o todas si es la
// principal. Un bono inexistente, ya borrado, de otra agencia o que otra baja borró antes es
// ErrBondBoundNotFound; fuera del alcance ni siquiera intenta la baja. CA-06.13
func TestRemove(t *testing.T) {
	removed := func(b *bondBound_domain.BondBound) { b.IsRemove = true }
	inAgencyB := func(b *bondBound_domain.BondBound) { b.AgencyID = agencyB }

	tests := []struct {
		name        string
		ctx         context.Context
		modify      func(b *bondBound_domain.BondBound)
		deleteLost  bool
		id          string
		wantErr     error
		wantDeleted []string
	}{
		{name: "admin de la agencia del bono", ctx: withAdmin(), id: boundID, wantDeleted: []string{boundID}},
		{name: "id legacy", ctx: withAdmin(), id: legacyBoundID, wantDeleted: []string{legacyBoundID}},
		{name: "admin principal, bono de otra agencia", ctx: withMainAdmin(), modify: inAgencyB, id: boundID, wantDeleted: []string{boundID}},
		{name: "admin no principal, bono de otra agencia", ctx: withAdmin(), modify: inAgencyB, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "inexistente", ctx: withAdmin(), id: missingID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "ya borrado", ctx: withAdmin(), modify: removed, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "otra baja ganó la carrera", ctx: withAdmin(), deleteLost: true, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound, wantDeleted: []string{boundID}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := []*bondBound_domain.BondBound{storedBound(boundID), storedBound(legacyBoundID)}
			for _, b := range stored {
				if tt.modify != nil {
					tt.modify(b)
				}
			}
			f := newFakes(stored...)
			f.repo.deleteLost = tt.deleteLost

			err := f.app().Remove(tt.ctx, tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Remove = %v, se esperaba %v", err, tt.wantErr)
			}
			if !slices.Equal(f.repo.deleted, tt.wantDeleted) {
				t.Errorf("bajas %v, se esperaban %v", f.repo.deleted, tt.wantDeleted)
			}
			if tt.wantErr == nil && !f.repo.bounds[tt.id].IsRemove {
				t.Errorf("el bono %q no quedó borrado", tt.id)
			}
		})
	}
}

// Un id fuera de formato es ErrInvalidID y no llega al repositorio. CA-06.13
func TestRemoveInvalidID(t *testing.T) {
	f := newFakes(storedBound(boundID))

	err := f.app().Remove(withAdmin(), "abc")

	if !errors.Is(err, shared_domain.ErrInvalidID) {
		t.Fatalf("Remove = %v, se esperaba ErrInvalidID", err)
	}
	assertNoQueries(t, f)
}

// Un error del repositorio se devuelve sin cambiarlo y no intenta la baja. CA-06.13
func TestRemoveRepositoryError(t *testing.T) {
	f := newFakes(storedBound(boundID))
	f.repo.err = errors.New("mongo caído")

	err := f.app().Remove(withAdmin(), boundID)

	if !errors.Is(err, f.repo.err) {
		t.Fatalf("Remove = %v, se esperaba el error del repositorio", err)
	}
	if len(f.repo.deleted) != 0 {
		t.Errorf("bajas %v, se esperaba ninguna", f.repo.deleted)
	}
}
