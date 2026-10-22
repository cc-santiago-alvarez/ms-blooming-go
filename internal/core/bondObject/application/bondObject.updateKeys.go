package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
)

// UpdateKeys normaliza y valida p, cambia la clave de los productos no borrados de la lista y devuelve
// cuántos encontró y cuántos cambió. No registra historial.
func (app *BondObjectApplication) UpdateKeys(ctx context.Context, p bondObject_domain.UpdateKeysParams) (bondObject_domain.UpdateKeysResult, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return bondObject_domain.UpdateKeysResult{}, err
	}
	return app.Repository.UpdateKeys(ctx, p.Bonds, p.Key)
}
