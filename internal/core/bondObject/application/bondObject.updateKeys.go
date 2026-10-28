package bondObject_application

import (
	"context"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
)

// UpdateKeys normaliza y valida p, cambia la clave de los productos no borrados de la lista y devuelve
// cuántos encontró y cuántos cambió. Fuera de la agencia principal solo cambia los que son únicamente de la
// agencia de la sesión; los demás no cuentan ni son error, y sin sesión no cambia ninguno. No registra
// historial.
func (app *BondObjectApplication) UpdateKeys(ctx context.Context, p bondObject_domain.UpdateKeysParams) (bondObject_domain.UpdateKeysResult, error) {
	p.Normalize()
	if err := p.Validation(); err != nil {
		return bondObject_domain.UpdateKeysResult{}, err
	}

	scope := shared_application.AgencyScope(ctx)
	if scope != nil && *scope == "" {
		return bondObject_domain.UpdateKeysResult{}, nil
	}
	return app.Repository.UpdateKeys(ctx, p.Bonds, p.Key, scope)
}
