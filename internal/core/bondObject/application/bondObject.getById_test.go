package bondObject_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// El detalle trae en authorizedUser los perfiles públicos de authorizedUserIds, leídos en una sola
// consulta; un producto borrado es ErrBondObjectNotFound y no se leen perfiles. CA-02.3
func TestGetByID(t *testing.T) {
	t.Run("con autorizados", func(t *testing.T) {
		stored := storedObject(objectID)
		stored.IsPrivateBond = true
		stored.AuthorizedUserIDs = []string{userID, user2ID}
		f := newFakes(stored)
		f.profiles.profiles[userID] = shared_domain.PublicProfile{ID: userID, FirstName: "Ana", Email: "ana@futurapps.com"}
		f.profiles.profiles[user2ID] = shared_domain.PublicProfile{ID: user2ID, FirstName: "Luis", Email: "luis@futurapps.com"}

		got, err := f.app().GetByID(withAdmin(agencyMain, true), objectID)

		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ID != objectID || got.Name != "Bono Plus 12M" {
			t.Errorf("GetByID devolvió %+v, se esperaba el producto %s", got.BondObject, objectID)
		}
		if len(f.profiles.calls) != 1 || !slices.Equal(f.profiles.calls[0], []string{userID, user2ID}) {
			t.Fatalf("GetPublic recibió %v, se esperaba una llamada con [%s %s]", f.profiles.calls, userID, user2ID)
		}
		if len(got.AuthorizedUser) != 2 {
			t.Fatalf("authorizedUser tiene %d perfiles, se esperaban 2", len(got.AuthorizedUser))
		}
		if first := got.AuthorizedUser[0]; first.ID != userID || first.FirstName != "Ana" || first.Email != "ana@futurapps.com" {
			t.Errorf("authorizedUser[0] = %+v, se esperaba el perfil de %s con first_name y email", first, userID)
		}
	})

	removed := storedObject(otherObjectID)
	removed.IsRemove = true

	notFound := []struct {
		name string
		id   string
	}{
		{name: "borrado", id: otherObjectID},
		{name: "inexistente", id: missingID},
	}
	for _, tt := range notFound {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(storedObject(objectID), removed)

			got, err := f.app().GetByID(withAdmin(agencyMain, true), tt.id)

			if !errors.Is(err, bondObject_domain.ErrBondObjectNotFound) || got != nil {
				t.Fatalf("GetByID = %v, %v; se esperaba nil, ErrBondObjectNotFound", got, err)
			}
			if len(f.profiles.calls) != 0 {
				t.Fatalf("GetPublic se llamó %d veces sobre un producto %s", len(f.profiles.calls), tt.name)
			}
		})
	}
}

// Un admin de una agencia no principal ve el detalle de los productos que incluyen su agencia, también los
// compartidos; uno solo de otra agencia es ErrBondObjectNotFound y no se leen perfiles. La agencia principal
// ve todos, y sin sesión no se ve ninguno. CA-02.18
func TestGetByIDAgencyScope(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		id      string
		wantErr error
	}{
		{name: "admin de A, producto de A", ctx: withAdmin(agencyA, false), id: objectID},
		{name: "admin de A, producto compartido", ctx: withAdmin(agencyA, false), id: sharedObjectID},
		{name: "admin de A, producto de B", ctx: withAdmin(agencyA, false), id: otherObjectID, wantErr: bondObject_domain.ErrBondObjectNotFound},
		{name: "principal, producto de A", ctx: withAdmin(agencyMain, true), id: objectID},
		{name: "principal, producto de B", ctx: withAdmin(agencyMain, true), id: otherObjectID},
		{name: "principal, producto compartido", ctx: withAdmin(agencyMain, true), id: sharedObjectID},
		{name: "sin sesión", ctx: context.Background(), id: objectID, wantErr: bondObject_domain.ErrBondObjectNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(agencyObjects()...)

			got, err := f.app().GetByID(tt.ctx, tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || got != nil {
					t.Fatalf("GetByID(%q) = %v, %v; se esperaba nil, %v", tt.id, got, err, tt.wantErr)
				}
				if len(f.profiles.calls) != 0 {
					t.Fatalf("GetPublic se llamó %d veces sobre un producto fuera del alcance", len(f.profiles.calls))
				}
				return
			}
			if err != nil {
				t.Fatalf("GetByID(%q): %v", tt.id, err)
			}
			if got.ID != tt.id {
				t.Fatalf("GetByID devolvió %s, se esperaba %s", got.ID, tt.id)
			}
		})
	}
}
