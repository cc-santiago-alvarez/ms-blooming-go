package bondUserSetting_domain

import (
	"strings"
	"unicode/utf8"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// MaxLanguageLength es la cantidad máxima de caracteres de un idioma.
const MaxLanguageLength = 5

// SetSelectedAccountParams es el body del cambio de cuenta destino. El usuario sale de la sesión.
type SetSelectedAccountParams struct {
	SelectedAccountID string `json:"selectedAccountId"`
}

// Normalize recorta selectedAccountId.
func (p *SetSelectedAccountParams) Normalize() {
	p.SelectedAccountID = strings.TrimSpace(p.SelectedAccountID)
}

// Validation exige que selectedAccountId sea un id válido.
func (p SetSelectedAccountParams) Validation() error {
	if !shared_domain.IsValidID(p.SelectedAccountID) {
		return shared_domain.ErrInvalidID
	}
	return nil
}

// SetSelectedBondBoundParams es el body del cambio de bono seleccionado; null lo deselecciona.
type SetSelectedBondBoundParams struct {
	SelectedBondBoundID *string `json:"selectedBondBoundId"`
}

// Normalize recorta selectedBondBoundId si viene.
func (p *SetSelectedBondBoundParams) Normalize() {
	if p.SelectedBondBoundID != nil {
		id := strings.TrimSpace(*p.SelectedBondBoundID)
		p.SelectedBondBoundID = &id
	}
}

// Validation acepta null o un id válido.
func (p SetSelectedBondBoundParams) Validation() error {
	if p.SelectedBondBoundID != nil && !shared_domain.IsValidID(*p.SelectedBondBoundID) {
		return shared_domain.ErrInvalidID
	}
	return nil
}

// SetLanguageParams es el body del cambio de idioma.
type SetLanguageParams struct {
	Language string `json:"language"`
}

// Normalize recorta language.
func (p *SetLanguageParams) Normalize() {
	p.Language = strings.TrimSpace(p.Language)
}

// Validation exige un idioma no vacío de hasta MaxLanguageLength caracteres.
func (p SetLanguageParams) Validation() error {
	if p.Language == "" || utf8.RuneCountInString(p.Language) > MaxLanguageLength {
		return ErrInvalidLanguage
	}
	return nil
}

// SelectedAccount es la cuenta seleccionada como la muestra el setting, con el número enmascarado.
type SelectedAccount struct {
	ID        string  `json:"_id"`
	Number    string  `json:"number"`
	Available float64 `json:"available"`
	StatusID  string  `json:"statusId"`
}

// NewSelectedAccount arma la cuenta que muestra el setting, o nil si no hay cuenta.
func NewSelectedAccount(a *shared_domain.BalanceAccount) *SelectedAccount {
	if a == nil {
		return nil
	}
	return &SelectedAccount{
		ID:        a.ID,
		Number:    shared_domain.MaskAccountNumber(a.Number),
		Available: shared_domain.Round2(a.Available),
		StatusID:  a.StatusID,
	}
}

// Estados de bondBound que suma el resumen del portafolio.
const (
	boundStatusInitiated = "1"
	boundStatusFinalized = "2"
)

// BoundSummary es el resumen del portafolio: lo invertido en bonos vigentes y lo madurado. profit
// siempre es 0.
type BoundSummary struct {
	Investment float64 `json:"investment"`
	Matured    float64 `json:"matured"`
	Profit     float64 `json:"profit"`
}

// NewBoundSummary arma el resumen con la suma de amount por statusId: investment son los bonos iniciados y
// matured los finalizados; los demás estados no cuentan.
func NewBoundSummary(totalByStatus map[string]float64) BoundSummary {
	return BoundSummary{
		Investment: shared_domain.Round2(totalByStatus[boundStatusInitiated]),
		Matured:    shared_domain.Round2(totalByStatus[boundStatusFinalized]),
		Profit:     0,
	}
}

// BondUserSettingDetail es el setting del usuario con su cuenta seleccionada, su perfil público, su
// diferencia horaria y el resumen de su portafolio.
type BondUserSettingDetail struct {
	BondUserSetting
	Account      *SelectedAccount             `json:"account"`
	User         *shared_domain.PublicProfile `json:"user"`
	UtcDiff      int                          `json:"utcDiff"`
	BoundSummary BoundSummary                 `json:"boundSummary"`
}
