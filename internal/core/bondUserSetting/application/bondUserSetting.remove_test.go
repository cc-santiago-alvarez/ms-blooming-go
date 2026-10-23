package bondUserSetting_application_test

import (
	"errors"
	"slices"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// La baja deja isRemove true y una segunda baja es ErrBondUserSettingNotFound. Para un admin de una agencia
// no principal, un setting inexistente o de otra agencia es ErrBondUserSettingNotFound y no se borra, y un id
// inválido no llega al repositorio. CA-09.9
func TestRemove(t *testing.T) {
	t.Run("baja y segunda baja", func(t *testing.T) {
		f := newFakes(storedSetting(settingID, userID, agencyB))
		app := f.app()

		if err := app.Remove(withAdmin(agencyB, false), settingID); err != nil {
			t.Fatalf("Remove: %v", err)
		}
		if !f.repo.settings[settingID].IsRemove {
			t.Fatalf("isRemove false, se esperaba true después de la baja")
		}

		err := app.Remove(withAdmin(agencyB, false), settingID)

		if !errors.Is(err, bondUserSetting_domain.ErrBondUserSettingNotFound) {
			t.Fatalf("la segunda baja devolvió %v, se esperaba ErrBondUserSettingNotFound", err)
		}
		if !slices.Equal(f.repo.deleted, []string{settingID}) {
			t.Fatalf("SoftDelete recibió %v, se esperaba solo la primera baja de %s", f.repo.deleted, settingID)
		}
	})

	tests := []struct {
		name      string
		id        string
		wantErr   error
		wantReads int
	}{
		{name: "inexistente", id: missingID, wantErr: bondUserSetting_domain.ErrBondUserSettingNotFound, wantReads: 1},
		{name: "de otra agencia", id: legacySettingID, wantErr: bondUserSetting_domain.ErrBondUserSettingNotFound, wantReads: 1},
		{name: "id inválido", id: "abc", wantErr: shared_domain.ErrInvalidID, wantReads: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(storedSetting(settingID, userID, agencyB), storedSetting(legacySettingID, user2ID, agencyA))

			err := f.app().Remove(withAdmin(agencyB, false), tt.id)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Remove(%q) = %v, se esperaba %v", tt.id, err, tt.wantErr)
			}
			if len(f.repo.reads) != tt.wantReads {
				t.Errorf("el repositorio se leyó %d veces, se esperaban %d", len(f.repo.reads), tt.wantReads)
			}
			if len(f.repo.deleted) != 0 {
				t.Fatalf("SoftDelete recibió %v sin un setting de su alcance", f.repo.deleted)
			}
			for id, s := range f.repo.settings {
				if s.IsRemove {
					t.Errorf("el setting %s quedó borrado", id)
				}
			}
		})
	}
}
