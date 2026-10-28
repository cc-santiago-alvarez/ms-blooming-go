package bondBound_application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// El dueño y un admin que no es dueño reciben la cotización del bono a hoy, sin que se escriba nada, y otro
// usuario es ErrForbidden. Un bono inexistente o de otra agencia que la de la sesión es ErrBondBoundNotFound
// aunque la sesión sea admin o del dueño, y también sin sesión; el admin de la agencia principal cotiza el de
// cualquier agencia. CA-05.6, CA-05.18
func TestPreSaleAccess(t *testing.T) {
	owner := withSession(ownerID, -5)

	tests := []struct {
		name     string
		ctx      context.Context
		agencyID string
		id       string
		err      error
	}{
		{name: "dueño", ctx: owner, id: boundID},
		{name: "admin que no es dueño", ctx: withAdmin(), id: boundID},
		{name: "otro usuario", ctx: withSession(otherUserID, -5), id: boundID, err: shared_domain.ErrForbidden},
		{name: "sin sesión", ctx: context.Background(), id: boundID, err: bondBound_domain.ErrBondBoundNotFound},
		{name: "inexistente", ctx: owner, id: missingID, err: bondBound_domain.ErrBondBoundNotFound},
		{name: "inexistente, admin", ctx: withAdmin(), id: missingID, err: bondBound_domain.ErrBondBoundNotFound},
		{name: "de otra agencia, admin", ctx: withAdmin(), agencyID: agencyB, id: boundID, err: bondBound_domain.ErrBondBoundNotFound},
		{name: "de otra agencia, dueño", ctx: owner, agencyID: agencyB, id: boundID, err: bondBound_domain.ErrBondBoundNotFound},
		{name: "de otra agencia, otro usuario", ctx: withSession(otherUserID, -5), agencyID: agencyB, id: boundID, err: bondBound_domain.ErrBondBoundNotFound},
		{name: "de otra agencia, admin de la principal", ctx: withMainAdmin(), agencyID: agencyB, id: boundID},
		{name: "id inválido", ctx: owner, id: "abc", err: shared_domain.ErrInvalidID},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := boundOnDay(50)
			if tt.agencyID != "" {
				b.AgencyID = tt.agencyID
			}
			f := newFakes(b)

			projection, err := f.app().PreSale(tt.ctx, tt.id)

			if tt.err != nil {
				if !errors.Is(err, tt.err) {
					t.Fatalf("PreSale = %v, se esperaba %v", err, tt.err)
				}
				if tt.id == "abc" && len(f.repo.reads) != 0 {
					t.Errorf("FindOneByID se llamó con %v con un id inválido", f.repo.reads)
				}
				return
			}
			if err != nil {
				t.Fatalf("PreSale: %v", err)
			}
			want, err := bondBound_domain.QuoteSale(*b, time.Now(), -5)
			if err != nil {
				t.Fatalf("QuoteSale: %v", err)
			}
			if projection != want || projection.TotalDaysPassed != 50 {
				t.Errorf("PreSale devolvió %+v, se esperaba %+v con 50 días", projection, want)
			}
			assertNothingWritten(t, f)
		})
	}
}

// Antes del día mínimo de retiro la cotización es ErrInvalidWithdrawDate, también para un admin; el día mínimo
// ya se cotiza. CA-05.5
func TestPreSaleBeforeWithdrawDate(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		day  int
		err  error
	}{
		{name: "día 10, dueño", ctx: withSession(ownerID, -5), day: 10, err: bondBound_domain.ErrInvalidWithdrawDate},
		{name: "día 10, admin", ctx: withAdmin(), day: 10, err: bondBound_domain.ErrInvalidWithdrawDate},
		{name: "día 30", ctx: withSession(ownerID, -5), day: 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := boundOnDay(tt.day)
			b.EarlyCashTerm.InitialDayForWithdraw = 30

			projection, err := newFakes(b).app().PreSale(tt.ctx, boundID)

			if !errors.Is(err, tt.err) {
				t.Fatalf("PreSale = %v, se esperaba %v", err, tt.err)
			}
			if tt.err == nil && projection.TotalDaysPassed != tt.day {
				t.Errorf("PreSale devolvió %+v, se esperaban %d días", projection, tt.day)
			}
		})
	}
}

// La cotización de un bono bloqueado, desligado, vendido o canjeado da el mismo error que la venta. CA-05.9
func TestPreSaleNotSellable(t *testing.T) {
	for _, tt := range notSellableCases() {
		t.Run(tt.name, func(t *testing.T) {
			b := boundOnDay(50)
			tt.modify(b)

			_, err := newFakes(b).app().PreSale(withSession(ownerID, -5), boundID)

			if !errors.Is(err, tt.err) {
				t.Fatalf("PreSale = %v, se esperaba %v", err, tt.err)
			}
		})
	}
}
