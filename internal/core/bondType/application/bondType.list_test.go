package bondType_application_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Para un admin de la agencia principal, el listado de administración busca los tipos no borrados de
// cualquiera de las agencias pedidas, ya limpias, sin filtrar por activos. Un agencyId inválido no llega al
// repositorio. CA-01.1
func TestListAgencyFilter(t *testing.T) {
	tests := []struct {
		name      string
		agencyIDs []string
		want      []string
	}{
		{name: "varias agencias", agencyIDs: []string{" " + agencyA + " ", agencyB, "", agencyA}, want: []string{agencyA, agencyB}},
		{name: "sin filtro", agencyIDs: nil, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository()
			repo.found = append(repo.found, storedType(typeID))

			types, err := newApp(repo, &fakeCounter{}).List(withAdmin(agencyMain, true), tt.agencyIDs)

			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if len(repo.finds) != 1 {
				t.Fatalf("Find se llamó %d veces, se esperaba 1", len(repo.finds))
			}
			if got := repo.finds[0]; !slices.Equal(got.agencyIDs, tt.want) || got.onlyActive {
				t.Fatalf("Find(%v, %v), se esperaba Find(%v, false)", got.agencyIDs, got.onlyActive, tt.want)
			}
			if len(types) != 1 || types[0].ID != typeID {
				t.Fatalf("List devolvió %v, se esperaba el tipo %s", types, typeID)
			}
		})
	}

	t.Run("agencyId inválido", func(t *testing.T) {
		repo := newRepository()

		_, err := newApp(repo, &fakeCounter{}).List(withAdmin(agencyMain, true), []string{agencyA, "abc"})

		if !errors.Is(err, shared_domain.ErrInvalidID) {
			t.Fatalf("error %v, se esperaba ErrInvalidID", err)
		}
		if len(repo.finds) != 0 {
			t.Fatalf("Find se llamó %d veces con un agencyId inválido", len(repo.finds))
		}
	})
}

// Para un admin de una agencia no principal, el listado busca solo los tipos de su agencia, aunque el query
// pida otras; un agencyId inválido sigue siendo ErrInvalidID. Sin sesión o sin agencia no hay tipos ni se
// consulta el repositorio. CA-01.10
func TestListAgencyScope(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		agencyIDs []string
		wantFind  []string
		wantErr   error
	}{
		{name: "admin de A sin filtro", ctx: withAdmin(agencyA, false), wantFind: []string{agencyA}},
		{name: "admin de A pide B", ctx: withAdmin(agencyA, false), agencyIDs: []string{agencyB}, wantFind: []string{agencyA}},
		{name: "admin de A pide A y B", ctx: withAdmin(agencyA, false), agencyIDs: []string{agencyA, agencyB}, wantFind: []string{agencyA}},
		{name: "admin de A con agencyId inválido", ctx: withAdmin(agencyA, false), agencyIDs: []string{agencyB, "abc"}, wantErr: shared_domain.ErrInvalidID},
		{name: "sin sesión", ctx: context.Background(), agencyIDs: []string{agencyA}},
		{name: "sesión sin agencia", ctx: withAdmin("", false)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository()
			repo.found = append(repo.found, storedType(typeID))

			types, err := newApp(repo, &fakeCounter{}).List(tt.ctx, tt.agencyIDs)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || types != nil {
					t.Fatalf("List = %v, %v; se esperaba nil, %v", types, err, tt.wantErr)
				}
				if len(repo.finds) != 0 {
					t.Fatalf("Find se llamó %d veces con un agencyId inválido", len(repo.finds))
				}
				return
			}
			if err != nil {
				t.Fatalf("List: %v", err)
			}
			if tt.wantFind == nil {
				raw, err := json.Marshal(types)
				if err != nil {
					t.Fatalf("json.Marshal: %v", err)
				}
				if string(raw) != "[]" {
					t.Fatalf("List = %s, se esperaba []", raw)
				}
				if len(repo.finds) != 0 {
					t.Fatalf("Find se llamó %d veces, se esperaba 0", len(repo.finds))
				}
				return
			}
			if len(repo.finds) != 1 {
				t.Fatalf("Find se llamó %d veces, se esperaba 1", len(repo.finds))
			}
			if got := repo.finds[0]; !slices.Equal(got.agencyIDs, tt.wantFind) || got.onlyActive {
				t.Fatalf("Find(%v, %v), se esperaba Find(%v, false)", got.agencyIDs, got.onlyActive, tt.wantFind)
			}
		})
	}
}
