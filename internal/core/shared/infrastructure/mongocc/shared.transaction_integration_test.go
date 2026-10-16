//go:build integration

package shared_mongocc_repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/codecraftkit/mongocc"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var errCallback = errors.New("falla del callback")

// insertProbe guarda un documento de prueba con ctx y falla el test si no se pudo.
func insertProbe(t *testing.T, ctx context.Context, db *mongocc.MongoQueries, id string) {
	t.Helper()
	if _, err := db.InsertOne(ctx, txProbeCollection, bson.M{"_id": id}); err != nil {
		t.Fatalf("no se pudo insertar %s dentro de la transacción: %v", id, err)
	}
}

// Si el callback falla o hace panic no queda nada escrito, tampoco lo que escribió el BalancePort
// con el ctx de la transacción del llamador. CA-00.13
func TestRunInTransaction(t *testing.T) {
	ctx := context.Background()

	t.Run("callback con error", func(t *testing.T) {
		db := setup(t)
		runner := &shared_mongocc_repository.TransactionRunner{DB: db}

		err := runner.RunInTransaction(ctx, func(txCtx context.Context) error {
			insertProbe(t, txCtx, db, meteorAccountID)
			return errCallback
		})

		if !errors.Is(err, errCallback) {
			t.Fatalf("RunInTransaction() = %v, se esperaba el error del callback", err)
		}
		if n := countDocs(t, db, txProbeCollection); n != 0 {
			t.Fatalf("quedaron %d documentos tras el rollback", n)
		}
	})

	t.Run("callback sin error", func(t *testing.T) {
		db := setup(t)
		runner := &shared_mongocc_repository.TransactionRunner{DB: db}

		err := runner.RunInTransaction(ctx, func(txCtx context.Context) error {
			insertProbe(t, txCtx, db, uuidAccountID)
			return nil
		})

		if err != nil {
			t.Fatalf("RunInTransaction() = %v", err)
		}
		if n := countDocs(t, db, txProbeCollection); n != 1 {
			t.Fatalf("hay %d documentos, se esperaba 1", n)
		}
	})

	t.Run("panic en el callback", func(t *testing.T) {
		db := setup(t)
		runner := &shared_mongocc_repository.TransactionRunner{DB: db}

		recovered := func() (rec any) {
			defer func() { rec = recover() }()
			_ = runner.RunInTransaction(ctx, func(txCtx context.Context) error {
				insertProbe(t, txCtx, db, meteorAccountID)
				panic("falla inesperada")
			})
			return nil
		}()

		if recovered != "falla inesperada" {
			t.Fatalf("recover() = %v, se esperaba que el panic llegara al llamador", recovered)
		}
		if n := countDocs(t, db, txProbeCollection); n != 0 {
			t.Fatalf("quedaron %d documentos tras el panic", n)
		}
	})

	t.Run("RunInTransaction anidado", func(t *testing.T) {
		db := setup(t)
		runner := &shared_mongocc_repository.TransactionRunner{DB: db}

		err := runner.RunInTransaction(ctx, func(txCtx context.Context) error {
			inner := runner.RunInTransaction(txCtx, func(innerCtx context.Context) error {
				insertProbe(t, innerCtx, db, meteorAccountID)
				return nil
			})
			if inner != nil {
				return inner
			}
			return errCallback
		})

		if !errors.Is(err, errCallback) {
			t.Fatalf("RunInTransaction() = %v, se esperaba el error del callback", err)
		}
		if n := countDocs(t, db, txProbeCollection); n != 0 {
			t.Fatalf("el anidado confirmó %d documentos aunque el de afuera falló", n)
		}
	})

	for _, a := range accountFixtures {
		t.Run(a.name+"/Withdraw se suma a la transacción del llamador", func(t *testing.T) {
			db := setup(t)
			insertAccount(t, db, a, activeAccount)
			before := rawDoc(t, db, "balanceAccount", a.id)
			runner := &shared_mongocc_repository.TransactionRunner{DB: db}
			repo := newBalanceRepo(db)

			err := runner.RunInTransaction(ctx, func(txCtx context.Context) error {
				mov, err := repo.Withdraw(txCtx, withdrawParams(a, 300))
				if err != nil {
					return err
				}
				if mov == nil || mov.Available != 700 {
					t.Errorf("el retiro dentro de la transacción devolvió %+v", mov)
				}
				return errCallback
			})

			if !errors.Is(err, errCallback) {
				t.Fatalf("RunInTransaction() = %v, se esperaba el error del callback", err)
			}
			assertDocUnchanged(t, db, "balanceAccount", a.id, before)
			assertNoMovements(t, db)
		})
	}
}
