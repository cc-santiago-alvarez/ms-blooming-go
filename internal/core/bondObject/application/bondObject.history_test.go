package bondObject_application_test

import (
	"errors"
	"strings"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Si History.Add falla, el alta, la edición y la baja responden con éxito igual, la escritura queda hecha
// y el error queda en el log; el alta registra la activity de todos modos. CA-02.14
func TestHistoryIsBestEffort(t *testing.T) {
	historyErr := errors.New("historial caído")

	tests := []struct {
		name   string
		text   string
		run    func(f *fakes) error
		stored func(f *fakes) bool
	}{
		{
			name: "alta",
			text: shared_domain.TextBondAdded,
			run: func(f *fakes) error {
				_, err := f.app().Create(withAdmin(agencyMain, true), createParams())
				return err
			},
			stored: func(f *fakes) bool { return len(f.repo.inserted) == 1 && len(f.activity.added) == 1 },
		},
		{
			name: "edición",
			text: shared_domain.TextBondUpdated,
			run: func(f *fakes) error {
				params := paramsFrom(storedObject(objectID))
				params.Amount = 2000
				_, err := f.app().Update(withAdmin(agencyMain, true), objectID, params)
				return err
			},
			stored: func(f *fakes) bool { return f.repo.objects[objectID].Amount == 2000 },
		},
		{
			name: "baja",
			text: shared_domain.TextBondRemove,
			run: func(f *fakes) error {
				return f.app().Remove(withAdmin(agencyMain, true), objectID)
			},
			stored: func(f *fakes) bool { return f.repo.objects[objectID].IsRemove },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLog(t)
			f := newFakes(storedObject(objectID))
			f.history.err = historyErr

			if err := tt.run(f); err != nil {
				t.Fatalf("la %s devolvió %v, se esperaba éxito aunque falle el historial", tt.name, err)
			}
			if !tt.stored(f) {
				t.Errorf("la %s no quedó guardada cuando falló el historial", tt.name)
			}
			if len(f.history.added) != 1 {
				t.Fatalf("History.Add se llamó %d veces, se esperaba 1", len(f.history.added))
			}
			if out := logs.String(); !strings.Contains(out, tt.text) || !strings.Contains(out, historyErr.Error()) {
				t.Errorf("log %q, se esperaba %s y el error %q", out, tt.text, historyErr)
			}
		})
	}
}
