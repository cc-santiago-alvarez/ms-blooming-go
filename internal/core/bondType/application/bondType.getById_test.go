package bondType_application_test

import (
	"context"
	"errors"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Un tipo borrado o inexistente es ErrBondTypeNotFound, y un id inválido no llega al repositorio.
// CA-01.4
func TestGetByIDNotFound(t *testing.T) {
	removed := storedType(otherTypeID)
	removed.IsRemove = true

	tests := []struct {
		name      string
		id        string
		wantErr   error
		wantReads int
	}{
		{name: "borrado", id: otherTypeID, wantErr: bondType_domain.ErrBondTypeNotFound, wantReads: 1},
		{name: "inexistente", id: missingID, wantErr: bondType_domain.ErrBondTypeNotFound, wantReads: 1},
		{name: "id inválido", id: "abc", wantErr: shared_domain.ErrInvalidID, wantReads: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository(storedType(typeID), removed)

			got, err := newApp(repo, &fakeCounter{}).GetByID(context.Background(), tt.id)

			if !errors.Is(err, tt.wantErr) || got != nil {
				t.Fatalf("GetByID(%q) = %v, %v; se esperaba nil, %v", tt.id, got, err, tt.wantErr)
			}
			if len(repo.reads) != tt.wantReads {
				t.Fatalf("el repositorio se leyó %d veces, se esperaban %d", len(repo.reads), tt.wantReads)
			}
		})
	}

	t.Run("existente", func(t *testing.T) {
		repo := newRepository(storedType(typeID), removed)

		got, err := newApp(repo, &fakeCounter{}).GetByID(context.Background(), typeID)

		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ID != typeID || got.Name != "Bono Plus" {
			t.Fatalf("GetByID devolvió %+v, se esperaba el tipo %s", got, typeID)
		}
	})
}
