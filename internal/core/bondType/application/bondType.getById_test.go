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

			got, err := newApp(repo, &fakeCounter{}).GetByID(withAdmin(agencyMain, true), tt.id)

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

		got, err := newApp(repo, &fakeCounter{}).GetByID(withAdmin(agencyMain, true), typeID)

		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.ID != typeID || got.Name != "Bono Plus" {
			t.Fatalf("GetByID devolvió %+v, se esperaba el tipo %s", got, typeID)
		}
	})
}

// Un admin de una agencia no principal lee los tipos que incluyen su agencia, también los compartidos; uno
// solo de otra agencia es ErrBondTypeNotFound. La agencia principal lee todos, y sin sesión no se lee
// ninguno. CA-01.10
func TestGetByIDAgencyScope(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		id      string
		wantErr error
	}{
		{name: "admin de A, tipo de A", ctx: withAdmin(agencyA, false), id: typeID},
		{name: "admin de A, tipo compartido", ctx: withAdmin(agencyA, false), id: sharedTypeID},
		{name: "admin de A, tipo de B", ctx: withAdmin(agencyA, false), id: otherTypeID, wantErr: bondType_domain.ErrBondTypeNotFound},
		{name: "principal, tipo de A", ctx: withAdmin(agencyMain, true), id: typeID},
		{name: "principal, tipo de B", ctx: withAdmin(agencyMain, true), id: otherTypeID},
		{name: "principal, tipo compartido", ctx: withAdmin(agencyMain, true), id: sharedTypeID},
		{name: "sin sesión", ctx: context.Background(), id: typeID, wantErr: bondType_domain.ErrBondTypeNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository(agencyTypes()...)

			got, err := newApp(repo, &fakeCounter{}).GetByID(tt.ctx, tt.id)

			if len(repo.reads) != 1 {
				t.Errorf("el repositorio se leyó %d veces, se esperaba 1", len(repo.reads))
			}
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || got != nil {
					t.Fatalf("GetByID(%q) = %v, %v; se esperaba nil, %v", tt.id, got, err, tt.wantErr)
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
