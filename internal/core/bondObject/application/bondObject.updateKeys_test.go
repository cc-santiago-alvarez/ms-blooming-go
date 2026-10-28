package bondObject_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// La actualización masiva de clave no filtra por agencia para un admin de la agencia principal, y para uno
// de otra agencia filtra por la suya; devuelve lo que responde el repositorio. Sin sesión no cambia nada, y
// una clave vacía es ErrBondObjectKeyRequired; en los dos casos no se llama al repositorio. CA-02.20
func TestUpdateKeysAgencyScope(t *testing.T) {
	stored := bondObject_domain.UpdateKeysResult{Matched: 2, Modified: 2}

	tests := []struct {
		name       string
		ctx        context.Context
		key        string
		wantCall   bool
		wantAgency string
		want       bondObject_domain.UpdateKeysResult
		wantErr    error
	}{
		{name: "admin de la principal", ctx: withAdmin(agencyMain, true), key: "BP24", wantCall: true, want: stored},
		{name: "admin de A", ctx: withAdmin(agencyA, false), key: "BP24", wantCall: true, wantAgency: agencyA, want: stored},
		{name: "sin sesión", ctx: context.Background(), key: "BP24"},
		{name: "admin de A sin clave", ctx: withAdmin(agencyA, false), key: " ", wantErr: bondObject_domain.ErrBondObjectKeyRequired},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			f.repo.keysResult = stored
			ids := []string{objectID, otherObjectID, sharedObjectID}

			got, err := f.app().UpdateKeys(tt.ctx, bondObject_domain.UpdateKeysParams{Bonds: ids, Key: tt.key})

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateKeys = %v, se esperaba %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("UpdateKeys devolvió %+v, se esperaba %+v", got, tt.want)
			}
			if !tt.wantCall {
				if len(f.repo.keyUpdates) != 0 {
					t.Fatalf("UpdateKeys del repositorio se llamó %d veces, se esperaba 0", len(f.repo.keyUpdates))
				}
				return
			}
			if len(f.repo.keyUpdates) != 1 {
				t.Fatalf("UpdateKeys del repositorio se llamó %d veces, se esperaba 1", len(f.repo.keyUpdates))
			}
			call := f.repo.keyUpdates[0]
			if !slices.Equal(call.ids, ids) || call.key != tt.key {
				t.Errorf("UpdateKeys recibió %v y %q, se esperaba %v y %q", call.ids, call.key, ids, tt.key)
			}
			switch {
			case tt.wantAgency == "" && call.agencyID != nil:
				t.Errorf("agencyID %q, se esperaba nil", *call.agencyID)
			case tt.wantAgency != "" && (call.agencyID == nil || *call.agencyID != tt.wantAgency):
				t.Errorf("agencyID %v, se esperaba %q", call.agencyID, tt.wantAgency)
			}
		})
	}
}
