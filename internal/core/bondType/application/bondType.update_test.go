package bondType_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// editParams es una edición válida que cambia los cuatro campos del tipo guardado.
func editParams() bondType_domain.UpdateParams {
	return bondType_domain.UpdateParams{
		Name:        "Bono Max",
		Description: "Renta fija 24 meses",
		Active:      false,
		AgencyID:    []string{agencyB},
	}
}

// La edición de un tipo existente reemplaza name, description, active y agencyId y actualiza updatedAt;
// sobre un tipo inexistente o borrado es ErrBondTypeNotFound. CA-01.6
func TestUpdate(t *testing.T) {
	t.Run("tipo existente", func(t *testing.T) {
		repo := newRepository(storedType(typeID))

		got, err := newApp(repo, &fakeCounter{}).Update(withAdmin(agencyMain, true), typeID, editParams())

		if err != nil {
			t.Fatalf("Update: %v", err)
		}
		if got.Name != "Bono Max" || got.Description != "Renta fija 24 meses" || got.Active ||
			!slices.Equal(got.AgencyID, []string{agencyB}) {
			t.Errorf("Update devolvió %+v, se esperaban los cuatro campos de la edición", got)
		}
		if !got.UpdatedAt.After(storedAt) {
			t.Errorf("updatedAt %v, se esperaba posterior a %v", got.UpdatedAt, storedAt)
		}
		if got.ID != typeID || !got.CreatedAt.Equal(storedAt) || got.IsRemove {
			t.Errorf("Update cambió _id, createdAt o isRemove: %+v", got)
		}
		if len(repo.updated) != 1 {
			t.Fatalf("UpdateOne se llamó %d veces, se esperaba 1", len(repo.updated))
		}
		if saved := repo.updated[0]; saved.ID != typeID || saved.Name != got.Name || !saved.UpdatedAt.Equal(got.UpdatedAt) {
			t.Errorf("UpdateOne recibió %+v, se esperaba %+v", saved, *got)
		}
	})

	removed := storedType(otherTypeID)
	removed.IsRemove = true

	notFound := []struct {
		name string
		id   string
	}{
		{name: "inexistente", id: missingID},
		{name: "borrado", id: otherTypeID},
	}
	for _, tt := range notFound {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository(storedType(typeID), removed)

			got, err := newApp(repo, &fakeCounter{}).Update(withAdmin(agencyMain, true), tt.id, editParams())

			if !errors.Is(err, bondType_domain.ErrBondTypeNotFound) || got != nil {
				t.Fatalf("Update = %v, %v; se esperaba ErrBondTypeNotFound", got, err)
			}
			if len(repo.updated) != 0 {
				t.Fatalf("UpdateOne se llamó %d veces sobre un tipo %s", len(repo.updated), tt.name)
			}
		})
	}

	t.Run("borrado después de leerlo", func(t *testing.T) {
		repo := newRepository(storedType(typeID))
		repo.staleUpdate = true

		got, err := newApp(repo, &fakeCounter{}).Update(withAdmin(agencyMain, true), typeID, editParams())

		if !errors.Is(err, bondType_domain.ErrBondTypeNotFound) || got != nil {
			t.Fatalf("Update = %v, %v; se esperaba ErrBondTypeNotFound", got, err)
		}
	})

	t.Run("sin name", func(t *testing.T) {
		repo := newRepository(storedType(typeID))
		params := editParams()
		params.Name = " "

		_, err := newApp(repo, &fakeCounter{}).Update(withAdmin(agencyMain, true), typeID, params)

		if !errors.Is(err, bondType_domain.ErrBondTypeNameRequired) {
			t.Fatalf("error %v, se esperaba ErrBondTypeNameRequired", err)
		}
		if len(repo.updated) != 0 {
			t.Fatalf("UpdateOne se llamó %d veces con una edición inválida", len(repo.updated))
		}
	})
}

// Un admin de una agencia no principal solo edita tipos únicamente de su agencia y los deja solo en ella:
// un tipo de otra agencia es ErrBondTypeNotFound, y uno compartido o una edición con otra agencia,
// ErrForbidden. Una edición inválida es error de validación antes que de alcance. La agencia principal edita
// todos. Ningún error guarda el tipo. CA-01.11
func TestUpdateAgencyScope(t *testing.T) {
	withAgencies := func(agencyIDs ...string) bondType_domain.UpdateParams {
		p := editParams()
		p.AgencyID = agencyIDs
		return p
	}
	withoutName := editParams()
	withoutName.Name = ""

	tests := []struct {
		name    string
		ctx     context.Context
		id      string
		params  bondType_domain.UpdateParams
		wantErr error
	}{
		{name: "admin de A, tipo de B", ctx: withAdmin(agencyA, false), id: otherTypeID, params: withAgencies(agencyB), wantErr: bondType_domain.ErrBondTypeNotFound},
		{name: "admin de A, tipo de B sin name", ctx: withAdmin(agencyA, false), id: otherTypeID, params: withoutName, wantErr: bondType_domain.ErrBondTypeNameRequired},
		{name: "admin de A, tipo compartido", ctx: withAdmin(agencyA, false), id: sharedTypeID, params: withAgencies(agencyA), wantErr: shared_domain.ErrForbidden},
		{name: "admin de A, tipo de A con A y B", ctx: withAdmin(agencyA, false), id: typeID, params: withAgencies(agencyA, agencyB), wantErr: shared_domain.ErrForbidden},
		{name: "admin de A, tipo de A con A", ctx: withAdmin(agencyA, false), id: typeID, params: withAgencies(agencyA)},
		{name: "principal, tipo de B con A y B", ctx: withAdmin(agencyMain, true), id: otherTypeID, params: withAgencies(agencyA, agencyB)},
		{name: "principal, tipo compartido con B", ctx: withAdmin(agencyMain, true), id: sharedTypeID, params: withAgencies(agencyB)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository(agencyTypes()...)
			before := *repo.types[tt.id]

			got, err := newApp(repo, &fakeCounter{}).Update(tt.ctx, tt.id, tt.params)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || got != nil {
					t.Fatalf("Update = %v, %v; se esperaba nil, %v", got, err, tt.wantErr)
				}
				if len(repo.updated) != 0 {
					t.Fatalf("UpdateOne se llamó %d veces con una edición rechazada", len(repo.updated))
				}
				if after := repo.types[tt.id]; after.Name != before.Name || !slices.Equal(after.AgencyID, before.AgencyID) {
					t.Fatalf("el tipo guardado cambió: %+v, se esperaba %+v", *after, before)
				}
				return
			}
			if err != nil {
				t.Fatalf("Update: %v", err)
			}
			if !slices.Equal(got.AgencyID, tt.params.AgencyID) || got.Name != "Bono Max" {
				t.Errorf("Update devolvió %+v, se esperaban los campos de la edición", got)
			}
			if len(repo.updated) != 1 || repo.updated[0].ID != tt.id {
				t.Fatalf("UpdateOne recibió %v, se esperaba el tipo %s", repo.updated, tt.id)
			}
		})
	}
}
