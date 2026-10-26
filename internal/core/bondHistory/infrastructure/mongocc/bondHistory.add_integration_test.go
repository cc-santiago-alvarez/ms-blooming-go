//go:build integration

package bondHistory_mongocc_repository_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// statusChanged es el cambio de estado de boundID de iniciado a vendido.
var statusChanged = bondHistory_domain.AddParams{
	RelatedID: boundID,
	TypeID:    bondHistory_domain.HistoryBondBound,
	Text:      "BOND_BOUND_STATUS_CHANGED",
	Params:    []string{"1", "5"},
}

// rawStrings devuelve el array de strings key del documento; ok es false si no es un array de strings.
func rawStrings(raw bson.Raw, key string) (values []string, ok bool) {
	array, ok := raw.Lookup(key).ArrayOK()
	if !ok {
		return nil, false
	}
	elems, err := array.Values()
	if err != nil {
		return nil, false
	}
	values = []string{}
	for _, v := range elems {
		s, ok := v.StringValueOK()
		if !ok {
			return nil, false
		}
		values = append(values, s)
	}
	return values, true
}

// Add guarda exactamente los valores recibidos como strings, con el usuario y la agencia de la sesión, _id
// UUID, timestamps e isRemove false; sin params guarda un array vacío, y en modo sistema el usuario queda
// vacío con la agencia marcada. CA-08.1
func TestAddPersists(t *testing.T) {
	sold := bondHistory_domain.AddParams{RelatedID: boundID, TypeID: bondHistory_domain.HistoryBondBound, Text: "BOND_BOUND_SOLD"}

	tests := []struct {
		name       string
		ctx        context.Context
		params     bondHistory_domain.AddParams
		wantUser   string
		wantAgency string
		wantParams []string
	}{
		{name: "cambio de estado", ctx: userSession(userID, agencyA), params: statusChanged,
			wantUser: userID, wantAgency: agencyA, wantParams: []string{"1", "5"}},
		{name: "sin params", ctx: userSession(userID, agencyA), params: sold,
			wantUser: userID, wantAgency: agencyA, wantParams: []string{}},
		{name: "modo sistema", ctx: session_domain.WithSystemAgency(context.Background(), agencyB), params: statusChanged,
			wantUser: "", wantAgency: agencyB, wantParams: []string{"1", "5"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			before := time.Now()

			if err := newApp().Add(tt.ctx, tt.params); err != nil {
				t.Fatalf("Add: %v", err)
			}

			after := time.Now()
			docs := rawDocs(t, historyCollection)
			if len(docs) != 1 {
				t.Fatalf("hay %d registros, se esperaba 1", len(docs))
			}
			raw := docs[0]
			id, _ := raw.Lookup("_id").StringValueOK()
			if shared_domain.IDFormat(id) != shared_domain.IDKindUUID {
				t.Errorf("_id %q, se esperaba un UUID v4", id)
			}
			fields := map[string]string{
				"relatedId": boundID,
				"typeId":    "2",
				"text":      tt.params.Text,
				"userId":    tt.wantUser,
				"agencyId":  tt.wantAgency,
			}
			for key, want := range fields {
				if got, ok := raw.Lookup(key).StringValueOK(); !ok || got != want {
					t.Errorf("%s = %v, se esperaba el string %q", key, raw.Lookup(key), want)
				}
			}
			if params, ok := rawStrings(raw, "params"); !ok || !slices.Equal(params, tt.wantParams) {
				t.Errorf("params = %v, se esperaba el array %q", raw.Lookup("params"), tt.wantParams)
			}
			for _, key := range []string{"createdAt", "updatedAt"} {
				at, ok := raw.Lookup(key).TimeOK()
				if !ok || at.Before(before.Truncate(time.Millisecond)) || at.After(after) {
					t.Errorf("%s = %v, se esperaba una fecha entre %v y %v", key, raw.Lookup(key), before, after)
				}
			}
			if removed, ok := raw.Lookup("isRemove").BooleanOK(); !ok || removed {
				t.Errorf("isRemove = %v, se esperaba false", raw.Lookup("isRemove"))
			}
			for _, key := range []string{"changes", "editedProperties"} {
				if _, err := raw.LookupErr(key); err == nil {
					t.Errorf("el registro tiene %s sin que se lo pasaran: %v", key, raw.Lookup(key))
				}
			}
		})
	}
}

// Un Add dentro de una transacción que después se revierte no deja el registro; si la transacción se
// confirma, sí. CA-08.2
func TestAddRollback(t *testing.T) {
	errRollback := errors.New("falla simulada después del registro")

	tests := []struct {
		name      string
		after     error
		wantCount int64
	}{
		{name: "revertida", after: errRollback, wantCount: 0},
		{name: "confirmada", wantCount: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)

			err := txRunner().RunInTransaction(userSession(userID, agencyA), func(txCtx context.Context) error {
				if err := newApp().Add(txCtx, statusChanged); err != nil {
					return err
				}
				return tt.after
			})

			if !errors.Is(err, tt.after) {
				t.Fatalf("RunInTransaction = %v, se esperaba %v", err, tt.after)
			}
			if n := countDocs(t, historyCollection); n != tt.wantCount {
				t.Errorf("hay %d registros, se esperaban %d", n, tt.wantCount)
			}
		})
	}
}
