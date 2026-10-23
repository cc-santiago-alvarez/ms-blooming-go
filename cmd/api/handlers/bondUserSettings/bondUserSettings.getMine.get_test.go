package bondUserSettings_handlers_test

import (
	"net/http"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// GET /bond-user-settings/me de un usuario sin cuentas responde 200 con selectedAccountId y account en
// null. CA-09.2
func TestGetMineWithoutAccount(t *testing.T) {
	s := sample()
	s.SelectedAccountID = nil
	app := &fakeApp{detail: &bondUserSetting_domain.BondUserSettingDetail{BondUserSetting: *s}}

	rec := serve(newTestAPI(app, investor), http.MethodGet, minePath, "")

	var body map[string]any
	decodeJSON(t, rec, http.StatusOK, &body)
	if body["_id"] != settingID {
		t.Fatalf("body %s, se esperaba el setting %s", rec.Body, settingID)
	}
	for _, key := range []string{"selectedAccountId", "account"} {
		if v, ok := body[key]; !ok || v != nil {
			t.Errorf("%s = %v (presente: %t), se esperaba null", key, v, ok)
		}
	}
}
