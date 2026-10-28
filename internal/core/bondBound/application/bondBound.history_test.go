package bondBound_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// El dueño y un admin que no es dueño reciben el historial del bono, pedido como bono comprado con su id, y
// otro usuario es ErrForbidden. Un bono inexistente, borrado o de otra agencia que la de la sesión es
// ErrBondBoundNotFound aunque la sesión sea admin o del dueño, y también sin sesión; el admin de la agencia
// principal recibe el de cualquier agencia. Con error no se consulta el historial. CA-08.4, CA-08.10
func TestHistoryOfAccess(t *testing.T) {
	removed := func(b *bondBound_domain.BondBound) { b.IsRemove = true }
	inAgencyB := func(b *bondBound_domain.BondBound) { b.AgencyID = agencyB }
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
		{name: "sin sesión", ctx: context.Background(), id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "inexistente", ctx: owner, id: missingID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "inexistente, admin", ctx: withAdmin(), id: missingID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "borrado", ctx: owner, modify: removed, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "borrado, admin", ctx: withAdmin(), modify: removed, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "de otra agencia, admin", ctx: withAdmin(), modify: inAgencyB, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "de otra agencia, dueño", ctx: owner, modify: inAgencyB, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "de otra agencia, otro usuario", ctx: withSession(otherUserID, -5), modify: inAgencyB, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "de otra agencia, admin de la principal", ctx: withMainAdmin(), modify: inAgencyB, id: boundID},
		{name: "id inválido", ctx: owner, id: "abc", wantErr: shared_domain.ErrInvalidID},
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
			f.history.entries = []bondHistory_domain.HistoryEntry{{ID: "h2"}, {ID: "h1"}}

			entries, err := f.app().HistoryOf(tt.ctx, tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("HistoryOf = %v, se esperaba %v", err, tt.wantErr)
				}
				if len(f.history.listed) != 0 {
					t.Errorf("se consultó el historial %+v con %v", f.history.listed, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("HistoryOf: %v", err)
			}
			want := []historyQuery{{typeID: bondHistory_domain.HistoryBondBound, relatedID: tt.id}}
			if !slices.Equal(f.history.listed, want) {
				t.Errorf("consultas al historial %+v, se esperaba %+v", f.history.listed, want)
			}
			if len(entries) != 2 || entries[0].ID != "h2" || entries[1].ID != "h1" {
				t.Errorf("HistoryOf devolvió %+v, se esperaban los registros del historial en su orden", entries)
			}
		})
	}
}
