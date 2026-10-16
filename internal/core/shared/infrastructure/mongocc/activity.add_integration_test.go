//go:build integration

package shared_mongocc_repository_test

import (
	"context"
	"reflect"
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	shared_mongocc_repository "github.com/futurapps/ms-blooming-go/internal/core/shared/infrastructure/mongocc"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// ActivityPort.Add guarda la actividad del módulo Blooming, con params {} si no vienen, aunque el
// ctx del llamador esté cancelado. CA-00.35
func TestActivityAdd(t *testing.T) {
	purchase := map[string]any{"key": "AB-3", "amount": 1500.5}

	tests := []struct {
		name       string
		params     map[string]any
		wantParams map[string]any
		cancelled  bool
	}{
		{name: "con params", params: purchase, wantParams: purchase},
		{name: "sin params", params: nil, wantParams: map[string]any{}},
		{name: "ctx cancelado", params: purchase, wantParams: purchase, cancelled: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setup(t)
			ctx, cancel := context.WithCancel(context.Background())
			if tt.cancelled {
				cancel()
			} else {
				defer cancel()
			}
			repo := &shared_mongocc_repository.ActivityMongoccRepository{DB: db}

			repo.Add(ctx, shared_domain.Activity{
				Description: "BOND_BOUND_PURCHASED",
				RelatedID:   meteorBondID,
				AgencyID:    meteorAgencyID,
				UserID:      uuidUserID,
				Params:      tt.params,
			})

			records := findAll[shared_domain.ActivityRecord](t, db, "activity")
			if len(records) != 1 {
				t.Fatalf("hay %d actividades, se esperaba 1", len(records))
			}
			r := records[0]
			assertUUIDv4(t, r.ID)
			assertFieldType(t, rawDoc(t, db, "activity", r.ID), "params", bson.TypeEmbeddedDocument)
			if time.Since(r.CreatedAt) > time.Minute {
				t.Fatalf("createdAt %v no es de ahora", r.CreatedAt)
			}
			want := shared_domain.ActivityRecord{
				ID:          r.ID,
				Description: "BOND_BOUND_PURCHASED",
				ModuleID:    "1",
				Name:        "Blooming",
				RelatedID:   meteorBondID,
				AgencyID:    meteorAgencyID,
				UserID:      uuidUserID,
				Params:      tt.wantParams,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.CreatedAt,
			}
			if !reflect.DeepEqual(r, want) {
				t.Fatalf("activity:\n got  %+v\n want %+v", r, want)
			}
		})
	}
}
