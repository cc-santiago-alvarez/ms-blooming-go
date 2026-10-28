package bondBound_application

import (
	"context"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_application "github.com/futurapps/ms-blooming-go/internal/core/shared/application"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GetDetails devuelve el detalle del bono con lo acumulado a hoy en la zona de la sesión. Si no existe, le
// falta el producto o el tipo, o está fuera del alcance de agencia de la sesión, es ErrBondBoundNotFound; si
// es ajeno y la sesión no es admin, ErrForbidden. Un id inválido es ErrInvalidID.
func (app *BondBoundApplication) GetDetails(ctx context.Context, id string) (*bondBound_ports.BondBoundDetail, error) {
	if !shared_domain.IsValidID(id) {
		return nil, shared_domain.ErrInvalidID
	}

	row, err := app.Repository.FindDetails(ctx, id)
	if err != nil {
		return nil, err
	}
	if row == nil || !shared_application.InAgencyScope(ctx, row.Bound.AgencyID) {
		return nil, bondBound_domain.ErrBondBoundNotFound
	}
	if !shared_application.IsOwnerOrAdmin(ctx, row.Bound.UserID) {
		return nil, shared_domain.ErrForbidden
	}

	b := row.Bound
	daysPassed, accumulated := bondBound_domain.Accumulated(b, shared_domain.Now(), session_domain.UtcDiff(ctx))
	return &bondBound_ports.BondBoundDetail{
		ID:                        b.ID,
		Key:                       b.Key,
		Token:                     b.Token,
		AgencyID:                  b.AgencyID,
		Amount:                    b.Amount,
		MaturityTimeInDays:        b.MaturityTimeInDays,
		MaturityRewardsPercentage: b.MaturityRewardsPercentage,
		EarlyCashTerm:             b.EarlyCashTerm,
		InitialDate:               b.InitialDate,
		MaturityDate:              b.MaturityDate,
		SoldDate:                  b.SoldDate,
		TookDate:                  b.TookDate,
		Matured:                   b.Matured,
		CreatedAt:                 b.CreatedAt,
		UpdatedAt:                 b.UpdatedAt,
		IsRemove:                  b.IsRemove,
		BondObject:                publicBondObject(row.BondObject),
		BondType:                  row.BondType,
		Status:                    b.StatusID.Ref(),
		DaysPassed:                daysPassed,
		AcumulateAmount:           accumulated,
	}, nil
}

// publicBondObject copia el producto sin sus usuarios autorizados.
func publicBondObject(o bondObject_domain.BondObject) bondBound_ports.BondObjectPublic {
	return bondBound_ports.BondObjectPublic{
		ID:                        o.ID,
		Name:                      o.Name,
		Description:               o.Description,
		AgencyID:                  o.AgencyID,
		Key:                       o.Key,
		BondTypeID:                o.BondTypeID,
		Amount:                    o.Amount,
		IsPrivateBond:             o.IsPrivateBond,
		MaturityTimeInDays:        o.MaturityTimeInDays,
		MaturityRewardsPercentage: o.MaturityRewardsPercentage,
		Active:                    o.Active,
		Available:                 o.Available,
		SaleInitDate:              o.SaleInitDate,
		SaleEndDate:               o.SaleEndDate,
		EarlyCashTerm:             o.EarlyCashTerm,
		CreatedAt:                 o.CreatedAt,
		UpdatedAt:                 o.UpdatedAt,
		IsRemove:                  o.IsRemove,
	}
}
