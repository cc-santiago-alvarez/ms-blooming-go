package bondObject_domain

import (
	"strings"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// CreateParams es el body del alta de un producto.
type CreateParams struct {
	Name                      string                      `json:"name"`
	Description               string                      `json:"description"`
	AgencyID                  shared_domain.AgencyIDs     `json:"agencyId"`
	Key                       string                      `json:"key"`
	BondTypeID                string                      `json:"bondTypeId"`
	Amount                    float64                     `json:"amount"`
	IsPrivateBond             bool                        `json:"isPrivateBond"`
	AuthorizedUserIDs         []string                    `json:"authorizedUserIds"`
	MaturityTimeInDays        int                         `json:"maturityTimeInDays"`
	MaturityRewardsPercentage float64                     `json:"maturityRewardsPercentage"`
	Active                    bool                        `json:"active"`
	Available                 bool                        `json:"available"`
	SaleInitDate              *time.Time                  `json:"saleInitDate"`
	SaleEndDate               *time.Time                  `json:"saleEndDate"`
	EarlyCashTerm             shared_domain.EarlyCashTerm `json:"earlyCashTerm"`
}

// UpdateParams es el body de la edición: reemplaza los mismos campos del alta.
type UpdateParams = CreateParams

// defaultPenaltyPercentage es la penalidad que el legado aplica cuando llega 0 o no llega.
const defaultPenaltyPercentage = 100

// Normalize recorta los textos, limpia las listas de ids, redondea amount, pasa las fechas a UTC y
// aplica los defaults del legado: las penalidades en 0 quedan en 100.
func (p *CreateParams) Normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.Key = strings.TrimSpace(p.Key)
	p.BondTypeID = strings.TrimSpace(p.BondTypeID)
	p.AgencyID = p.AgencyID.Normalize()
	p.AuthorizedUserIDs = uniqueIDs(p.AuthorizedUserIDs)
	p.Amount = shared_domain.Round2(p.Amount)
	p.SaleInitDate = utc(p.SaleInitDate)
	p.SaleEndDate = utc(p.SaleEndDate)
	p.EarlyCashTerm = withDefaultPenalties(p.EarlyCashTerm)
}

// Validation exige los textos y las agencias, ids válidos, términos válidos y una ventana de venta
// ordenada.
func (p CreateParams) Validation() error {
	switch {
	case p.Name == "":
		return ErrBondObjectNameRequired
	case p.Key == "":
		return ErrBondObjectKeyRequired
	case !shared_domain.IsValidID(p.BondTypeID):
		return shared_domain.ErrInvalidID
	case len(p.AgencyID) == 0:
		return ErrBondObjectAgencyRequired
	case !allValidIDs(p.AgencyID) || !allValidIDs(p.AuthorizedUserIDs):
		return shared_domain.ErrInvalidID
	}

	terms := shared_domain.ProjectionTerms{
		MaturityTimeInDays:        p.MaturityTimeInDays,
		MaturityRewardsPercentage: p.MaturityRewardsPercentage,
		EarlyCashTerm:             p.EarlyCashTerm,
	}
	if err := validateTerms(p.Amount, terms); err != nil {
		return err
	}
	if p.SaleInitDate != nil && p.SaleEndDate != nil && p.SaleEndDate.Before(*p.SaleInitDate) {
		return ErrBondObjectInvalidSaleWindow
	}
	return nil
}

// ChartPreviewParams es el body de la vista previa: los términos de un producto sin guardar.
type ChartPreviewParams struct {
	Amount                    float64                     `json:"amount"`
	MaturityTimeInDays        int                         `json:"maturityTimeInDays"`
	MaturityRewardsPercentage float64                     `json:"maturityRewardsPercentage"`
	EarlyCashTerm             shared_domain.EarlyCashTerm `json:"earlyCashTerm"`
	Simulation                bool                        `json:"simulation"`
}

// Normalize redondea amount y aplica los mismos defaults de penalidad que el alta.
func (p *ChartPreviewParams) Normalize() {
	p.Amount = shared_domain.Round2(p.Amount)
	p.EarlyCashTerm = withDefaultPenalties(p.EarlyCashTerm)
}

// Validation exige términos válidos, con las mismas reglas del alta.
func (p ChartPreviewParams) Validation() error {
	return validateTerms(p.Amount, p.ProjectionTerms())
}

// ProjectionTerms devuelve los términos que usan la simulación y las guías.
func (p ChartPreviewParams) ProjectionTerms() shared_domain.ProjectionTerms {
	return shared_domain.ProjectionTerms{
		MaturityTimeInDays:        p.MaturityTimeInDays,
		MaturityRewardsPercentage: p.MaturityRewardsPercentage,
		EarlyCashTerm:             p.EarlyCashTerm,
	}
}

// BondObjectChart son las guías de la gráfica de un producto y, si se pidió, su simulación diaria.
type BondObjectChart struct {
	Guides []shared_domain.ChartGuide `json:"guides"`
	Data   []shared_domain.ChartPoint `json:"data,omitempty"`
}

