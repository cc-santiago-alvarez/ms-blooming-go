package bondUserSetting_application

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GetByID devuelve el setting vivo del alcance de la sesión, o ErrBondUserSettingNotFound. Un id inválido
// es ErrInvalidID.
func (app *BondUserSettingApplication) GetByID(ctx context.Context, id string) (*bondUserSetting_domain.BondUserSetting, error) {
	return app.findInScope(ctx, id)
}

// findInScope devuelve el setting vivo si es de la agencia de la sesión, o de cualquiera si es la
// principal; si no, ErrBondUserSettingNotFound. Un id inválido es ErrInvalidID.
func (app *BondUserSettingApplication) findInScope(ctx context.Context, id string) (*bondUserSetting_domain.BondUserSetting, error) {
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}

	s, err := app.Repository.FindOneByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s == nil {
		return nil, bondUserSetting_domain.ErrBondUserSettingNotFound
	}
	if scope := shared_application.AgencyScope(ctx); scope != nil && s.AgencyID != *scope {
		return nil, bondUserSetting_domain.ErrBondUserSettingNotFound
	}
	return s, nil
}
