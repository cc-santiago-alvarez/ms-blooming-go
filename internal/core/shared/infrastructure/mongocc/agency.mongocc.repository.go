package shared_mongocc_repository

import (
	"github.com/codecraftkit/mongocc"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// Colecciones de agencias y países de la plataforma. Solo lectura; no se crean índices.
const (
	agencyCollection  = "agency"
	countryCollection = "country"
)

// defaultSymbolCurrency es el símbolo cuando el país de la agencia no define uno.
const defaultSymbolCurrency = "$"

// AgencyMongoccRepository implementa AgencyReader sobre agency y country.
type AgencyMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ shared_ports.AgencyReader = (*AgencyMongoccRepository)(nil)
