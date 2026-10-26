package bondBound_application

import (
	"context"
	"html"
	"log"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// Notificaciones de ms-notification que manda la venta.
const (
	notifyBondSold = "notify-blooming-bondSold"
	notifyBondTook = "notify-blooming-bondTook"
)

// saleDateLayout es el formato de las fechas del aviso de venta: DD/MM/YYYY h:m:s a.
const saleDateLayout = "02/01/2006 3:4:5 pm"

// notifySale avisa al dueño la venta o el canje de un bono: guarda el aviso del bono y, en segundo plano,
// manda el email. Los términos salen del bono, las fechas van en la zona de la sesión y el disponible es el
// que quedó después del depósito. Ningún fallo llega al llamador.
func (app *BondBoundApplication) notifySale(ctx context.Context, session *session_domain.UserSession, b *bondBound_domain.BondBound,
	t bondBound_domain.SaleTransition, quote bondBound_domain.SaleProjection, account *shared_domain.BalanceAccount, mov *shared_domain.BalanceMovement,
) {
	ctx = context.WithoutCancel(ctx)
	zone := time.FixedZone("", session_domain.UtcDiff(ctx)*3600)

	bondName := app.bondName(ctx, b.BondObjectID)
	amount := shared_domain.FormatCurrency(b.Amount)
	initialDate := b.InitialDate.In(zone).Format(saleDateLayout)
	maturityDate := b.MaturityDate.In(zone).Format(saleDateLayout)
	accountMasked := shared_domain.MaskAccountNumber(account.Number)
	profit := shared_domain.FormatCurrency(quote.Profit)
	credited := shared_domain.FormatCurrency(quote.Amount)
	available := shared_domain.FormatCurrency(mov.Available)

	subject, message, notification := shared_domain.TextBondBoundSubjectSold, shared_domain.TextBondBoundMessageSold, notifyBondSold
	if t.To == bondBound_domain.StatusTook {
		subject, message, notification = shared_domain.TextBondBoundSubjectTook, shared_domain.TextBondBoundMessageTook, notifyBondTook
	}

	app.Notices.Add(ctx, shared_domain.Notice{
		Type:     shared_domain.NoticeTypeBond,
		UserID:   session.UserID,
		AgencyID: session.AgencyID,
		Title:    shared_domain.Format(shared_domain.Text(subject), b.Key),
		Message: shared_domain.Format(shared_domain.Text(message),
			html.EscapeString(bondName), html.EscapeString(b.Key), amount, b.MaturityRewardsPercentage,
			b.MaturityTimeInDays, initialDate, maturityDate, html.EscapeString(accountMasked), profit, credited,
			available),
	})

	go app.Notifications.Send(ctx, groupUserIDs, notification, map[string]any{
		"userIds":             []string{session.UserID},
		"bondName":            bondName,
		"key":                 b.Key,
		"amount":              amount,
		"rewardsPercentage":   b.MaturityRewardsPercentage,
		"maturityDays":        b.MaturityTimeInDays,
		"initialDate":         initialDate,
		"maturityDate":        maturityDate,
		"accountNumberMasked": accountMasked,
		"profit":              profit,
		"credited":            credited,
		"available":           available,
	})
}

// bondName devuelve el nombre del producto, aunque esté borrado, o "" si no se puede leer.
func (app *BondBoundApplication) bondName(ctx context.Context, bondObjectID string) string {
	o, err := app.BondObjects.GetByIDIncludingRemoved(ctx, bondObjectID)
	if err != nil {
		log.Printf("[bondBound] no se pudo leer el producto %s para los avisos de la venta: %v", bondObjectID, err)
		return ""
	}
	return o.Name
}
