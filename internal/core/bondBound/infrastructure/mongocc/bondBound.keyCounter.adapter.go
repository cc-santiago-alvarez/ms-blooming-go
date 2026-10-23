package bondBound_mongocc_repository

import (
	"context"
	"fmt"

	"github.com/codecraftkit/mongocc"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Colección de los contadores de clave por tipo de bono.
const keyCounterCollection = "bondKeyCounter"

// KeyCounterMongoccRepository implementa KeyCounterPort sobre bondKeyCounter.
type KeyCounterMongoccRepository struct {
	DB *mongocc.MongoQueries
}

var _ bondBound_ports.KeyCounterPort = (*KeyCounterMongoccRepository)(nil)

// Reserve suma n al contador del tipo, creándolo desde 0 si no existe, y devuelve el primero de los n
// números reservados. Dentro de una transacción, dos reservas del mismo tipo chocan y el driver reintenta
// una, así que nunca se repite un número.
func (repo *KeyCounterMongoccRepository) Reserve(ctx context.Context, bondTypeID string, n int) (int, error) {
	if n < 1 {
		return 0, fmt.Errorf("reservar claves de %s: n debe ser al menos 1 y es %d", bondTypeID, n)
	}

	filter := bson.M{"_id": bondTypeID}
	update := bson.M{"$inc": bson.M{"seq": n}}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var c bondBound_domain.BondKeyCounter
	if err := repo.DB.FindOneAndUpdate(ctx, keyCounterCollection, filter, update, opts).Decode(&c); err != nil {
		return 0, err
	}
	return c.Seq - n + 1, nil
}
