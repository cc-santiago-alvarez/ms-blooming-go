package bondType_application_test

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"testing"
)

// Cada tipo activo de la agencia de la sesión trae numOfBonds del contador, que se consulta una sola vez
// con todos los tipos y el usuario de la sesión; un tipo que el contador no nombra tiene 0. CA-01.3
func TestListActiveNumOfBonds(t *testing.T) {
	t.Run("con tipos", func(t *testing.T) {
		repo := newRepository()
		repo.found = append(repo.found, storedType(typeID), storedType(otherTypeID))
		counter := &fakeCounter{counts: map[string]int{typeID: 2}}

		types, err := newApp(repo, counter).ListActive(withSession("user-1", agencyA))

		if err != nil {
			t.Fatalf("ListActive: %v", err)
		}
		if len(repo.finds) != 1 {
			t.Fatalf("Find se llamó %d veces, se esperaba 1", len(repo.finds))
		}
		if got := repo.finds[0]; !slices.Equal(got.agencyIDs, []string{agencyA}) || !got.onlyActive {
			t.Fatalf("Find(%v, %v), se esperaba Find([%s], true)", got.agencyIDs, got.onlyActive, agencyA)
		}
		if len(counter.countCalls) != 1 {
			t.Fatalf("el contador se llamó %d veces, se esperaba 1", len(counter.countCalls))
		}
		if got := counter.countCalls[0]; !slices.Equal(got.ids, []string{typeID, otherTypeID}) || got.userID != "user-1" {
			t.Fatalf("CountActiveByType(%v, %q), se esperaba ([%s %s], \"user-1\")", got.ids, got.userID, typeID, otherTypeID)
		}
		if len(types) != 2 {
			t.Fatalf("ListActive devolvió %d tipos, se esperaban 2", len(types))
		}
		if types[0].ID != typeID || types[0].NumOfBonds != 2 {
			t.Errorf("primer tipo %s con numOfBonds %d, se esperaba %s con 2", types[0].ID, types[0].NumOfBonds, typeID)
		}
		if types[1].ID != otherTypeID || types[1].NumOfBonds != 0 {
			t.Errorf("segundo tipo %s con numOfBonds %d, se esperaba %s con 0", types[1].ID, types[1].NumOfBonds, otherTypeID)
		}
		if types[0].Name != "Bono Plus" || !types[0].CreatedAt.Equal(storedAt) {
			t.Errorf("el tipo no conserva los campos del repositorio: %+v", types[0].BondType)
		}
	})

	empty := []struct {
		name     string
		ctx      context.Context
		wantFind bool
	}{
		{name: "sin sesión", ctx: context.Background()},
		{name: "sesión sin agencia", ctx: withSession("user-1", "")},
		{name: "sin tipos", ctx: withSession("user-1", agencyA), wantFind: true},
	}
	for _, tt := range empty {
		t.Run(tt.name, func(t *testing.T) {
			repo := newRepository()
			counter := &fakeCounter{}

			types, err := newApp(repo, counter).ListActive(tt.ctx)

			if err != nil {
				t.Fatalf("ListActive: %v", err)
			}
			raw, err := json.Marshal(types)
			if err != nil {
				t.Fatalf("json.Marshal: %v", err)
			}
			if string(raw) != "[]" {
				t.Fatalf("ListActive = %s, se esperaba []", raw)
			}
			if tt.wantFind != (len(repo.finds) == 1) {
				t.Errorf("Find se llamó %d veces", len(repo.finds))
			}
			if len(counter.countCalls) != 0 {
				t.Errorf("el contador se llamó %d veces, se esperaba 0", len(counter.countCalls))
			}
		})
	}

	t.Run("error del contador", func(t *testing.T) {
		repo := newRepository()
		repo.found = append(repo.found, storedType(typeID))
		counterErr := errors.New("aggregate falló")

		types, err := newApp(repo, &fakeCounter{err: counterErr}).ListActive(withSession("user-1", agencyA))

		if !errors.Is(err, counterErr) || types != nil {
			t.Fatalf("ListActive = %v, %v; se esperaba nil y el error del contador", types, err)
		}
	})
}
