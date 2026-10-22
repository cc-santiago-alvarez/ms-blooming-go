package bondObject_application

import (
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// BondObjectApplication implementa los casos de uso de bondObject.
type BondObjectApplication struct {
	Repository bondObject_ports.BondObjectMongoccRepositoryPorts
	Profiles   shared_ports.ProfileReader
}
