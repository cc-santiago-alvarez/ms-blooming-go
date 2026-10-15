package shared_mongocc_repository

import (
	"context"

	"github.com/codecraftkit/mongocc"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// TransactionRunner implementa TransactionRunnerPort sobre el cliente de mongocc. Requiere replica set.
type TransactionRunner struct {
	DB *mongocc.MongoQueries
}

// RunInTransaction ejecuta fn en una transacción; si fn devuelve error o hace panic, no queda escrito
// nada. Si ctx ya está en una transacción, fn se suma a esa. fn puede reintentarse, así que debe ser
// idempotente.
func (r *TransactionRunner) RunInTransaction(ctx context.Context, fn func(txCtx context.Context) error) error {
	if mongo.SessionFromContext(ctx) != nil {
		return fn(ctx)
	}

	session, err := r.DB.Database().Client().StartSession()
	if err != nil {
		return err
	}
	// Se cierra aunque el ctx del request ya esté cancelado; si la transacción sigue abierta, la aborta.
	defer session.EndSession(context.Background())

	txOpts := options.Transaction().
		SetReadConcern(readconcern.Snapshot()).
		SetWriteConcern(writeconcern.Majority())

	_, err = session.WithTransaction(ctx, func(txCtx context.Context) (any, error) {
		return nil, fn(txCtx)
	}, txOpts)
	return err
}
