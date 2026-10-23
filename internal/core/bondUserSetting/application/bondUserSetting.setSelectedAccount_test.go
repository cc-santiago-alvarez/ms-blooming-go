package bondUserSetting_application_test

import (
	"errors"
	"slices"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Con una cuenta propia, el cambio se guarda en el setting del usuario de la sesión, conserva su _id y no
// toca el de otro usuario. Una cuenta de otro usuario o inexistente es ErrAccountNotFound y ningún setting
// cambia; un id inválido no llega a leer la cuenta. CA-09.5
func TestSetSelectedAccountOwnership(t *testing.T) {
	t.Run("cuenta propia", func(t *testing.T) {
		mine := storedSetting(settingID, userID, agencyA)
		mine.SelectedAccountID = ptr(accountOld)
		other := storedSetting(legacySettingID, user2ID, agencyA)
		other.SelectedAccountID = ptr(accountOther)
		f := newFakes(mine, other)

		got, err := f.app().SetSelectedAccount(withSession(userID, agencyA),
			bondUserSetting_domain.SetSelectedAccountParams{SelectedAccountID: " " + accountNew + " "})

		if err != nil {
			t.Fatalf("SetSelectedAccount: %v", err)
		}
		if got.ID != settingID || val(got.SelectedAccountID) != accountNew {
			t.Fatalf("SetSelectedAccount devolvió %s con la cuenta %s, se esperaba %s con %s", got.ID, val(got.SelectedAccountID), settingID, accountNew)
		}
		if want := []accountCall{{accountID: accountNew, userID: userID}}; !slices.Equal(f.balance.calls, want) {
			t.Errorf("GetAccount recibió %v, se esperaba %v", f.balance.calls, want)
		}
		if len(f.repo.upserts) != 1 {
			t.Fatalf("UpsertField se llamó %d veces, se esperaba 1", len(f.repo.upserts))
		}
		if u := f.repo.upserts[0]; u.setting.UserID != userID || u.field != bondUserSetting_ports.FieldSelectedAccountID {
			t.Errorf("UpsertField recibió el usuario %s y el campo %s, se esperaban %s y %s", u.setting.UserID, u.field, userID, bondUserSetting_ports.FieldSelectedAccountID)
		}
		if s := f.repo.settings[legacySettingID]; val(s.SelectedAccountID) != accountOther {
			t.Errorf("el setting de %s quedó con la cuenta %s; no debía cambiar", user2ID, val(s.SelectedAccountID))
		}
	})

	tests := []struct {
		name      string
		accountID string
		wantErr   error
		wantReads int
	}{
		{name: "cuenta de otro usuario", accountID: accountOther, wantErr: shared_domain.ErrAccountNotFound, wantReads: 1},
		{name: "cuenta inexistente", accountID: "account9AAAAAAAAA", wantErr: shared_domain.ErrAccountNotFound, wantReads: 1},
		{name: "id inválido", accountID: "abc", wantErr: shared_domain.ErrInvalidID, wantReads: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mine := storedSetting(settingID, userID, agencyA)
			mine.SelectedAccountID = ptr(accountOld)
			other := storedSetting(legacySettingID, user2ID, agencyA)
			other.SelectedAccountID = ptr(accountOther)
			f := newFakes(mine, other)

			got, err := f.app().SetSelectedAccount(withSession(userID, agencyA),
				bondUserSetting_domain.SetSelectedAccountParams{SelectedAccountID: tt.accountID})

			if !errors.Is(err, tt.wantErr) || got != nil {
				t.Fatalf("SetSelectedAccount(%q) = %v, %v; se esperaba nil, %v", tt.accountID, got, err, tt.wantErr)
			}
			if len(f.balance.calls) != tt.wantReads {
				t.Errorf("GetAccount se llamó %d veces, se esperaban %d", len(f.balance.calls), tt.wantReads)
			}
			if len(f.repo.upserts) != 0 {
				t.Fatalf("UpsertField se llamó con %+v sin una cuenta propia", f.repo.upserts)
			}
			if a, b := val(f.repo.settings[settingID].SelectedAccountID), val(f.repo.settings[legacySettingID].SelectedAccountID); a != accountOld || b != accountOther {
				t.Errorf("los settings quedaron con las cuentas %s y %s; no debían cambiar", a, b)
			}
		})
	}
}

// Sin setting previo, el cambio de cuenta lo crea con idioma "es" y la cuenta elegida. CA-09.6
func TestSetSelectedAccountWithoutSetting(t *testing.T) {
	f := newFakes()

	got, err := f.app().SetSelectedAccount(withSession(userID, agencyA),
		bondUserSetting_domain.SetSelectedAccountParams{SelectedAccountID: accountNew})

	if err != nil {
		t.Fatalf("SetSelectedAccount: %v", err)
	}
	if got.UserID != userID || got.AgencyID != agencyA || got.Language != bondUserSetting_domain.DefaultLanguage ||
		val(got.SelectedAccountID) != accountNew || got.SelectedBondBoundID != nil {
		t.Fatalf("SetSelectedAccount devolvió %+v, se esperaba un setting nuevo de %s con la cuenta %s", got, userID, accountNew)
	}
	if len(f.repo.settings) != 1 {
		t.Fatalf("quedaron %d settings, se esperaba 1", len(f.repo.settings))
	}
}
