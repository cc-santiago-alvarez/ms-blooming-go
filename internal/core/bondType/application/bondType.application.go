package bondType_application

import (
	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
)

// BondTypeApplication implementa los casos de uso de bondType. BondObjects se conecta después de
// inicializar bondObject.
type BondTypeApplication struct {
	Repository  bondType_ports.BondTypeMongoccRepositoryPorts
	BondObjects bondType_ports.BondObjectCounterPort
}
