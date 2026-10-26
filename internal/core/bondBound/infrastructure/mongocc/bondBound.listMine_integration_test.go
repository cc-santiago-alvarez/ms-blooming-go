//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Con 12 bonos propios iniciados o madurados, la página 2 de 5 trae los bonos 6 a 10 por createdAt
// descendente, con total 12 y 3 páginas, el nombre de su producto y lo que usan los derivados. No salen los
// de otro usuario, los de otra agencia, los borrados ni los que no tienen producto; sin estados se suman los
// bloqueados. CA-06.8
func TestListMinePage(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	var mine []bson.M
	for i := range 12 {
		extra := bson.M{"statusId": "1"}
		if i%2 == 1 {
			extra["statusId"] = "2"
		}
		if i%3 == 0 {
			extra["bondObjectId"] = legacyObjectID
		}
		mine = append(mine, boundAt(nextBoundID(), i, extra))
	}
	insert(t, boundCollection, mine...)
	insert(t, boundCollection,
		boundAt(nextBoundID(), 20, bson.M{"statusId": "3"}),
		boundAt(nextBoundID(), 21, bson.M{"statusId": "3"}),
		boundAt(nextBoundID(), 22, bson.M{"userId": user2ID}),
		boundAt(nextBoundID(), 23, bson.M{"agencyId": agencyB}),
		boundAt(nextBoundID(), 24, bson.M{"isRemove": true}),
		boundAt(nextBoundID(), 25, bson.M{"bondObjectId": missingID}))
	statuses := []bondBound_domain.BondBoundStatus{bondBound_domain.StatusInitiated, bondBound_domain.StatusFinalized}

	page, err := repo.ListMine(context.Background(), userID, agencyA, statuses, shared_domain.Pagination{Page: 2, Limit: 5})
	if err != nil {
		t.Fatalf("ListMine: %v", err)
	}

	want := ids(mine[6], mine[5], mine[4], mine[3], mine[2])
	if got := withObjectIDs(page.Items); !slices.Equal(got, want) {
		t.Errorf("bonos %v, se esperaban %v", got, want)
	}
	if page.Total != 12 || page.Page != 2 || page.Limit != 5 || page.TotalPages != 3 {
		t.Errorf("total %d, page %d, limit %d y totalPages %d, se esperaban 12, 2, 5 y 3",
			page.Total, page.Page, page.Limit, page.TotalPages)
	}
	for _, row := range page.Items {
		wantName := "Bono Plus 12M"
		if row.Bound.BondObjectID == legacyObjectID {
			wantName = "Bono Clásico 6M"
		}
		if row.BondObject == nil || row.BondObject.Name != wantName {
			t.Errorf("bono %s con producto %+v, se esperaba %q", row.Bound.ID, row.BondObject, wantName)
		}
		b := row.Bound
		if b.InitialDate.IsZero() || !b.MaturityDate.Equal(fixtureAt.AddDate(0, 0, 365)) ||
			b.EarlyCashTerm.InflectionPoint != 90 || b.MaturityTimeInDays != 365 {
			t.Errorf("bono %s con initialDate %v, maturityDate %v, inflexión %d y plazo %d, se esperaba el bono completo",
				b.ID, b.InitialDate, b.MaturityDate, b.EarlyCashTerm.InflectionPoint, b.MaturityTimeInDays)
		}
	}

	all, err := repo.ListMine(context.Background(), userID, agencyA, nil, shared_domain.Pagination{Page: 1, Limit: 100})
	if err != nil {
		t.Fatalf("ListMine sin estados: %v", err)
	}
	if all.Total != 14 {
		t.Errorf("sin estados: total %d, se esperaba 14", all.Total)
	}
}

// Un bono legacy, con _id Meteor, numéricos como string y producto legacy, sale junto a uno nuevo con UUID
// y keySeq, cada uno con sus valores y su producto. CA-06.14
func TestListMineLegacyAndNew(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	legacy := boundDoc(legacyBoundID, legacyTypeID, "BC06-3", bson.M{
		"bondObjectId":              legacyObjectID,
		"amount":                    "1000",
		"maturityTimeInDays":        "180",
		"maturityRewardsPercentage": "7.5",
	})
	newer := boundDoc(boundID, typeID, "BP12-9", bson.M{
		"keySeq":      9,
		"createdAt":   fixtureAt.Add(time.Hour),
		"initialDate": fixtureAt.Add(time.Hour),
	})
	insert(t, boundCollection, legacy, newer)

	page, err := repo.ListMine(context.Background(), userID, agencyA, nil, shared_domain.Pagination{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("ListMine: %v", err)
	}

	if got, want := withObjectIDs(page.Items), []string{boundID, legacyBoundID}; !slices.Equal(got, want) {
		t.Fatalf("bonos %v, se esperaban %v", got, want)
	}
	n, l := page.Items[0], page.Items[1]
	if n.Bound.KeySeq == nil || *n.Bound.KeySeq != 9 || n.BondObject == nil || n.BondObject.Name != "Bono Plus 12M" {
		t.Errorf("bono nuevo con keySeq %v y producto %+v, se esperaban 9 y Bono Plus 12M", n.Bound.KeySeq, n.BondObject)
	}
	if l.Bound.Amount != 1000 || l.Bound.MaturityTimeInDays != 180 || l.Bound.MaturityRewardsPercentage != 7.5 || l.Bound.KeySeq != nil {
		t.Errorf("bono legacy con amount %v, plazo %d, tasa %v y keySeq %v, se esperaban 1000, 180, 7.5 y nil",
			l.Bound.Amount, l.Bound.MaturityTimeInDays, l.Bound.MaturityRewardsPercentage, l.Bound.KeySeq)
	}
	if l.BondObject == nil || l.BondObject.Name != "Bono Clásico 6M" {
		t.Errorf("bono legacy con producto %+v, se esperaba Bono Clásico 6M", l.BondObject)
	}
}
