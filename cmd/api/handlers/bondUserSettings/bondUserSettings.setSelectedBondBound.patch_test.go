package bondUserSettings_handlers_test

import (
	"net/http"
	"testing"
)

// PATCH /me/selected-bond-bound responde 200 con el setting; la aplicación recibe el bono del body, o nil
// con null. CA-09.6
func TestSetSelectedBondBound(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "bono", body: `{"selectedBondBoundId":"` + bondBoundID + `"}`, want: bondBoundID},
		{name: "null", body: `{"selectedBondBoundId":null}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{setting: sample()}

			rec := serve(newTestAPI(app, investor), http.MethodPatch, minePath+"/selected-bond-bound", tt.body)

			var got struct {
				ID string `json:"_id"`
			}
			decodeJSON(t, rec, http.StatusOK, &got)
			if got.ID != settingID {
				t.Errorf("body %s, se esperaba el setting %s", rec.Body, settingID)
			}
			received := app.bondBound.SelectedBondBoundID
			if tt.want == "" && received != nil {
				t.Fatalf("la aplicación recibió el bono %q, se esperaba nil", *received)
			}
			if tt.want != "" && (received == nil || *received != tt.want) {
				t.Fatalf("la aplicación recibió %v, se esperaba el bono %s", received, tt.want)
			}
		})
	}
}
