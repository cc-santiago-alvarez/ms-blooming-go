package bondType_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Un tipo con bondObjects no borrados no se borra (ErrBondTypeHasBondObjects); si solo tiene bondObjects
// borrados queda con isRemove true. CA-01.7
func TestRemove(t *testing.T) {
	t.Run("con bondObjects no borrados", func(t *testing.T) {
		repo := newRepository(storedType(typeID))
		counter := &fakeCounter{inUse: true}

		err := newApp(repo, counter).Remove(context.Background(), typeID)

		if !errors.Is(err, bondType_domain.ErrBondTypeHasBondObjects) {
			t.Fatalf("error %v, se esperaba ErrBondTypeHasBondObjects", err)
		}
		if !slices.Equal(counter.existsCalls, []string{typeID}) {
			t.Errorf("ExistsNotRemovedByType recibió %v, se esperaba [%s]", counter.existsCalls, typeID)
		}
		if len(repo.deleted) != 0 || repo.types[typeID].IsRemove {
			t.Fatalf("el tipo se borró aunque tiene bondObjects no borrados")
		}
	})

	t.Run("solo con bondObjects borrados", func(t *testing.T) {
		repo := newRepository(storedType(typeID))
		counter := &fakeCounter{inUse: false}

		if err := newApp(repo, counter).Remove(context.Background(), typeID); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if !slices.Equal(counter.existsCalls, []string{typeID}) {
			t.Errorf("ExistsNotRemovedByType recibió %v, se esperaba [%s]", counter.existsCalls, typeID)
		}
		if !slices.Equal(repo.deleted, []string{typeID}) || !repo.types[typeID].IsRemove {
			t.Fatalf("SoftDelete recibió %v, se esperaba [%s] y el tipo con isRemove true", repo.deleted, typeID)
		}
	})

	notFound := []struct {
		name    string
		id      string
		wantErr error
	}{
		{name: "inexistente", id: missingID, wantErr: bondType_domain.ErrBondTypeNotFound},
		{name: "id inválido", id: "abc", wantErr: shared_domain.ErrInvalidID},
	}
	for _, tt := range notFound {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository(storedType(typeID))
			counter := &fakeCounter{}

			err := newApp(repo, counter).Remove(context.Background(), tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error %v, se esperaba %v", err, tt.wantErr)
			}
			if len(counter.existsCalls) != 0 || len(repo.deleted) != 0 {
				t.Fatalf("se consultó el contador (%v) o se borró (%v) un tipo %s", counter.existsCalls, repo.deleted, tt.name)
			}
		})
	}
}
