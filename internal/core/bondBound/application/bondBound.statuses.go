package bondBound_application

import (
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// Statuses devuelve los seis estados de un bono en orden, con su nombre.
func (app *BondBoundApplication) Statuses() []bondBound_domain.StatusRef {
	return bondBound_domain.StatusRefs()
}
