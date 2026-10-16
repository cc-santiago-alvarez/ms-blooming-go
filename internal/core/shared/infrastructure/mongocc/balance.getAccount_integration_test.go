//go:build integration

package shared_mongocc_repository_test

import (
	"context"
	"reflect"
	"testing"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// GetAccount devuelve la cuenta solo a su dueño; ajena, borrada o inexistente da nil, nil. CA-00.29
func TestGetAccount(t *testing.T) {
	ctx := context.Background()

	for _, a := range accountFixtures {
		t.Run(a.name+"/cuenta propia", func(t *testing.T) {
			db := setup(t)
			insertAccount(t, db, a, activeAccount)

			got, err := newBalanceRepo(db).GetAccount(ctx, a.id, a.userID)

			if err != nil {
				t.Fatalf("GetAccount() = %v", err)
			}
			want := &shared_domain.BalanceAccount{
				ID:        a.id,
				UserID:    a.userID,
				AgencyID:  a.agencyID,
				StatusID:  "1",
				Available: 1000,
				Number:    accountNumber,
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("GetAccount() = %+v, se esperaba %+v", got, want)
			}
		})

		tests := []struct {
			name      string
			isRemove  bool
			accountID string
			userID    string
		}{
			{name: "cuenta de otro usuario", accountID: a.id, userID: otherUserID},
			{name: "cuenta borrada", isRemove: true, accountID: a.id, userID: a.userID},
			{name: "cuenta inexistente", accountID: missingID, userID: a.userID},
		}
		for _, tt := range tests {
			t.Run(a.name+"/"+tt.name, func(t *testing.T) {
				db := setup(t)
				state := activeAccount
				state.isRemove = tt.isRemove
				insertAccount(t, db, a, state)

				got, err := newBalanceRepo(db).GetAccount(ctx, tt.accountID, tt.userID)

				if err != nil || got != nil {
					t.Fatalf("GetAccount() = (%+v, %v), se esperaba (nil, nil)", got, err)
				}
			})
		}
	}
}
