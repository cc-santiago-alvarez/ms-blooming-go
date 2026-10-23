//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
)

// Reserve crea el contador desde 0 si no existe, sigue desde donde lo dejó la siembra y avanza n por
// reserva; un bondTypeId legacy es el _id del contador. CA-04.7, CA-04.14
func TestReserve(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name       string
		bondTypeID string
		seed       *int
		n          int
		wantStart  int
		wantSeq    int
	}{
		{name: "sin contador", bondTypeID: typeID, n: 1, wantStart: 1, wantSeq: 1},
		{name: "sembrado en 7", bondTypeID: typeID, seed: new(7), n: 1, wantStart: 8, wantSeq: 8},
		{name: "tres claves", bondTypeID: typeID, seed: new(8), n: 3, wantStart: 9, wantSeq: 11},
		{name: "bondTypeId legacy", bondTypeID: legacyTypeID, n: 2, wantStart: 1, wantSeq: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			if tt.seed != nil {
				seedCounter(t, tt.bondTypeID, *tt.seed)
			}

			start, err := keyCounter().Reserve(ctx, tt.bondTypeID, tt.n)

			if err != nil {
				t.Fatalf("Reserve: %v", err)
			}
			seq, ok := counterSeq(t, tt.bondTypeID)
			if start != tt.wantStart || !ok || seq != tt.wantSeq {
				t.Errorf("Reserve devolvió %d y el contador de %s quedó en %d (existe %v), se esperaban %d y %d",
					start, tt.bondTypeID, seq, ok, tt.wantStart, tt.wantSeq)
			}
		})
	}

	t.Run("n menor que 1", func(t *testing.T) {
		setup(t)

		_, err := keyCounter().Reserve(ctx, typeID, 0)

		if err == nil {
			t.Fatal("Reserve(0) no devolvió error")
		}
		if _, ok := counterSeq(t, typeID); ok {
			t.Error("Reserve(0) creó el contador")
		}
	})
}

// Una reserva dentro de una transacción que falla no hace avanzar el contador. CA-04.9
func TestReserveRollback(t *testing.T) {
	setup(t)
	seedCounter(t, typeID, 58)
	errAbort := errors.New("falla después de reservar")

	err := txRunner().RunInTransaction(context.Background(), func(txCtx context.Context) error {
		start, err := keyCounter().Reserve(txCtx, typeID, 3)
		if err != nil {
			return err
		}
		if start != 59 {
			t.Errorf("Reserve dentro de la transacción devolvió %d, se esperaba 59", start)
		}
		return errAbort
	})

	if !errors.Is(err, errAbort) {
		t.Fatalf("RunInTransaction = %v, se esperaba el error del callback", err)
	}
	if seq, _ := counterSeq(t, typeID); seq != 58 {
		t.Errorf("el contador quedó en %d, se esperaba 58", seq)
	}
}

// Reservas concurrentes del mismo tipo, cada una en su transacción, obtienen rangos que no se pisan, exista
// o no el contador. CA-04.7
func TestReserveConcurrent(t *testing.T) {
	const (
		workers = 10
		n       = 2
	)

	tests := []struct {
		name string
		seed *int
		base int
	}{
		{name: "contador inexistente"},
		{name: "contador sembrado en 7", seed: new(7), base: 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			if tt.seed != nil {
				seedCounter(t, typeID, *tt.seed)
			}

			starts := make([]int, workers)
			errs := make([]error, workers)
			ready := make(chan struct{})
			var wg sync.WaitGroup
			for i := range workers {
				wg.Go(func() {
					<-ready
					errs[i] = txRunner().RunInTransaction(context.Background(), func(txCtx context.Context) error {
						// el callback puede reintentarse: queda el inicio del intento que confirmó
						start, err := keyCounter().Reserve(txCtx, typeID, n)
						starts[i] = start
						return err
					})
				})
			}
			close(ready)
			wg.Wait()

			for i, err := range errs {
				if err != nil {
					t.Fatalf("la reserva %d falló: %v", i, err)
				}
			}
			slices.Sort(starts)
			for i, start := range starts {
				if want := tt.base + 1 + i*n; start != want {
					t.Fatalf("inicios %v, se esperaban rangos consecutivos de %d desde %d", starts, n, tt.base+1)
				}
			}
			if seq, _ := counterSeq(t, typeID); seq != tt.base+workers*n {
				t.Errorf("el contador quedó en %d, se esperaba %d", seq, tt.base+workers*n)
			}
		})
	}
}
