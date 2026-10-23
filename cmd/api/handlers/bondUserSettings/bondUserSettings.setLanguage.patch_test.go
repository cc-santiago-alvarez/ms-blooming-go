package bondUserSettings_handlers_test

import (
	"net/http"
	"testing"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
)

// PATCH /me/language responde 200 con el setting y pasa el idioma del body a la aplicación. CA-09.6
func TestSetLanguage(t *testing.T) {
	updated := sample()
	updated.Language = "en"
	app := &fakeApp{setting: updated}

	rec := serve(newTestAPI(app, investor), http.MethodPatch, minePath+"/language", `{"language":"en"}`)

	var got struct {
		ID       string `json:"_id"`
		Language string `json:"language"`
	}
	decodeJSON(t, rec, http.StatusOK, &got)
	if got.ID != settingID || got.Language != "en" {
		t.Errorf("body %s, se esperaba el setting %s en \"en\"", rec.Body, settingID)
	}
	if app.language.Language != "en" {
		t.Fatalf("la aplicación recibió %q, se esperaba \"en\"", app.language.Language)
	}
}

// Un idioma inválido responde 400 ERR_INVALID_LANGUAGE. CA-09.7
func TestSetLanguageInvalid(t *testing.T) {
	app := &fakeApp{err: bondUserSetting_domain.ErrInvalidLanguage}

	rec := serve(newTestAPI(app, investor), http.MethodPatch, minePath+"/language", `{"language":"español"}`)

	assertError(t, rec, http.StatusBadRequest, "ERR_INVALID_LANGUAGE")
	if app.language.Language != "español" {
		t.Fatalf("la aplicación recibió %q, se esperaba \"español\"", app.language.Language)
	}
}
