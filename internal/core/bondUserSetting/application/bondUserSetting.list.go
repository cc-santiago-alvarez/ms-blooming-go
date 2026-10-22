package bondUserSetting_application

import (
	"context"

	bondUserSetting_domain "github.com/futurapps/ms-blooming-go/internal/core/bondUserSetting/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// List devuelve la página de settings vivos del alcance de la sesión, por createdAt descendente.
func (app *BondUserSettingApplication) List(ctx context.Context, p shared_domain.Pagination) (shared_domain.Paginated[*bondUserSetting_domain.BondUserSetting], error) {
	p.Normalize()
	return app.Repository.FindPage(ctx, shared_application.AgencyScope(ctx), p)
}
