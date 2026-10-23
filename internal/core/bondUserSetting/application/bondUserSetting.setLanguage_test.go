package bondUserSetting_application_test

import (
	"errors"
	"slices"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// Sin setting previo, cambiar el idioma crea el setting con ese idioma y la cuenta más antigua del usuario;
// con setting, cambia solo el idioma. CA-09.6
func TestSetLanguage(t *testing.T) {
	t.Run("sin setting", func(t *testing.T) {
		f := newFakes()

		got, err := f.app().SetLanguage(withSession(userID, agencyA), bondUserSetting_domain.SetLanguageParams{Language: "en"})

		if err != nil {
			t.Fatalf("SetLanguage: %v", err)
		}
		if got.UserID != userID || got.Language != "en" || val(got.SelectedAccountID) != accountOld || got.SelectedBondBoundID != nil {
			t.Fatalf("SetLanguage devolvió %+v, se esperaba un setting nuevo en \"en\" con la cuenta %s", got, accountOld)
		}
		if want := []ownerCall{{userID: userID, agencyID: agencyA}}; !slices.Equal(f.firstAccount.calls, want) {
			t.Errorf("FirstAccountID recibió %v, se esperaba %v", f.firstAccount.calls, want)
		}
		if len(f.repo.settings) != 1 {
			t.Fatalf("quedaron %d settings, se esperaba 1", len(f.repo.settings))
		}
	})

	t.Run("con setting", func(t *testing.T) {
		mine := storedSetting(settingID, userID, agencyA)
		mine.SelectedAccountID = ptr(accountNew)
		mine.SelectedBondBoundID = ptr(bondBoundID)
		f := newFakes(mine)

		got, err := f.app().SetLanguage(withSession(userID, agencyA), bondUserSetting_domain.SetLanguageParams{Language: "en"})

		if err != nil {
			t.Fatalf("SetLanguage: %v", err)
		}
		if got.ID != settingID || got.Language != "en" || val(got.SelectedAccountID) != accountNew || val(got.SelectedBondBoundID) != bondBoundID {
			t.Fatalf("SetLanguage devolvió %+v, se esperaba %s en \"en\" y los demás campos sin cambios", got, settingID)
		}
		if len(f.firstAccount.calls) != 0 {
			t.Errorf("FirstAccountID se llamó con %v teniendo setting", f.firstAccount.calls)
		}
	})
}

// Un idioma vacío o de más de 5 caracteres es ErrInvalidLanguage y no llega al repositorio; uno de 5
// caracteres, o con espacios alrededor, se acepta recortado. CA-09.7
func TestSetLanguageValidation(t *testing.T) {
	tests := []struct {
		name     string
		language string
		wantErr  error
		want     string
	}{
		{name: "vacío", language: "", wantErr: bondUserSetting_domain.ErrInvalidLanguage},
		{name: "solo espacios", language: "   ", wantErr: bondUserSetting_domain.ErrInvalidLanguage},
		{name: "más de 5 caracteres", language: "español", wantErr: bondUserSetting_domain.ErrInvalidLanguage},
		{name: "5 caracteres", language: "es-CO", want: "es-CO"},
		{name: "5 caracteres con ñ", language: "españ", want: "españ"},
		{name: "con espacios alrededor", language: " en ", want: "en"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(storedSetting(settingID, userID, agencyA))

			got, err := f.app().SetLanguage(withSession(userID, agencyA), bondUserSetting_domain.SetLanguageParams{Language: tt.language})

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || got != nil {
					t.Fatalf("SetLanguage(%q) = %v, %v; se esperaba nil, %v", tt.language, got, err, tt.wantErr)
				}
				if len(f.repo.userReads) != 0 || len(f.repo.upserts) != 0 {
					t.Fatalf("el repositorio se usó con un idioma inválido: %d lecturas y %d upserts", len(f.repo.userReads), len(f.repo.upserts))
				}
				return
			}
			if err != nil {
				t.Fatalf("SetLanguage(%q): %v", tt.language, err)
			}
			if got.Language != tt.want {
				t.Fatalf("SetLanguage(%q) guardó %q, se esperaba %q", tt.language, got.Language, tt.want)
			}
		})
	}
}
