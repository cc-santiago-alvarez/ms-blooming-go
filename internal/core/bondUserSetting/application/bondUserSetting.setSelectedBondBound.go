package bondUserSetting_application

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
)

// SetSelectedBondBound cambia el bono seleccionado del usuario de la sesión, o lo deselecciona con null, y
// devuelve el setting; si no tiene, lo crea. No valida que el bono sea suyo.
func (app *BondUserSettingApplication) SetSelectedBondBound(ctx context.Context, p bondUserSetting_domain.SetSelectedBondBoundParams) (*bondUserSetting_domain.BondUserSetting, error) {
	session, err := sessionUser(ctx)
	if err != nil {
		return nil, err
	}
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}

	return app.upsertField(ctx, session, bondUserSetting_ports.FieldSelectedBondBoundID, func(s *bondUserSetting_domain.BondUserSetting) {
		s.SelectedBondBoundID = p.SelectedBondBoundID
	})
}
