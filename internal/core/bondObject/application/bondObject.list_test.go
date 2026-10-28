package bondObject_application_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Para un admin de la agencia principal, el listado de administración busca los productos de cualquiera
// de las agencias pedidas, ya limpias; sin agencias no filtra. Un agencyId inválido no llega al
// repositorio. CA-02.1
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
			f := newFakes()

			if _, err := f.app().List(withAdmin(agencyMain, true), tt.agencyIDs); err != nil {
				t.Fatalf("List: %v", err)
			}

			if len(f.repo.finds) != 1 || !slices.Equal(f.repo.finds[0], tt.want) {
				t.Fatalf("Find recibió %v, se esperaba una llamada con %v", f.repo.finds, tt.want)
			}
		})
	}

	t.Run("agencyId inválido", func(t *testing.T) {
		f := newFakes()

		_, err := f.app().List(withAdmin(agencyMain, true), []string{agencyA, "abc"})

		if !errors.Is(err, shared_domain.ErrInvalidID) {
			t.Fatalf("error %v, se esperaba ErrInvalidID", err)
		}
		if len(f.repo.finds) != 0 {
			t.Fatalf("Find se llamó %d veces con un agencyId inválido", len(f.repo.finds))
		}
	})
}

// Para un admin de una agencia no principal, el listado y el agrupado por tipo buscan solo los productos
// de su agencia, aunque el query pida otras; un agencyId inválido sigue siendo ErrInvalidID. Sin sesión no
// hay productos ni se consulta el repositorio. CA-02.18
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			app := f.app()

			objects, err := app.List(tt.ctx, tt.agencyIDs)
			groups, groupedErr := app.ListGroupedByType(tt.ctx, tt.agencyIDs)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || !errors.Is(groupedErr, tt.wantErr) {
					t.Fatalf("List = %v y ListGroupedByType = %v, se esperaba %v", err, groupedErr, tt.wantErr)
				}
				if len(f.repo.finds) != 0 || len(f.repo.groupedFinds) != 0 {
					t.Fatalf("se consultó el repositorio con un agencyId inválido: %v y %v", f.repo.finds, f.repo.groupedFinds)
				}
				return
			}
			if err != nil || groupedErr != nil {
				t.Fatalf("List = %v y ListGroupedByType = %v", err, groupedErr)
			}
			if tt.wantFind == nil {
				for name, result := range map[string]any{"List": objects, "ListGroupedByType": groups} {
					if raw, _ := json.Marshal(result); string(raw) != "[]" {
						t.Errorf("%s = %s, se esperaba []", name, raw)
					}
				}
				if len(f.repo.finds) != 0 || len(f.repo.groupedFinds) != 0 {
					t.Fatalf("se consultó el repositorio sin sesión: %v y %v", f.repo.finds, f.repo.groupedFinds)
				}
				return
			}
			if len(f.repo.finds) != 1 || !slices.Equal(f.repo.finds[0], tt.wantFind) {
				t.Errorf("Find recibió %v, se esperaba una llamada con %v", f.repo.finds, tt.wantFind)
			}
			if len(f.repo.groupedFinds) != 1 || !slices.Equal(f.repo.groupedFinds[0], tt.wantFind) {
				t.Errorf("FindGroupedByType recibió %v, se esperaba una llamada con %v", f.repo.groupedFinds, tt.wantFind)
			}
		})
	}
}
