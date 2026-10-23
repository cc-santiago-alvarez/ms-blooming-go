package bondUserSetting_application_test

import (
	"context"
	"testing"
)

// GetByUser de un usuario sin setting, o con el setting borrado, devuelve nil sin crear nada; con setting lo
// devuelve. CA-09.11
func TestGetByUser(t *testing.T) {
	removed := storedSetting(legacySettingID, user2ID, agencyA)
	removed.IsRemove = true

	tests := []struct {
		name   string
		userID string
		wantID string
	}{
		{name: "sin setting", userID: user3ID},
		{name: "con el setting borrado", userID: user2ID},
		{name: "con setting", userID: userID, wantID: settingID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes(storedSetting(settingID, userID, agencyA), removed)

			got, err := f.app().GetByUser(context.Background(), tt.userID)

			if err != nil {
				t.Fatalf("GetByUser(%q): %v", tt.userID, err)
			}
			if tt.wantID == "" && got != nil {
				t.Fatalf("GetByUser(%q) devolvió %+v, se esperaba nil", tt.userID, got)
			}
			if tt.wantID != "" && (got == nil || got.ID != tt.wantID) {
				t.Fatalf("GetByUser(%q) devolvió %+v, se esperaba el setting %s", tt.userID, got, tt.wantID)
			}
			if len(f.repo.defaults) != 0 || len(f.repo.upserts) != 0 || len(f.repo.settings) != 2 {
				t.Fatalf("GetByUser creó o cambió settings: %d defaults, %d upserts y %d settings", len(f.repo.defaults), len(f.repo.upserts), len(f.repo.settings))
			}
		})
	}
}
