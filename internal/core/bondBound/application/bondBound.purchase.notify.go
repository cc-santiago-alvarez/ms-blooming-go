package bondBound_application

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Grupos de destinatarios de ms-notification.
const (
	groupUserIDs   = "user-ids"
	groupRawEmails = "to-raw-emails"
)

// Notificaciones de ms-notification que manda la compra.
const (
	notifyBondPurchased     = "notify-blooming-bondPurchased"
	notifyBondBulkPurchased = "notify-blooming-bondBulkPurchased"
	notifyWithdrawUser      = "notify-blooming-withdrawUser"
	notifyWithdrawAdmin     = "notify-blooming-withdrawAdmin"
)

// adminDateLayout es el formato de la fecha del aviso de retiro al área administrativa.
const adminDateLayout = "2006-01-02 15:04:05"

// boundRowHTML es la fila de un bono en la tabla del aviso de compra masiva: número, clave, referencia del
// cobro y monto.
const boundRowHTML = `<tr>
      <td style="border: solid 1px #d8d8d8;">%d</td>
      <td style="border: solid 1px #d8d8d8;">%s</td>
      <td style="border: solid 1px #d8d8d8;">%s</td>
      <td style="border: solid 1px #d8d8d8;">$ %s</td>
      </tr>`

// purchaseSummary son los datos ya formateados de una compra confirmada que comparten sus avisos.
type purchaseSummary struct {
	userID        string
	agencyID      string
	firstName     string
	lastName      string
	accountMasked string
	unitPrice     string
	total         string
	keys          []string
	references    []string
	date          string
}

// fullName es "nombre apellido", sin espacios sobrantes.
func (s purchaseSummary) fullName() string {
	return strings.TrimSpace(s.firstName + " " + s.lastName)
}

// notifyPurchase avisa la compra de un bono: guarda el aviso del bono y el del retiro y, en segundo plano,
// manda el email del bono y los del retiro al comprador y al área administrativa. Ningún fallo llega al
// llamador.
func (app *BondBoundApplication) notifyPurchase(ctx context.Context, session *session_domain.UserSession, order *purchaseOrder, bounds []*bondBound_domain.BondBound, movements []*shared_domain.BalanceMovement, now time.Time) {
	ctx = context.WithoutCancel(ctx)
	s := app.summarize(ctx, session, order, bounds, movements, now)
	b := bounds[0]

	app.Notices.Add(ctx, shared_domain.Notice{
		Type:     shared_domain.NoticeTypeBond,
		UserID:   s.userID,
		AgencyID: s.agencyID,
		Title:    shared_domain.Format(shared_domain.Text(shared_domain.TextBondBoundSubjectPurchased), b.Key, s.total),
		Message: shared_domain.Format(shared_domain.Text(shared_domain.TextBondBoundMessagePurchased),
			html.EscapeString(s.firstName), html.EscapeString(s.lastName), html.EscapeString(b.Key), s.total,
			html.EscapeString(s.accountMasked)),
	})
	app.Notices.Add(ctx, shared_domain.Notice{
		Type:     shared_domain.NoticeTypeBalance,
		UserID:   s.userID,
		AgencyID: s.agencyID,
		Title:    shared_domain.Text(shared_domain.TextNewWithdrawal),
		Message: shared_domain.Format(shared_domain.Text(shared_domain.TextBalanceAccountWithdrawContent),
			html.EscapeString(s.fullName()), html.EscapeString(s.accountMasked), s.total),
	})

	go func() {
		app.Notifications.Send(ctx, groupUserIDs, notifyBondPurchased, map[string]any{
			"userIds":             []string{s.userID},
			"key":                 b.Key,
			"amount":              s.total,
			"accountNumberMasked": s.accountMasked,
		})
		app.sendWithdrawEmails(ctx, s)
	}()
}

