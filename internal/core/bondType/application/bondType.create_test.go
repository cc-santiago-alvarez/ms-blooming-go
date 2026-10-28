package bondType_application_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	bondType_domain "github.com/futurapps/ms-blooming-go/internal/core/bondType/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Un alta válida guarda y devuelve un tipo con _id UUID, isRemove false, createdAt en UTC y los campos
// limpios; active es false si no viene. Sin name o sin agencias no se guarda nada. CA-01.5
func TestCreate(t *testing.T) {
	t.Run("alta válida", func(t *testing.T) {
		repo := newRepository()
		params := bondType_domain.CreateParams{
			Name:        "  Bono Plus ",
			Description: " Renta fija 12 meses ",
			AgencyID:    []string{agencyA, " " + agencyB, "", agencyA},
		}

		got, err := newApp(repo, &fakeCounter{}).Create(withAdmin(agencyMain, true), params)

		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if shared_domain.IDFormat(got.ID) != shared_domain.IDKindUUID {
			t.Errorf("_id %q, se esperaba un UUID v4", got.ID)
		}
		if got.IsRemove {
			t.Errorf("isRemove true, se esperaba false")
		}
		if got.CreatedAt.IsZero() || got.CreatedAt.Location() != time.UTC {
			t.Errorf("createdAt %v, se esperaba una hora en UTC", got.CreatedAt)
		}
		if !got.UpdatedAt.Equal(got.CreatedAt) {
			t.Errorf("updatedAt %v, se esperaba igual a createdAt %v", got.UpdatedAt, got.CreatedAt)
		}
		if got.Name != "Bono Plus" || got.Description != "Renta fija 12 meses" {
			t.Errorf("name %q y description %q, se esperaban sin espacios", got.Name, got.Description)
		}
		if !slices.Equal(got.AgencyID, []string{agencyA, agencyB}) {
			t.Errorf("agencyId %v, se esperaba [%s %s]", got.AgencyID, agencyA, agencyB)
		}
		if got.Active {
			t.Errorf("active true, se esperaba false por defecto")
		}
		if len(repo.inserted) != 1 || repo.inserted[0].ID != got.ID {
			t.Fatalf("InsertOne recibió %v, se esperaba el tipo %s", repo.inserted, got.ID)
		}
	})

	invalid := []struct {
		name    string
		params  bondType_domain.CreateParams
		wantErr error
	}{
		{name: "sin name", params: bondType_domain.CreateParams{AgencyID: []string{agencyA}}, wantErr: bondType_domain.ErrBondTypeNameRequired},
		{name: "name con solo espacios", params: bondType_domain.CreateParams{Name: "   ", AgencyID: []string{agencyA}}, wantErr: bondType_domain.ErrBondTypeNameRequired},
		{name: "sin agencias", params: bondType_domain.CreateParams{Name: "Bono Plus"}, wantErr: bondType_domain.ErrBondTypeAgencyRequired},
		{name: "agencias vacías", params: bondType_domain.CreateParams{Name: "Bono Plus", AgencyID: []string{"", " "}}, wantErr: bondType_domain.ErrBondTypeAgencyRequired},
		{name: "agencia inválida", params: bondType_domain.CreateParams{Name: "Bono Plus", AgencyID: []string{agencyA, "abc"}}, wantErr: shared_domain.ErrInvalidID},
	}
	for _, tt := range invalid {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository()

			got, err := newApp(repo, &fakeCounter{}).Create(withAdmin(agencyMain, true), tt.params)

			if !errors.Is(err, tt.wantErr) || got != nil {
				t.Fatalf("Create = %v, %v; se esperaba nil, %v", got, err, tt.wantErr)
			}
			if len(repo.inserted) != 0 {
				t.Fatalf("InsertOne se llamó %d veces con un alta inválida", len(repo.inserted))
			}
		})
	}
}

// Un admin de una agencia no principal solo crea tipos únicamente de su agencia: con otra agencia, sola o
// junto a la suya, es ErrForbidden, pero sin name sigue siendo ErrBondTypeNameRequired. La agencia
// principal crea para cualquiera, y sin sesión no se crea nada. Ningún error guarda el tipo. CA-01.11
func TestCreateAgencyScope(t *testing.T) {
	tests := []struct {
		name    string
		ctx     context.Context
		params  bondType_domain.CreateParams
		wantErr error
	}{
		{name: "admin de A con A y B", ctx: withAdmin(agencyA, false), params: bondType_domain.CreateParams{Name: "Bono Plus", AgencyID: []string{agencyA, agencyB}}, wantErr: shared_domain.ErrForbidden},
		{name: "admin de A con B", ctx: withAdmin(agencyA, false), params: bondType_domain.CreateParams{Name: "Bono Plus", AgencyID: []string{agencyB}}, wantErr: shared_domain.ErrForbidden},
		{name: "admin de A sin name con B", ctx: withAdmin(agencyA, false), params: bondType_domain.CreateParams{AgencyID: []string{agencyB}}, wantErr: bondType_domain.ErrBondTypeNameRequired},
		{name: "admin de A con A", ctx: withAdmin(agencyA, false), params: bondType_domain.CreateParams{Name: "Bono Plus", AgencyID: []string{agencyA}}},
		{name: "principal con B", ctx: withAdmin(agencyMain, true), params: bondType_domain.CreateParams{Name: "Bono Plus", AgencyID: []string{agencyB}}},
		{name: "sin sesión", ctx: context.Background(), params: bondType_domain.CreateParams{Name: "Bono Plus", AgencyID: []string{agencyA}}, wantErr: shared_domain.ErrForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository()

			got, err := newApp(repo, &fakeCounter{}).Create(tt.ctx, tt.params)

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || got != nil {
					t.Fatalf("Create = %v, %v; se esperaba nil, %v", got, err, tt.wantErr)
				}
				if len(repo.inserted) != 0 {
					t.Fatalf("InsertOne se llamó %d veces con un alta rechazada", len(repo.inserted))
				}
				return
			}
			if err != nil {
				t.Fatalf("Create: %v", err)
			}
			if !slices.Equal(got.AgencyID, tt.params.AgencyID) {
				t.Errorf("agencyId %v, se esperaba %v", got.AgencyID, tt.params.AgencyID)
			}
			if len(repo.inserted) != 1 || repo.inserted[0].ID != got.ID {
				t.Fatalf("InsertOne recibió %v, se esperaba el tipo %s", repo.inserted, got.ID)
			}
		})
	}
}
