package bondObject_application

import (
	bondHistory_ports "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/ports"
	bondObject_ports "github.com/futurapps/ms-blooming-go/internal/core/bondObject/ports"
	bondType_ports "github.com/futurapps/ms-blooming-go/internal/core/bondType/ports"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// BondObjectApplication implementa los casos de uso de bondObject.
type BondObjectApplication struct {
	Repository     bondObject_ports.BondObjectMongoccRepositoryPorts
	BondBoundStats bondObject_ports.BondBoundStatsPort
	BondTypes      bondType_ports.BondTypeApplicationPorts
	History        bondHistory_ports.BondHistoryPort
	Activity       shared_ports.ActivityPort
	Profiles       shared_ports.ProfileReader
}

var _ bondObject_ports.BondObjectApplicationPorts = (*BondObjectApplication)(nil)