// withDefaultPenalties aplica el default del legado: las penalidades en 0 quedan en 100.
func withDefaultPenalties(term shared_domain.EarlyCashTerm) shared_domain.EarlyCashTerm {
	if term.InitialPenaltyPercentage == 0 {
		term.InitialPenaltyPercentage = defaultPenaltyPercentage
	}
	if term.InflectionPenaltyPercentage == 0 {
		term.InflectionPenaltyPercentage = defaultPenaltyPercentage
	}
	return term
}

// validateTerms exige monto y plazo positivos, un punto de inflexión y un día de retiro dentro del plazo
// y penalidades entre 0 y 100.
func validateTerms(amount float64, terms shared_domain.ProjectionTerms) error {
	m, term := terms.MaturityTimeInDays, terms.EarlyCashTerm
	switch {
	case amount <= 0:
		return ErrBondObjectInvalidAmount
	case m <= 0 || terms.MaturityRewardsPercentage < 0:
		return ErrBondObjectInvalidMaturity
	case term.InflectionPoint < 0 || term.InflectionPoint >= m:
		return ErrBondObjectInvalidInflectionPoint
	case term.InitialDayForWithdraw < 0 || term.InitialDayForWithdraw > m:
		return ErrBondObjectInvalidWithdrawDay
	case !isPercentage(term.InitialPenaltyPercentage) || !isPercentage(term.InflectionPenaltyPercentage):
		return ErrBondObjectInvalidPercentage
	}
	return nil
}

// MaxUpdateKeysIDs es la cantidad máxima de productos de una actualización masiva de clave.
const MaxUpdateKeysIDs = 500

// UpdateKeysParams es el body de la actualización masiva de clave.
type UpdateKeysParams struct {
	Bonds []string `json:"bonds"`
	Key   string   `json:"key"`
}

// Normalize recorta key y limpia la lista de ids.
func (p *UpdateKeysParams) Normalize() {
	p.Key = strings.TrimSpace(p.Key)
	p.Bonds = uniqueIDs(p.Bonds)
}

// Validation exige key, entre 1 y MaxUpdateKeysIDs productos e ids válidos.
func (p UpdateKeysParams) Validation() error {
	switch {
	case p.Key == "":
		return ErrBondObjectKeyRequired
	case len(p.Bonds) == 0 || len(p.Bonds) > MaxUpdateKeysIDs:
		return ErrBondObjectIDsRequired
	case !allValidIDs(p.Bonds):
		return shared_domain.ErrInvalidID
	}
	return nil
}

// UpdateKeysResult es cuántos productos encontró y cuántos cambió la actualización masiva de clave.
type UpdateKeysResult struct {
	Matched  int64 `json:"matched"`
	Modified int64 `json:"modified"`
}

// BondObjectDetail es el producto con los perfiles públicos de sus usuarios autorizados.
type BondObjectDetail struct {
	BondObject
	AuthorizedUser []shared_domain.PublicProfile `json:"authorizedUser"`
}

// BondObjectForInvestor es un producto del listado por tipo, con la fecha de maduración si se compra hoy.
type BondObjectForInvestor struct {
	BondObject
	MaturityDateEstimated time.Time `json:"maturityDateEstimated"`
	MaturityDateFormatted string    `json:"maturityDateFormatted"`
}

// InvestorProfile son los datos del perfil que se muestran de un inversionista.
type InvestorProfile struct {
	PublicID  string `json:"id" bson:"id"`
	FirstName string `json:"first_name" bson:"first_name"`
	LastName  string `json:"last_name" bson:"last_name"`
	Email     string `json:"email" bson:"email"`
	Avatar    string `json:"avatar" bson:"avatar"`
	Address   string `json:"address" bson:"address"`
	AvatarURL string `json:"avatar_url" bson:"avatar_url"`
}

// BondObjectInvestor es un usuario con bonos vigentes de un producto y cuántos tiene.
type BondObjectInvestor struct {
	UserID      string          `json:"_id" bson:"_id"`
	NumOfBounds int             `json:"numOfBounds" bson:"numOfBounds"`
	Investor    InvestorProfile `json:"investor" bson:"investor"`
}

// VisibleFilter son los productos que ve un inversionista: no borrados, de su agencia y, si son privados,
// solo si está autorizado. BondTypeID e IDs vacíos no filtran; Limit 0 no limita.
type VisibleFilter struct {
	AgencyID   string
	UserID     string
	BondTypeID string
	IDs        []string
	OnlyActive bool
	Limit      int64
}

// uniqueIDs recorta cada id y quita los vacíos y los repetidos, conservando el orden; nunca es nil.
func uniqueIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// allValidIDs indica si todos los ids son UUID v4 o Meteor.
func allValidIDs(ids []string) bool {
	for _, id := range ids {
		if !shared_domain.IsValidID(id) {
			return false
		}
	}
	return true
}

// isPercentage indica si v está entre 0 y 100.
func isPercentage(v float64) bool {
	return v >= 0 && v <= 100
}

// utc devuelve la fecha en UTC, o nil.
func utc(t *time.Time) *time.Time {
	if t == nil {
		return nil
	}
	v := t.UTC()
	return &v
}
