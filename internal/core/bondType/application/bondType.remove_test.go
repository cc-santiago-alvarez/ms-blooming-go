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

		err := newApp(repo, counter).Remove(withAdmin(agencyMain, true), typeID)

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

		if err := newApp(repo, counter).Remove(withAdmin(agencyMain, true), typeID); err != nil {
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

			err := newApp(repo, counter).Remove(withAdmin(agencyMain, true), tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error %v, se esperaba %v", err, tt.wantErr)
			}
			if len(counter.existsCalls) != 0 || len(repo.deleted) != 0 {
				t.Fatalf("se consultó el contador (%v) o se borró (%v) un tipo %s", counter.existsCalls, repo.deleted, tt.name)
			}
		})
	}
}

// Un admin de una agencia no principal solo borra tipos únicamente de su agencia: uno de otra agencia es
// ErrBondTypeNotFound, y uno compartido, ErrForbidden aunque tenga bondObjects no borrados, sin consultar el
// contador. La agencia principal borra todos. Ningún error borra el tipo. CA-01.11
func TestRemoveAgencyScope(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		id      string
		inUse   bool
		wantErr error
	}{
		{name: "admin de A, tipo de B", ctx: withAdmin(agencyA, false), id: otherTypeID, wantErr: bondType_domain.ErrBondTypeNotFound},
		{name: "admin de A, tipo compartido con bondObjects", ctx: withAdmin(agencyA, false), id: sharedTypeID, inUse: true, wantErr: shared_domain.ErrForbidden},
		{name: "admin de A, tipo compartido sin bondObjects", ctx: withAdmin(agencyA, false), id: sharedTypeID, wantErr: shared_domain.ErrForbidden},
		{name: "admin de A, tipo de A", ctx: withAdmin(agencyA, false), id: typeID},
		{name: "principal, tipo de B", ctx: withAdmin(agencyMain, true), id: otherTypeID},
		{name: "principal, tipo compartido", ctx: withAdmin(agencyMain, true), id: sharedTypeID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository(agencyTypes()...)
			counter := &fakeCounter{inUse: tt.inUse}

			err := newApp(repo, counter).Remove(tt.ctx, tt.id)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error %v, se esperaba %v", err, tt.wantErr)
				}
				if len(counter.existsCalls) != 0 || len(repo.deleted) != 0 || repo.types[tt.id].IsRemove {
					t.Fatalf("se consultó el contador (%v) o se borró (%v) un tipo fuera del alcance", counter.existsCalls, repo.deleted)
				}
				return
			}
			if err != nil {
				t.Fatalf("Remove: %v", err)
			}
			if !slices.Equal(repo.deleted, []string{tt.id}) || !repo.types[tt.id].IsRemove {
				t.Fatalf("SoftDelete recibió %v, se esperaba [%s] y el tipo con isRemove true", repo.deleted, tt.id)
			}
		})
	}
}
