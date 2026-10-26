package bondBound_application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Los recientes consultan los 5 bonos del usuario de la sesión, y cada uno trae el nombre y los términos de
// retiro de su producto, con lo acumulado a hoy. CA-06.9
func TestListRecent(t *testing.T) {
	f := newFakes()
	term := shared_domain.EarlyCashTerm{InitialPenaltyPercentage: 100, InflectionPoint: 90, InflectionPenaltyPercentage: 75, InitialDayForWithdraw: 30}
	b := boundOnDay(40)
	f.repo.recent = []bondBound_ports.BondBoundWithObject{
		{Bound: *b, BondObject: &bondBound_ports.BondObjectRef{Name: "Bono Plus 12M", EarlyCashTerm: &term}},
	}

	items, err := f.app().ListRecent(withSession(ownerID, -5))
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}

	want := []repoQuery{{method: "ListRecent", userID: ownerID, limit: 5}}
	if !reflect.DeepEqual(f.repo.queries, want) {
		t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, want)
	}
	if len(items) != 1 {
		t.Fatalf("ListRecent devolvió %d ítems, se esperaba 1", len(items))
	}
	item := items[0]
	if item.BondObject == nil || item.BondObject.Name != "Bono Plus 12M" || item.BondObject.EarlyCashTerm == nil || *item.BondObject.EarlyCashTerm != term {
		t.Errorf("bondObject %+v, se esperaba el nombre y los términos de retiro del producto", item.BondObject)
	}
	// 1150 / 365 × 40, redondeado
	if item.ID != b.ID || item.DaysPassed != 40 || item.AcumulateAmount != 126.03 || !item.Growing {
		t.Errorf("_id %q, daysPassed %d, acumulado %v y growing %v, se esperaban %q, 40, 126.03 y true",
			item.ID, item.DaysPassed, item.AcumulateAmount, item.Growing, b.ID)
	}
}

// Sin sesión es un error y no consulta; un error del repositorio se devuelve sin cambiarlo. CA-06.9
func TestListRecentErrors(t *testing.T) {
	t.Run("sin sesión", func(t *testing.T) {
		f := newFakes()

		items, err := f.app().ListRecent(context.Background())

		if err == nil || items != nil {
			t.Fatalf("ListRecent = %+v, %v, se esperaba un error", items, err)
		}
		assertNoQueries(t, f)
	})

	t.Run("error del repositorio", func(t *testing.T) {
		f := newFakes()
		f.repo.err = errors.New("mongo caído")

		_, err := f.app().ListRecent(withSession(ownerID, -5))

		if !errors.Is(err, f.repo.err) {
			t.Fatalf("ListRecent = %v, se esperaba el error del repositorio", err)
		}
	})
}
