package bondUserSetting_application_test

import (
	"context"
	"errors"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Para un admin de una agencia no principal, un setting inexistente, borrado o de otra agencia es
// ErrBondUserSettingNotFound, y un id inválido no llega al repositorio. Uno de su agencia, o cualquiera para
// la agencia principal, se devuelve. CA-09.9
func TestGetByID(t *testing.T) {
	removed := storedSetting(setting3ID, user3ID, agencyB)
	removed.IsRemove = true
	settings := []*bondUserSetting_domain.BondUserSetting{
		storedSetting(settingID, userID, agencyB),
		storedSetting(legacySettingID, user2ID, agencyA),
		removed,
	}

	tests := []struct {
		name      string
		ctx       context.Context
		id        string
		wantErr   error
		wantReads int
	}{
		{name: "inexistente", ctx: withAdmin(agencyB, false), id: missingID, wantErr: bondUserSetting_domain.ErrBondUserSettingNotFound, wantReads: 1},
		{name: "borrado", ctx: withAdmin(agencyB, false), id: setting3ID, wantErr: bondUserSetting_domain.ErrBondUserSettingNotFound, wantReads: 1},
		{name: "de otra agencia", ctx: withAdmin(agencyB, false), id: legacySettingID, wantErr: bondUserSetting_domain.ErrBondUserSettingNotFound, wantReads: 1},
		{name: "id inválido", ctx: withAdmin(agencyB, false), id: "abc", wantErr: shared_domain.ErrInvalidID, wantReads: 0},
		{name: "de su agencia", ctx: withAdmin(agencyB, false), id: settingID, wantReads: 1},
		{name: "de otra agencia para la principal", ctx: withAdmin(agencyMain, true), id: legacySettingID, wantReads: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(settings...)

			got, err := f.app().GetByID(tt.ctx, tt.id)

			if len(f.repo.reads) != tt.wantReads {
				t.Errorf("el repositorio se leyó %d veces, se esperaban %d", len(f.repo.reads), tt.wantReads)
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
