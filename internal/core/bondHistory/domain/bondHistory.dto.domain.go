package bondHistory_domain

import (
	"fmt"
	"strings"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// AddParams son los datos de un evento del historial. El usuario y la agencia salen de la sesión.
type AddParams struct {
	RelatedID        string
	TypeID           HistoryType
	Text             string
	Params           []string
	Changes          map[string]ChangeEntry
	EditedProperties []string
}

// Normalize recorta relatedId y text, y deja params en [] si no viene.
func (p *AddParams) Normalize() {
	p.RelatedID = strings.TrimSpace(p.RelatedID)
	p.Text = strings.TrimSpace(p.Text)
	if p.Params == nil {
		p.Params = []string{}
	}
}

// Validation exige un tipo conocido, relatedId y text, y un cambio de estado con exactamente el estado
// anterior y el nuevo.
func (p AddParams) Validation() error {
	switch {
	case !p.TypeID.IsValid():
		return fmt.Errorf("%w: typeId %q", ErrBondHistoryInvalidParams, p.TypeID)
	case p.RelatedID == "":
		return fmt.Errorf("%w: relatedId vacío", ErrBondHistoryInvalidParams)
	case p.Text == "":
		return fmt.Errorf("%w: text vacío", ErrBondHistoryInvalidParams)
	case p.Text == shared_domain.TextBondBoundStatusChanged && len(p.Params) != 2:
		return fmt.Errorf("%w: %s con %d params", ErrBondHistoryInvalidParams, p.Text, len(p.Params))
	}
	return nil
}
