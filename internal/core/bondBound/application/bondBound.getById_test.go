package bondBound_application_test

import (
	"context"
	"errors"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// El dueño y un admin reciben el bono; otro usuario, o una llamada sin sesión, es ErrForbidden, y un bono
// inexistente o borrado es ErrBondBoundNotFound aunque la sesión sea admin. CA-06.10
func TestGetByIDAccess(t *testing.T) {
	removed := func(b *bondBound_domain.BondBound) { b.IsRemove = true }
	owner := withSession(ownerID, -5)

	tests := []struct {
		name    string
		ctx     context.Context
		modify  func(b *bondBound_domain.BondBound)
		id      string
		wantErr error
	}{
		{name: "dueño", ctx: owner, id: boundID},
		{name: "admin que no es dueño", ctx: withAdmin(), id: boundID},
		{name: "dueño con id legacy", ctx: owner, id: legacyBoundID},
		{name: "otro usuario", ctx: withSession(otherUserID, -5), id: boundID, wantErr: shared_domain.ErrForbidden},
		{name: "sin sesión", ctx: context.Background(), id: boundID, wantErr: shared_domain.ErrForbidden},
		{name: "inexistente", ctx: owner, id: missingID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "inexistente, admin", ctx: withAdmin(), id: missingID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "borrado", ctx: owner, modify: removed, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "borrado, admin", ctx: withAdmin(), modify: removed, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stored := []*bondBound_domain.BondBound{storedBound(boundID), storedBound(legacyBoundID)}
			for _, b := range stored {
				if tt.modify != nil {
					tt.modify(b)
				}
			}

			b, err := newFakes(stored...).app().GetByID(tt.ctx, tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || b != nil {
					t.Fatalf("GetByID = %+v, %v, se esperaba %v", b, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByID: %v", err)
			}
			if b == nil || b.ID != tt.id || b.UserID != ownerID || b.Key != "BP12-1" {
				t.Errorf("GetByID devolvió %+v, se esperaba el bono %q de %q", b, tt.id, ownerID)
			}
		})
	}
}

// Un id fuera de formato es ErrInvalidID y no llega al repositorio, ni siquiera para un admin. CA-06.10
func TestGetByIDInvalidID(t *testing.T) {
	sessions := []struct {
		name string
		ctx  context.Context
	}{
		{name: "dueño", ctx: withSession(ownerID, -5)},
		{name: "admin", ctx: withAdmin()},
	}
	for _, s := range sessions {
		t.Run(s.name, func(t *testing.T) {
			f := newFakes(storedBound(boundID))

			_, err := f.app().GetByID(s.ctx, "abc")

			if !errors.Is(err, shared_domain.ErrInvalidID) {
				t.Fatalf("GetByID = %v, se esperaba ErrInvalidID", err)
			}
			assertNoQueries(t, f)
		})
	}
}

// Un error del repositorio se devuelve sin cambiarlo. CA-06.10
func TestGetByIDRepositoryError(t *testing.T) {
	f := newFakes(storedBound(boundID))
	f.repo.err = errors.New("mongo caído")

	b, err := f.app().GetByID(withSession(ownerID, -5), boundID)

	if !errors.Is(err, f.repo.err) || b != nil {
		t.Fatalf("GetByID = %+v, %v, se esperaba el error del repositorio", b, err)
	}
}
