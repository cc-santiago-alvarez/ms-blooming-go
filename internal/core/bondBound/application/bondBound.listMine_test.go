package bondBound_application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// mineRow es el bono b con un producto de nombre "Bono Plus 12M".
func mineRow(b *bondBound_domain.BondBound) bondBound_ports.BondBoundWithObject {
	return bondBound_ports.BondBoundWithObject{Bound: *b, BondObject: &bondBound_ports.BondObjectRef{Name: "Bono Plus 12M"}}
}

// minePage es la página que responde el repositorio con los bonos dados y el total.
func minePage(total int64, bounds ...*bondBound_domain.BondBound) shared_domain.Paginated[bondBound_ports.BondBoundWithObject] {
	rows := make([]bondBound_ports.BondBoundWithObject, len(bounds))
	for i, b := range bounds {
		rows[i] = mineRow(b)
	}
	return shared_domain.Paginated[bondBound_ports.BondBoundWithObject]{Items: rows, Total: total}
}

// Mis bonos consulta con el usuario y la agencia de la sesión, los estados recortados y sin repetir y la
// página pedida; con 12 bonos y 5 por página son 3 páginas. CA-06.8
func TestListMineQuery(t *testing.T) {
	f := newFakes()
	f.repo.minePage = minePage(12, boundOnDay(40))
	filters := bondBound_domain.MineFilters{StatusIDs: []bondBound_domain.BondBoundStatus{" 1", "2", "1", ""}}

	page, err := f.app().ListMine(withSession(ownerID, -5), filters, shared_domain.Pagination{Page: 2, Limit: 5})
	if err != nil {
		t.Fatalf("ListMine: %v", err)
	}

	want := []repoQuery{{
		method:    "ListMine",
		userID:    ownerID,
		agencyID:  agency(agencyA),
		statusIDs: []bondBound_domain.BondBoundStatus{bondBound_domain.StatusInitiated, bondBound_domain.StatusFinalized},
		page:      shared_domain.Pagination{Page: 2, Limit: 5},
	}}
	if !reflect.DeepEqual(f.repo.queries, want) {
		t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, want)
	}
	if page.Total != 12 || page.Page != 2 || page.Limit != 5 || page.TotalPages != 3 || len(page.Items) != 1 {
		t.Errorf("total %d, page %d, limit %d, totalPages %d y %d ítems, se esperaban 12, 2, 5, 3 y 1",
			page.Total, page.Page, page.Limit, page.TotalPages, len(page.Items))
	}
}

// Cada bono trae su producto, lo acumulado a hoy y si está creciendo, desde el día 30 hasta el último del
// plazo, o ya creció. CA-06.8
func TestListMineItems(t *testing.T) {
	tests := []struct {
		name    string
		days    int
		amount  float64
		growing bool
		grew    bool
	}{
		// 1150 / 365 × 40, redondeado
		{name: "día 40", days: 40, amount: 126.03, growing: true, grew: false},
		{name: "día 400, pasado el plazo", days: 400, amount: 0, growing: false, grew: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			b := boundOnDay(tt.days)
			f.repo.minePage = minePage(1, b)

			page, err := f.app().ListMine(withSession(ownerID, -5), bondBound_domain.MineFilters{}, shared_domain.Pagination{})
			if err != nil {
				t.Fatalf("ListMine: %v", err)
			}

			if len(page.Items) != 1 {
				t.Fatalf("ListMine devolvió %d ítems, se esperaba 1", len(page.Items))
			}
			item := page.Items[0]
			if item.BondObject == nil || item.BondObject.Name != "Bono Plus 12M" {
				t.Errorf("bondObject %+v, se esperaba el producto del bono", item.BondObject)
			}
			if item.ID != b.ID || item.StatusID != bondBound_domain.StatusInitiated || item.BondObjectID != objectID {
				t.Errorf("_id %q, statusId %q y bondObjectId %q, se esperaban los del bono", item.ID, item.StatusID, item.BondObjectID)
			}
			if item.DaysPassed != tt.days || item.AcumulateAmount != tt.amount {
				t.Errorf("daysPassed %d y acumulado %v, se esperaban %d y %v", item.DaysPassed, item.AcumulateAmount, tt.days, tt.amount)
			}
			if item.Growing != tt.growing || item.Grew != tt.grew {
				t.Errorf("growing %v y grew %v, se esperaban %v y %v", item.Growing, item.Grew, tt.growing, tt.grew)
			}
		})
	}
}

