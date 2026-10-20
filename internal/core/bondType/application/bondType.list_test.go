package bondType_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// El listado de administración busca los tipos no borrados de cualquiera de las agencias pedidas, ya
// limpias, sin filtrar por activos. Un agencyId inválido no llega al repositorio. CA-01.1
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

			types, err := newApp(repo, &fakeCounter{}).List(context.Background(), tt.agencyIDs)

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

		_, err := newApp(repo, &fakeCounter{}).List(context.Background(), []string{agencyA, "abc"})

		if !errors.Is(err, shared_domain.ErrInvalidID) {
			t.Fatalf("error %v, se esperaba ErrInvalidID", err)
		}
		if len(repo.finds) != 0 {
			t.Fatalf("Find se llamó %d veces con un agencyId inválido", len(repo.finds))
		}
	})
}
