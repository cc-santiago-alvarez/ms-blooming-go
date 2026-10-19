package bondHistory_domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

// HistoryTypeRef es el tipo de un registro como lo muestra el historial.
type HistoryTypeRef struct {
	ID   HistoryType `json:"_id"`
	Name string      `json:"name"`
}

// HistoryUser es el autor de un registro: el nombre de su perfil o, si no tiene, el sistema.
type HistoryUser struct {
	FirstName string
	LastName  string
	System    bool
}

// MarshalJSON escribe {first_name, last_name} o, para el sistema, {name: "System"}.
func (u HistoryUser) MarshalJSON() ([]byte, error) {
	if u.System {
		return json.Marshal(struct {
			Name string `json:"name"`
		}{Name: "System"})
	}
	return json.Marshal(struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
	}{FirstName: u.FirstName, LastName: u.LastName})
}

// HistoryEntry es un registro del historial listo para mostrar: con el texto formateado y sin userId
// ni params.
type HistoryEntry struct {
	ID               string                 `json:"_id"`
	RelatedID        string                 `json:"relatedId"`
	TypeID           HistoryType            `json:"typeId"`
	Type             HistoryTypeRef         `json:"type"`
	AgencyID         string                 `json:"agencyId"`
	Text             string                 `json:"text"`
	User             HistoryUser            `json:"user"`
	Icon             string                 `json:"icon"`
	Color            string                 `json:"color"`
	BackgroundColor  string                 `json:"backgroundColor"`
	Changes          map[string]ChangeEntry `json:"changes,omitempty"`
	EditedProperties []string               `json:"editedProperties,omitempty"`
	CreatedAt        time.Time              `json:"createdAt"`
	CreatedAtFromNow string                 `json:"createdAtFromNow"`
}
