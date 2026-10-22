package bondObject_application

import (
	"context"
	"log"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
)

// addHistory registra un evento del producto en el historial; un fallo solo se loguea.
func (app *BondObjectApplication) addHistory(ctx context.Context, p bondHistory_domain.AddParams) {
	p.TypeID = bondHistory_domain.HistoryBond
	if err := app.History.Add(ctx, p); err != nil {
		log.Printf("[bondObject] no se pudo registrar %s del producto %s: %v", p.Text, p.RelatedID, err)
	}
}
