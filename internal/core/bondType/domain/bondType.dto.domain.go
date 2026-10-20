package bondType_domain

import (
	"strings"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// CreateParams es el body del alta de un tipo de bono. active es false si no viene.
type CreateParams struct {
	Name        string                  `json:"name"`
	Description string                  `json:"description"`
	Active      bool                    `json:"active"`
	AgencyID    shared_domain.AgencyIDs `json:"agencyId"`
}

// UpdateParams es el body de la edición: reemplaza los mismos campos del alta.
type UpdateParams = CreateParams

// Normalize recorta name y description, y limpia agencyId.
func (p *CreateParams) Normalize() {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.AgencyID = p.AgencyID.Normalize()
}

// Validation exige name y al menos una agencia, y que cada agencia sea un id válido.
func (p CreateParams) Validation() error {
	switch {
	case p.Name == "":
		return ErrBondTypeNameRequired
	case len(p.AgencyID) == 0:
		return ErrBondTypeAgencyRequired
	}
	for _, id := range p.AgencyID {
		if !shared_domain.IsValidID(id) {
			return shared_domain.ErrInvalidID
		}
	}
	return nil
}

// ActiveBondType es un tipo del listado del inversionista, con la cantidad de bonos que puede ver.
type ActiveBondType struct {
	BondType
	NumOfBonds int `json:"numOfBonds"`
}
