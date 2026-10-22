package bondUserSetting_application

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GetMine devuelve el setting del usuario de la sesión con su cuenta, su perfil y el resumen de su
// portafolio. Si no tiene, lo crea de forma atómica con idioma "es" y su cuenta más antigua de la agencia.
func (app *BondUserSettingApplication) GetMine(ctx context.Context) (*bondUserSetting_domain.BondUserSettingDetail, error) {
	session, err := sessionUser(ctx)
	if err != nil {
		return nil, err
	}

	s, err := app.Repository.FindByUser(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if s == nil {
		defaults, err := app.newSetting(ctx, session)
		if err != nil {
			return nil, err
		}
		if s, err = app.Repository.UpsertDefault(ctx, defaults); err != nil {
			return nil, err
		}
	}
	return app.detail(ctx, session, s)
}

// detail completa el setting con la cuenta seleccionada si es del usuario, su perfil público, la
// diferencia horaria de la sesión y el resumen de sus bonos en la agencia.
func (app *BondUserSettingApplication) detail(ctx context.Context, session *session_domain.UserSession, s *bondUserSetting_domain.BondUserSetting) (*bondUserSetting_domain.BondUserSettingDetail, error) {
	var account *shared_domain.BalanceAccount
	if id := s.SelectedAccountID; id != nil && *id != "" {
		var err error
		if account, err = app.Balance.GetAccount(ctx, *id, session.UserID); err != nil {
			return nil, err
		}
	}

	profiles, err := app.Profiles.GetPublic(ctx, s.UserID)
	if err != nil {
		return nil, err
	}
	var user *shared_domain.PublicProfile
	if len(profiles) > 0 {
		user = &profiles[0]
	}

	summary, err := app.BondBounds.BoundSummary(ctx, session.UserID, session.AgencyID)
	if err != nil {
		return nil, err
	}

	return &bondUserSetting_domain.BondUserSettingDetail{
		BondUserSetting: *s,
		Account:         bondUserSetting_domain.NewSelectedAccount(account),
		User:            user,
		UtcDiff:         session_domain.UtcDiff(ctx),
		BoundSummary:    summary,
	}, nil
}
