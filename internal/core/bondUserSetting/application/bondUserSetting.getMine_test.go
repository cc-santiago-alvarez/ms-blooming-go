package bondUserSetting_application_test

import (
	"slices"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// Un usuario sin setting y sin cuentas recibe un setting nuevo con idioma "es", sin cuenta ni bono
// seleccionados, y el detalle sale sin cuenta y sin consultar el saldo. CA-09.2
func TestGetMineWithoutAccounts(t *testing.T) {
	f := newFakes()
	f.firstAccount.oldest = map[string]string{}

	got, err := f.app().GetMine(withSession(user3ID, agencyA))

	if err != nil {
		t.Fatalf("GetMine: %v", err)
	}
	if got.SelectedAccountID != nil || got.Account != nil {
		t.Fatalf("selectedAccountId %s y account %+v, se esperaban nil", val(got.SelectedAccountID), got.Account)
	}
	if len(f.repo.defaults) != 1 {
		t.Fatalf("UpsertDefault se llamó %d veces, se esperaba 1", len(f.repo.defaults))
	}
	d := f.repo.defaults[0]
	if d.UserID != user3ID || d.AgencyID != agencyA || d.Language != bondUserSetting_domain.DefaultLanguage ||
		d.SelectedAccountID != nil || d.SelectedBondBoundID != nil {
		t.Errorf("UpsertDefault recibió %+v, se esperaba el setting por defecto de %s en %s", d, user3ID, agencyA)
	}
	if got.ID != d.ID {
		t.Errorf("GetMine devolvió el setting %s, se esperaba el creado %s", got.ID, d.ID)
	}
	if want := []ownerCall{{userID: user3ID, agencyID: agencyA}}; !slices.Equal(f.firstAccount.calls, want) {
		t.Errorf("FirstAccountID recibió %v, se esperaba %v", f.firstAccount.calls, want)
	}
	if len(f.balance.calls) != 0 {
		t.Errorf("GetAccount se llamó con %v sin cuenta seleccionada", f.balance.calls)
	}
}
