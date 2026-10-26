package bondBound_application_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// buyers son tres bonos del producto: dos de ownerID y uno de otherUserID, que no tiene perfil.
func buyers() []*bondBound_domain.BondBound {
	other := storedBound(legacyBoundID)
	other.UserID = otherUserID
	return []*bondBound_domain.BondBound{storedBound(boundID), other, storedBound(missingID)}
}

// ownerProfile es el perfil público de ownerID.
func ownerProfile() shared_domain.PublicProfile {
	return shared_domain.PublicProfile{ID: ownerID, PublicID: "1020", FirstName: "Juan", LastName: "Pérez", Email: "juan@example.com", AgencyID: agencyA}
}

// Los compradores se consultan en el alcance de la sesión, sus perfiles se leen en una sola consulta y cada
// bono sale con el perfil público de su dueño, sin password; el dueño sin perfil sale sin profile. CA-06.11
func TestListByObject(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		wantScope *string
	}{
		{name: "admin de la agencia A", ctx: withAdmin(), wantScope: agency(agencyA)},
		{name: "admin de la agencia principal", ctx: withMainAdmin()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			f.repo.listed = buyers()
			f.profiles.profiles[ownerID] = ownerProfile()

			items, err := f.app().ListByObject(tt.ctx, objectID)
			if err != nil {
				t.Fatalf("ListByObject: %v", err)
			}

			wantQuery := []repoQuery{{method: "ListByObject", bondObjectID: objectID, agencyID: tt.wantScope}}
			if !reflect.DeepEqual(f.repo.queries, wantQuery) {
				t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, wantQuery)
			}
			if want := [][]string{{ownerID, otherUserID, ownerID}}; !reflect.DeepEqual(f.profiles.requested, want) {
				t.Errorf("perfiles pedidos %v, se esperaba una consulta %v", f.profiles.requested, want)
			}

			if len(items) != 3 {
				t.Fatalf("ListByObject devolvió %d ítems, se esperaban 3", len(items))
			}
			for i, wantID := range []string{boundID, legacyBoundID, missingID} {
				if items[i].ID != wantID {
					t.Errorf("ítem %d con _id %q, se esperaba %q", i, items[i].ID, wantID)
				}
			}
			profile := ownerProfile()
			if items[0].Profile == nil || *items[0].Profile != profile || items[2].Profile == nil || *items[2].Profile != profile {
				t.Errorf("perfiles %+v y %+v, se esperaba el perfil público de %q", items[0].Profile, items[2].Profile, ownerID)
			}
			if items[1].Profile != nil {
				t.Errorf("el bono de %q tiene perfil %+v, se esperaba nil", otherUserID, items[1].Profile)
			}

			raw, err := json.Marshal(items)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			var body []map[string]any
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("json.Unmarshal: %v", err)
			}
			jsonProfile, ok := body[0]["profile"].(map[string]any)
			if !ok || jsonProfile["first_name"] != "Juan" {
				t.Fatalf("profile %v, se esperaba el perfil público: %s", body[0]["profile"], raw)
			}
			if _, ok := jsonProfile["password"]; ok {
				t.Errorf("el perfil tiene password: %s", raw)
			}
			if _, ok := body[1]["profile"]; ok {
				t.Errorf("el bono sin perfil tiene profile: %s", raw)
			}
		})
	}
}

// Un id fuera de formato es ErrInvalidID y no consulta ni lee perfiles. CA-06.11
func TestListByObjectInvalidID(t *testing.T) {
	f := newFakes()

	items, err := f.app().ListByObject(withAdmin(), "abc")

	if !errors.Is(err, shared_domain.ErrInvalidID) || items != nil {
		t.Fatalf("ListByObject = %+v, %v, se esperaba ErrInvalidID", items, err)
	}
	assertNoQueries(t, f)
	if len(f.profiles.requested) != 0 {
		t.Errorf("perfiles pedidos %v, se esperaba ninguno", f.profiles.requested)
	}
}

// Un error del repositorio o del lector de perfiles se devuelve sin cambiarlo. CA-06.11
func TestListByObjectErrors(t *testing.T) {
	t.Run("error del repositorio", func(t *testing.T) {
		f := newFakes()
		f.repo.err = errors.New("mongo caído")

		_, err := f.app().ListByObject(withAdmin(), objectID)

		if !errors.Is(err, f.repo.err) {
			t.Fatalf("ListByObject = %v, se esperaba el error del repositorio", err)
		}
		if len(f.profiles.requested) != 0 {
			t.Errorf("perfiles pedidos %v, se esperaba ninguno", f.profiles.requested)
		}
	})

	t.Run("error de los perfiles", func(t *testing.T) {
		f := newFakes()
		f.repo.listed = buyers()
		f.profiles.err = errors.New("perfiles caídos")

		items, err := f.app().ListByObject(withAdmin(), objectID)

		if !errors.Is(err, f.profiles.err) || items != nil {
			t.Fatalf("ListByObject = %+v, %v, se esperaba el error de los perfiles", items, err)
		}
	})
}