// Las fechas inicial y de vencimiento salen como "D MMM YYYY" en la zona de la sesión; una fecha que falta
// no se formatea, y un bono sin producto sale sin bondObject. CA-06.8
func TestListMineFormattedDates(t *testing.T) {
	tests := []struct {
		name         string
		utcDiff      int
		initial      string
		maturity     string
		withMaturity bool
	}{
		{name: "UTC-5", utcDiff: -5, initial: "1 Jan 2026", maturity: "1 Jan 2027", withMaturity: true},
		{name: "UTC+14, ya es 2 de enero", utcDiff: 14, initial: "2 Jan 2026", maturity: "2 Jan 2027", withMaturity: true},
		{name: "sin fecha de vencimiento", utcDiff: -5, initial: "1 Jan 2026", maturity: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			b := storedBound(boundID)
			if !tt.withMaturity {
				b.MaturityDate = time.Time{}
			}
			f.repo.minePage = shared_domain.Paginated[bondBound_ports.BondBoundWithObject]{
				Items: []bondBound_ports.BondBoundWithObject{{Bound: *b}},
				Total: 1,
			}

			page, err := f.app().ListMine(withSession(ownerID, tt.utcDiff), bondBound_domain.MineFilters{}, shared_domain.Pagination{})
			if err != nil {
				t.Fatalf("ListMine: %v", err)
			}

			item := page.Items[0]
			if item.InitialDateFormatted != tt.initial || item.MaturityDateFormatted != tt.maturity {
				t.Errorf("fechas %q y %q, se esperaban %q y %q",
					item.InitialDateFormatted, item.MaturityDateFormatted, tt.initial, tt.maturity)
			}
			if item.BondObject != nil {
				t.Errorf("bondObject %+v, se esperaba nil", item.BondObject)
			}
		})
	}
}

// Un estado desconocido es ErrInvalidStatus y no consulta. CA-06.8
func TestListMineInvalidStatus(t *testing.T) {
	f := newFakes()
	filters := bondBound_domain.MineFilters{StatusIDs: []bondBound_domain.BondBoundStatus{"1", "9"}}

	page, err := f.app().ListMine(withSession(ownerID, -5), filters, shared_domain.Pagination{})

	if !errors.Is(err, bondBound_domain.ErrInvalidStatus) || page.Items != nil {
		t.Fatalf("ListMine = %+v, %v, se esperaba ErrInvalidStatus", page, err)
	}
	assertNoQueries(t, f)
}

// Sin sesión es un error y no consulta; un error del repositorio se devuelve sin cambiarlo. CA-06.8
func TestListMineErrors(t *testing.T) {
	t.Run("sin sesión", func(t *testing.T) {
		f := newFakes()

		_, err := f.app().ListMine(context.Background(), bondBound_domain.MineFilters{}, shared_domain.Pagination{})

		if err == nil {
			t.Fatal("ListMine sin sesión no devolvió error")
		}
		assertNoQueries(t, f)
	})

	t.Run("error del repositorio", func(t *testing.T) {
		f := newFakes()
		f.repo.err = errors.New("mongo caído")

		_, err := f.app().ListMine(withSession(ownerID, -5), bondBound_domain.MineFilters{}, shared_domain.Pagination{})

		if !errors.Is(err, f.repo.err) {
			t.Fatalf("ListMine = %v, se esperaba el error del repositorio", err)
		}
	})
}
