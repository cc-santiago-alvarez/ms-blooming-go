package bondBound_domain_test

import (
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

const (
	objectID = "7c1d2e3f-4a5b-4c6d-8e7f-9a0b1c2d3e4f"
	typeID   = "3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f"
	userID   = "user1AAAAAAAAAAAA"
	agencyID = "agencyAAAAAAAAAAA"
)

// snapshot es un producto de 365 días con clave BP12.
func snapshot() bondBound_domain.ObjectSnapshot {
	return bondBound_domain.ObjectSnapshot{
		BondObjectID:              objectID,
		BondTypeID:                typeID,
		Key:                       "BP12",
		Amount:                    1000,
		MaturityTimeInDays:        365,
		MaturityRewardsPercentage: 15,
		EarlyCashTerm: shared_domain.EarlyCashTerm{
			InitialPenaltyPercentage:    100,
			InflectionPoint:             90,
			InflectionPenaltyPercentage: 75,
			InitialDayForWithdraw:       30,
		},
	}
}

// Una compra de la tarde vence a la misma hora exacta maturityTimeInDays días después, en UTC, aunque now
// venga en otra zona. CA-04.8
func TestNewBoundFromObjectMaturityDate(t *testing.T) {
	purchasedAt := time.Date(2026, 10, 13, 20, 30, 0, 0, time.UTC)
	wantMaturity := time.Date(2027, 10, 13, 20, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		now  time.Time
	}{
		{name: "now en UTC", now: purchasedAt},
		{name: "now en UTC-5", now: purchasedAt.In(time.FixedZone("", -5*3600))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := bondBound_domain.NewBoundFromObject(snapshot(), bondBound_domain.Buyer{UserID: userID, AgencyID: agencyID}, 1, "Ab3dE6", tt.now)

			if !b.MaturityDate.Equal(wantMaturity) || b.MaturityDate.Location() != time.UTC {
				t.Errorf("maturityDate %v, se esperaba %v en UTC", b.MaturityDate, wantMaturity)
			}
			if !b.InitialDate.Equal(purchasedAt) || b.InitialDate.Location() != time.UTC {
				t.Errorf("initialDate %v, se esperaba %v en UTC", b.InitialDate, purchasedAt)
			}
		})
	}
}

// El bono copia del producto los ids y los términos, y del comprador el usuario y la agencia; nace iniciado
// con la clave "<key>-<seq>", el token recibido y un _id UUID v4. Los ids legacy se copian tal cual. CA-04.8
func TestNewBoundFromObjectSnapshot(t *testing.T) {
	now := time.Date(2026, 10, 13, 20, 30, 0, 0, time.UTC)
	buyer := bondBound_domain.Buyer{UserID: userID, AgencyID: agencyID}

	t.Run("producto nuevo", func(t *testing.T) {
		obj := snapshot()

		b := bondBound_domain.NewBoundFromObject(obj, buyer, 59, "Ab3dE6", now)

		if shared_domain.IDFormat(b.ID) != shared_domain.IDKindUUID {
			t.Errorf("_id %q, se esperaba un UUID v4", b.ID)
		}
		if b.BondObjectID != objectID || b.BondTypeID != typeID || b.UserID != userID || b.AgencyID != agencyID {
			t.Errorf("bondObjectId %q, bondTypeId %q, userId %q y agencyId %q, se esperaban %q, %q, %q y %q",
				b.BondObjectID, b.BondTypeID, b.UserID, b.AgencyID, objectID, typeID, userID, agencyID)
		}
		if b.Amount != obj.Amount || b.MaturityTimeInDays != obj.MaturityTimeInDays ||
			b.MaturityRewardsPercentage != obj.MaturityRewardsPercentage || b.EarlyCashTerm != obj.EarlyCashTerm {
			t.Errorf("términos %v %d %v %+v, se esperaba la copia del producto %+v",
				b.Amount, b.MaturityTimeInDays, b.MaturityRewardsPercentage, b.EarlyCashTerm, obj)
		}
		if b.Key != "BP12-59" || b.KeySeq == nil || *b.KeySeq != 59 {
			t.Errorf("key %q y keySeq %v, se esperaban \"BP12-59\" y 59", b.Key, b.KeySeq)
		}
		if b.Token != "Ab3dE6" || b.StatusID != bondBound_domain.StatusInitiated || b.IsRemove {
			t.Errorf("token %q, statusId %q e isRemove %v, se esperaban \"Ab3dE6\", iniciado y false",
				b.Token, b.StatusID, b.IsRemove)
		}
		if !b.CreatedAt.Equal(now) || !b.UpdatedAt.Equal(now) || b.SoldDate != nil || b.TookDate != nil || b.Matured {
			t.Errorf("createdAt %v, updatedAt %v, soldDate %v, tookDate %v y matured %v, se esperaba un bono recién comprado a %v",
				b.CreatedAt, b.UpdatedAt, b.SoldDate, b.TookDate, b.Matured, now)
		}
	})

	t.Run("producto legacy", func(t *testing.T) {
		obj := snapshot()
		obj.BondObjectID = "Bd9Xc4Vn7Mz2Lk5Jh"
		obj.BondTypeID = "Tp4Hs8Nc2Vb6Qw3Er"

		b := bondBound_domain.NewBoundFromObject(obj, buyer, 8, "Ab3dE6", now)

		if b.BondObjectID != obj.BondObjectID || b.BondTypeID != obj.BondTypeID {
			t.Errorf("bondObjectId %q y bondTypeId %q, se esperaban los legacy %q y %q",
				b.BondObjectID, b.BondTypeID, obj.BondObjectID, obj.BondTypeID)
		}
		if shared_domain.IDFormat(b.ID) != shared_domain.IDKindUUID || b.Key != "BP12-8" {
			t.Errorf("_id %q y key %q, se esperaban un UUID v4 y \"BP12-8\"", b.ID, b.Key)
		}
	})

	t.Run("cada bono tiene su _id", func(t *testing.T) {
		a := bondBound_domain.NewBoundFromObject(snapshot(), buyer, 1, "Ab3dE6", now)
		b := bondBound_domain.NewBoundFromObject(snapshot(), buyer, 2, "Ab3dE6", now)

		if a.ID == b.ID {
			t.Errorf("dos bonos con el mismo _id %q", a.ID)
		}
	})
}
