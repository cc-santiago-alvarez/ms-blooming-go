package shared_mongocc_repository

import (
	"github.com/codecraftkit/mongocc"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
)

// Colección de actividad del módulo legado. No se crean índices.
const activityCollection = "activity"

// ActivityMongoccRepository implementa ActivityPort sobre activity.
type ActivityMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ shared_ports.ActivityPort = (*ActivityMongoccRepository)(nil)
