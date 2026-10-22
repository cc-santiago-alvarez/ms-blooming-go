//go:build integration

package bondObject_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Con dos tipos con productos, FindGroupedByType devuelve 2 grupos, cada uno con su bondType completo y
// su lista por name, en una sola agregación y sin consultar bondType aparte; el grupo de un tipo borrado
// queda con bondType nil. CA-02.2
func TestFindGroupedByType(t *testing.T) {
	ctx := context.Background()

	t.Run("dos tipos", func(t *testing.T) {
		repo := setup(t)
		insert(t, typeCollection,
			typeDoc(typeID, "Bono Plus", nil),
			typeDoc(legacyTypeID, "Ahorro", bson.M{"agencyId": agencyA}),
		)
		insert(t, objectCollection,
			objectDoc(meteor1, "Zeta", typeID, bson.A{agencyA}, nil),
			objectDoc(uuid1, "Alfa", typeID, agencyA, nil),
			objectDoc(uuid2, "Beta", legacyTypeID, bson.A{agencyA}, nil),
			objectDoc(meteor2, "Borrado", legacyTypeID, bson.A{agencyA}, bson.M{"isRemove": true}),
			objectDoc(meteor3, "De B", typeID, bson.A{agencyB}, nil),
		)
		commands.reset()

		groups, err := repo.FindGroupedByType(ctx, []string{agencyA})

		if err != nil {
			t.Fatalf("FindGroupedByType() = %v", err)
		}
		want := []struct {
			typeID string
			name   string
			list   []string
		}{
			{typeID: legacyTypeID, name: "Ahorro", list: []string{uuid2}},
			{typeID: typeID, name: "Bono Plus", list: []string{uuid1, meteor1}},
		}
		if len(groups) != len(want) {
			t.Fatalf("FindGroupedByType() devolvió %d grupos, se esperaban %d: %+v", len(groups), len(want), groups)
		}
		for i, w := range want {
			g := groups[i]
			if g.BondTypeID != w.typeID {
				t.Fatalf("grupo %d con _id %q, se esperaba %q", i, g.BondTypeID, w.typeID)
			}
			bt := g.BondType
			if bt == nil {
				t.Fatalf("grupo %s sin bondType", w.typeID)
			}
			if bt.ID != w.typeID || bt.Name != w.name || bt.Description != "Tipo "+w.name || !bt.Active ||
				!slices.Equal(bt.AgencyID, []string{agencyA}) || !bt.CreatedAt.Equal(fixtureAt) {
				t.Errorf("bondType del grupo %s = %+v, se esperaba el tipo completo %q", w.typeID, *bt, w.name)
			}
			got := make([]string, len(g.List))
			for j, o := range g.List {
				got[j] = o.ID
			}
			if !slices.Equal(got, w.list) {
				t.Errorf("list del grupo %s = %v, se esperaba %v", w.typeID, got, w.list)
			}
		}

		if n := len(commands.sent("aggregate", objectCollection)); n != 1 {
			t.Errorf("se mandaron %d aggregate sobre %s, se esperaba 1", n, objectCollection)
		}
		for _, name := range []string{"find", "aggregate"} {
			if n := len(commands.sent(name, typeCollection)); n != 0 {
				t.Errorf("se mandaron %d %s sobre %s, se esperaba ninguno", n, name, typeCollection)
			}
		}
	})

	t.Run("tipo borrado", func(t *testing.T) {
		repo := setup(t)
		insert(t, typeCollection, typeDoc(typeID, "Bono Plus", bson.M{"isRemove": true}))
		insert(t, objectCollection, objectDoc(uuid1, "Alfa", typeID, bson.A{agencyA}, nil))

		groups, err := repo.FindGroupedByType(ctx, nil)

		if err != nil {
			t.Fatalf("FindGroupedByType() = %v", err)
		}
		if len(groups) != 1 || groups[0].BondTypeID != typeID || groups[0].BondType != nil || len(groups[0].List) != 1 {
			t.Fatalf("FindGroupedByType() = %+v, se esperaba un grupo %s con bondType nil y un producto", groups, typeID)
		}
	})
}