// notifyBulkPurchase avisa una compra masiva: guarda el aviso de los bonos, con una fila por bono, y el del
// retiro y, en segundo plano, manda el email de los bonos y los del retiro al comprador y al área
// administrativa. Ningún fallo llega al llamador.
func (app *BondBoundApplication) notifyBulkPurchase(ctx context.Context, session *session_domain.UserSession, order *purchaseOrder, bounds []*bondBound_domain.BondBound, movements []*shared_domain.BalanceMovement, now time.Time) {
	ctx = context.WithoutCancel(ctx)
	s := app.summarize(ctx, session, order, bounds, movements, now)
	quantity := len(bounds)

	var tbody strings.Builder
	rows := make([]map[string]any, len(bounds))
	for i, b := range bounds {
		amount := shared_domain.FormatCurrency(b.Amount)
		fmt.Fprintf(&tbody, boundRowHTML, i+1, html.EscapeString(b.Key), html.EscapeString(s.references[i]), amount)
		rows[i] = map[string]any{"key": b.Key, "reference": s.references[i], "amount": amount}
	}

	app.Notices.Add(ctx, shared_domain.Notice{
		Type:     shared_domain.NoticeTypeBond,
		UserID:   s.userID,
		AgencyID: s.agencyID,
		Title:    shared_domain.Format(shared_domain.Text(shared_domain.TextBondBoundSubjectMassivePurchased), quantity, s.total),
		Message: shared_domain.Format(shared_domain.Text(shared_domain.TextBondBoundMessageMassivePurchased),
			html.EscapeString(s.fullName()), quantity, s.total, s.unitPrice, html.EscapeString(s.accountMasked),
			tbody.String()),
	})
	app.Notices.Add(ctx, shared_domain.Notice{
		Type:     shared_domain.NoticeTypeBalance,
		UserID:   s.userID,
		AgencyID: s.agencyID,
		Title:    shared_domain.Text(shared_domain.TextNewWithdrawal),
		Message: shared_domain.Format(shared_domain.Text(shared_domain.TextBalanceAccountWithdrawForBonus),
			html.EscapeString(s.fullName()), html.EscapeString(s.accountMasked), s.total,
			html.EscapeString(strings.Join(s.keys, ", ")), s.unitPrice),
	})

	go func() {
		app.Notifications.Send(ctx, groupUserIDs, notifyBondBulkPurchased, map[string]any{
			"userIds":             []string{s.userID},
			"quantity":            quantity,
			"total":               s.total,
			"unitPrice":           s.unitPrice,
			"accountNumberMasked": s.accountMasked,
			"rows":                rows,
		})
		app.sendWithdrawEmails(ctx, s)
	}()
}

// sendWithdrawEmails manda el email del retiro al comprador y, si hay destinatarios, al área
// administrativa. Una compra individual se informa como una compra de un bono.
func (app *BondBoundApplication) sendWithdrawEmails(ctx context.Context, s purchaseSummary) {
	app.Notifications.Send(ctx, groupUserIDs, notifyWithdrawUser, map[string]any{
		"userIds":             []string{s.userID},
		"accountNumberMasked": s.accountMasked,
		"amount":              s.total,
		"keys":                s.keys,
		"unitPrice":           s.unitPrice,
	})

	if len(app.AdminEmails) == 0 {
		return
	}
	app.Notifications.Send(ctx, groupRawEmails, notifyWithdrawAdmin, map[string]any{
		"emails":              app.AdminEmails,
		"userName":            s.fullName(),
		"accountNumberMasked": s.accountMasked,
		"amount":              s.total,
		"references":          s.references,
		"date":                s.date,
		"keys":                s.keys,
		"unitPrice":           s.unitPrice,
	})
}

// summarize arma los datos de los avisos de una compra confirmada: el nombre del comprador, la cuenta
// enmascarada, los montos formateados, las claves y referencias en orden y la fecha en la zona de la sesión.
func (app *BondBoundApplication) summarize(ctx context.Context, session *session_domain.UserSession, order *purchaseOrder, bounds []*bondBound_domain.BondBound, movements []*shared_domain.BalanceMovement, now time.Time) purchaseSummary {
	firstName, lastName := app.buyerName(ctx, session)
	s := purchaseSummary{
		userID:        order.buyer.UserID,
		agencyID:      order.buyer.AgencyID,
		firstName:     firstName,
		lastName:      lastName,
		accountMasked: shared_domain.MaskAccountNumber(order.account.Number),
		unitPrice:     shared_domain.FormatCurrency(bounds[0].Amount),
		keys:          make([]string, len(bounds)),
		references:    make([]string, len(bounds)),
		date:          now.In(time.FixedZone("", session_domain.UtcDiff(ctx)*3600)).Format(adminDateLayout),
	}

	var total float64
	for i, b := range bounds {
		s.keys[i] = b.Key
		s.references[i] = movements[i].Reference
		total += b.Amount
	}
	s.total = shared_domain.FormatCurrency(total)
	return s
}

// buyerName devuelve el nombre y el apellido del perfil del comprador o, si no se puede leer, los de la
// sesión.
func (app *BondBoundApplication) buyerName(ctx context.Context, session *session_domain.UserSession) (string, string) {
	profiles, err := app.Profiles.GetPublic(ctx, session.UserID)
	if err != nil {
		log.Printf("[bondBound] no se pudo leer el perfil de %s para los avisos de la compra: %v", session.UserID, err)
	}
	if len(profiles) > 0 {
		return profiles[0].FirstName, profiles[0].LastName
	}
	return session.FirstName, session.LastName
}
