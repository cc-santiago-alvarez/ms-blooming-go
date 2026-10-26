package bondBound_application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// detailFakes arma las dependencias con los bonos dados y el producto y el tipo del detalle; el producto
// tiene usuarios autorizados.
func detailFakes(bounds ...*bondBound_domain.BondBound) *fakes {
	f := newFakes(bounds...)
	obj := purchasableObject()
	obj.AuthorizedUserIDs = []string{ownerID, otherUserID}
	f.repo.detailObject = *obj
	f.repo.detailType = bondType_domain.BondType{ID: typeID, Name: "Plus", Active: true}
	return f
}

// El dueño y un admin reciben el detalle; otro usuario es ErrForbidden. Un bono inexistente, borrado o de
// otra agencia que la de la sesión, que no es la principal, es ErrBondBoundNotFound, también para el dueño o
// un admin; un admin de la agencia principal lo ve. CA-06.10
func TestGetDetailsAccess(t *testing.T) {
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
		{name: "admin principal, bono de otra agencia", ctx: withMainAdmin(), modify: inAgencyB, id: boundID},
		{name: "otro usuario", ctx: withSession(otherUserID, -5), id: boundID, wantErr: shared_domain.ErrForbidden},
		{name: "dueño, bono de otra agencia", ctx: owner, modify: inAgencyB, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "admin no principal, bono de otra agencia", ctx: withAdmin(), modify: inAgencyB, id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
		{name: "sin sesión", ctx: context.Background(), id: boundID, wantErr: bondBound_domain.ErrBondBoundNotFound},
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

			detail, err := detailFakes(stored...).app().GetDetails(tt.ctx, tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || detail != nil {
					t.Fatalf("GetDetails = %+v, %v, se esperaba %v", detail, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetDetails: %v", err)
			}
			if detail == nil || detail.ID != tt.id {
				t.Errorf("GetDetails devolvió %+v, se esperaba el detalle de %q", detail, tt.id)
			}
		})
	}
}

// El detalle trae el estado con su nombre, el producto público, el tipo y lo acumulado a hoy, y su JSON no
// tiene los ids de estado, dueño, producto y tipo ni los usuarios autorizados del producto. CA-06.10
func TestGetDetailsContent(t *testing.T) {
	f := detailFakes(boundOnDay(50))

	detail, err := f.app().GetDetails(withSession(ownerID, -5), boundID)
	if err != nil {
		t.Fatalf("GetDetails: %v", err)
	}

	// 1150 / 365 × 50, redondeado
	if detail.DaysPassed != 50 || detail.AcumulateAmount != 157.53 {
		t.Errorf("daysPassed %d y acumulado %v, se esperaban 50 y 157.53", detail.DaysPassed, detail.AcumulateAmount)
	}
	if want := (bondBound_domain.StatusRef{ID: "1", Name: "Iniciado"}); detail.Status != want {
		t.Errorf("status %+v, se esperaba %+v", detail.Status, want)
	}
	if detail.Key != "BP12-1" || detail.AgencyID != agencyA || detail.Amount != 1000 {
		t.Errorf("key %q, agencyId %q y amount %v, se esperaban los del bono", detail.Key, detail.AgencyID, detail.Amount)
	}
	if detail.BondObject.ID != objectID || detail.BondObject.Name != "Bono Plus 12M" || detail.BondType.Name != "Plus" {
		t.Errorf("producto %q %q y tipo %q, se esperaban el producto y el tipo del bono",
			detail.BondObject.ID, detail.BondObject.Name, detail.BondType.Name)
	}

	raw, err := json.Marshal(detail)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	for _, key := range []string{"statusId", "userId", "bondObjectId", "bondTypeId"} {
		if _, ok := body[key]; ok {
			t.Errorf("el detalle tiene %q: %s", key, raw)
		}
	}
	for _, key := range []string{"status", "daysPassed", "acumulateAmount", "bondType"} {
		if _, ok := body[key]; !ok {
			t.Errorf("el detalle no tiene %q: %s", key, raw)
		}
	}
	bondObject, ok := body["bondObject"].(map[string]any)
	if !ok {
		t.Fatalf("el detalle no tiene bondObject: %s", raw)
	}
	if _, ok := bondObject["authorizedUserIds"]; ok {
		t.Errorf("el producto del detalle tiene authorizedUserIds: %s", raw)
	}
}

// Un id fuera de formato es ErrInvalidID y no llega al repositorio, ni siquiera para un admin. CA-06.10
func TestGetDetailsInvalidID(t *testing.T) {
	sessions := []struct {
		name string
		ctx  context.Context
	}{
		{name: "dueño", ctx: withSession(ownerID, -5)},
		{name: "admin", ctx: withAdmin()},
	}
	for _, s := range sessions {
		t.Run(s.name, func(t *testing.T) {
			f := detailFakes(storedBound(boundID))

			_, err := f.app().GetDetails(s.ctx, "abc")

			if !errors.Is(err, shared_domain.ErrInvalidID) {
				t.Fatalf("GetDetails = %v, se esperaba ErrInvalidID", err)
			}
			assertNoQueries(t, f)
		})
	}
}

// Un error del repositorio se devuelve sin cambiarlo. CA-06.10
func TestGetDetailsRepositoryError(t *testing.T) {
	f := detailFakes(storedBound(boundID))
	f.repo.err = errors.New("mongo caído")

	detail, err := f.app().GetDetails(withSession(ownerID, -5), boundID)

	if !errors.Is(err, f.repo.err) || detail != nil {
		t.Fatalf("GetDetails = %+v, %v, se esperaba el error del repositorio", detail, err)
	}
}
