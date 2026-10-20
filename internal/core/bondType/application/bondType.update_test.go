package bondType_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
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

		got, err := newApp(repo, &fakeCounter{}).Update(context.Background(), typeID, editParams())

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

			got, err := newApp(repo, &fakeCounter{}).Update(context.Background(), tt.id, editParams())

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

		got, err := newApp(repo, &fakeCounter{}).Update(context.Background(), typeID, editParams())

		if !errors.Is(err, bondType_domain.ErrBondTypeNotFound) || got != nil {
			t.Fatalf("Update = %v, %v; se esperaba ErrBondTypeNotFound", got, err)
		}
	})

	t.Run("sin name", func(t *testing.T) {
		repo := newRepository(storedType(typeID))
		params := editParams()
		params.Name = " "

		_, err := newApp(repo, &fakeCounter{}).Update(context.Background(), typeID, params)

		if !errors.Is(err, bondType_domain.ErrBondTypeNameRequired) {
			t.Fatalf("error %v, se esperaba ErrBondTypeNameRequired", err)
		}
		if len(repo.updated) != 0 {
			t.Fatalf("UpdateOne se llamó %d veces con una edición inválida", len(repo.updated))
		}
	})
}
