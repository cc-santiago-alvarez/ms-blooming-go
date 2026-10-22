//go:build integration

package bondObject_mongocc_repository_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	bondObject_domain "github.com/futurapps/ms-blooming-go/internal/core/bondObject/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Un usuario con bonos del producto X en estados 1, 5 y 6 aparece con numOfBounds 2 y su perfil público,
// sin password; no cuentan los bonos borrados ni los de otro producto, y los de otra agencia solo cuentan
// sin filtro de agencia. CA-02.10
func TestInvestorsByObject(t *testing.T) {
	ctx := context.Background()
	setup(t)
	x, other := meteor1, uuid1
	insert(t, profileCollection, profileDoc(userID, "Ana"), profileDoc(user2ID, "Luis"))
	insert(t, boundCollection,
		boundDoc(nextBoundID(), x, userID, agencyA, "1", nil),
		boundDoc(nextBoundID(), x, userID, agencyA, "5", nil),
		boundDoc(nextBoundID(), x, userID, agencyA, "6", nil),
		boundDoc(nextBoundID(), x, userID, agencyA, "1", bson.M{"isRemove": true}),
		boundDoc(nextBoundID(), other, userID, agencyA, "1", nil),
		boundDoc(nextBoundID(), x, user2ID, agencyA, "2", nil),
		boundDoc(nextBoundID(), x, user2ID, agencyB, "3", nil),
	)
	agency := agencyA

	tests := []struct {
		name     string
		agencyID *string
		want     map[string]int
	}{
		{name: "agencia A", agencyID: &agency, want: map[string]int{userID: 2, user2ID: 1}},
		{name: "sin filtro de agencia", agencyID: nil, want: map[string]int{userID: 2, user2ID: 2}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := stats().InvestorsByObject(ctx, x, tt.agencyID)
			if err != nil {
				t.Fatalf("InvestorsByObject() = %v", err)
			}
			// ordenados por _id: el UUID de user2ID va antes que el Meteor de userID
			if len(got) != 2 || got[0].UserID != user2ID || got[1].UserID != userID {
				t.Fatalf("InvestorsByObject() = %+v, se esperaban %s y %s en ese orden", got, user2ID, userID)
			}
			for _, inv := range got {
				if inv.NumOfBounds != tt.want[inv.UserID] {
					t.Errorf("numOfBounds de %s = %d, se esperaba %d", inv.UserID, inv.NumOfBounds, tt.want[inv.UserID])
				}
			}
			ana := got[1].Investor
			wantAna := bondObject_domain.InvestorProfile{
				PublicID:  "CC-Ana",
				FirstName: "Ana",
				LastName:  "Gómez",
				Email:     "ana@futurapps.com",
				Avatar:    "avatar-" + userID,
				Address:   "Calle 10 # 20-30",
				AvatarURL: "https://cdn.futurapps.com/avatars/" + userID + ".png",
			}
			if ana != wantAna {
				t.Errorf("investor de %s = %+v, se esperaba %+v", userID, ana, wantAna)
			}

			body, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("json.Marshal() = %v", err)
			}
			if strings.Contains(string(body), "password") || strings.Contains(string(body), passwordHash) {
				t.Fatalf("la respuesta trae el password: %s", body)
			}
		})
	}
}
