package bondUserSetting_application_test

import (
	"slices"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// Sin setting previo, elegir un bono crea el setting con ese bono y la cuenta más antigua del usuario. Con
// setting, cambia solo el bono, sin buscar la cuenta más antigua, y null lo deselecciona. CA-09.6
func TestSetSelectedBondBound(t *testing.T) {
	t.Run("sin setting", func(t *testing.T) {
		f := newFakes()

		got, err := f.app().SetSelectedBondBound(withSession(userID, agencyA),
			bondUserSetting_domain.SetSelectedBondBoundParams{SelectedBondBoundID: ptr(bondBoundID)})

		if err != nil {
			t.Fatalf("SetSelectedBondBound: %v", err)
		}
		if got.UserID != userID || got.Language != bondUserSetting_domain.DefaultLanguage ||
			val(got.SelectedBondBoundID) != bondBoundID || val(got.SelectedAccountID) != accountOld {
			t.Fatalf("SetSelectedBondBound devolvió %+v, se esperaba un setting nuevo con el bono %s y la cuenta %s", got, bondBoundID, accountOld)
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
		mine.Language = "en"
		mine.SelectedAccountID = ptr(accountNew)
		f := newFakes(mine)

		got, err := f.app().SetSelectedBondBound(withSession(userID, agencyA),
			bondUserSetting_domain.SetSelectedBondBoundParams{SelectedBondBoundID: ptr(bondBoundID)})

		if err != nil {
			t.Fatalf("SetSelectedBondBound: %v", err)
		}
		if got.ID != settingID || val(got.SelectedBondBoundID) != bondBoundID ||
			val(got.SelectedAccountID) != accountNew || got.Language != "en" {
			t.Fatalf("SetSelectedBondBound devolvió %+v, se esperaba %s con el bono %s y los demás campos sin cambios", got, settingID, bondBoundID)
		}
		if len(f.firstAccount.calls) != 0 {
			t.Errorf("FirstAccountID se llamó con %v teniendo setting", f.firstAccount.calls)
		}
	})

	t.Run("null", func(t *testing.T) {
		mine := storedSetting(settingID, userID, agencyA)
		mine.SelectedBondBoundID = ptr(bondBoundID)
		f := newFakes(mine)

		got, err := f.app().SetSelectedBondBound(withSession(userID, agencyA),
			bondUserSetting_domain.SetSelectedBondBoundParams{SelectedBondBoundID: nil})

		if err != nil {
			t.Fatalf("SetSelectedBondBound: %v", err)
		}
		if got.SelectedBondBoundID != nil || f.repo.settings[settingID].SelectedBondBoundID != nil {
			t.Fatalf("selectedBondBoundId quedó en %s, se esperaba nil", val(f.repo.settings[settingID].SelectedBondBoundID))
		}
	})
}
