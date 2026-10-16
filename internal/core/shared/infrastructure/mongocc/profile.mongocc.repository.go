package shared_mongocc_repository

import (
	"github.com/codecraftkit/mongocc"
	shared_ports "github.com/futurapps/ms-blooming-go/internal/core/shared/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Colección de perfiles de la plataforma. Solo lectura; no se crean índices.
const profileCollection = "profile"

// publicProfileProjection es lo único que se lee de un perfil: nunca password ni permisos.
var publicProfileProjection = bson.M{
	"_id": 1, "id": 1, "first_name": 1, "last_name": 1, "email": 1, "avatar_url": 1, "agencyId": 1,
}

// ProfileMongoccRepository implementa ProfileReader sobre profile.
type ProfileMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ shared_ports.ProfileReader = (*ProfileMongoccRepository)(nil)
