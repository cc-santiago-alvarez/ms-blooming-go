package bondUserSetting_application

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	bondUserSetting_ports "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/ports"
)

// SetLanguage cambia el idioma del usuario de la sesión y devuelve el setting; si no tiene, lo crea. Un
// idioma vacío o de más de 5 caracteres es ErrInvalidLanguage.
func (app *BondUserSettingApplication) SetLanguage(ctx context.Context, p bondUserSetting_domain.SetLanguageParams) (*bondUserSetting_domain.BondUserSetting, error) {
	session, err := sessionUser(ctx)
	if err != nil {
		return nil, err
	}
	p.Normalize()
	if err := p.Validation(); err != nil {
		return nil, err
	}

	return app.upsertField(ctx, session, bondUserSetting_ports.FieldLanguage, func(s *bondUserSetting_domain.BondUserSetting) {
		s.Language = p.Language
	})
}
