//go:build integration

package bondHistory_mongocc_repository_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"

	bondHistory_domain "github.com/futurapps/ms-blooming-go/internal/core/bondHistory/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// El historial de un producto o de un bono comprado trae solo los registros de su tipo y su relatedId en la
// agencia de la sesión, sin los borrados, del más reciente al más antiguo. CA-08.3, CA-08.4
func TestListByRelatedFilters(t *testing.T) {
	tests := []struct {
		name      string
		typeID    bondHistory_domain.HistoryType
		relatedID string
		other     bondHistory_domain.HistoryType
	}{
		{name: "producto", typeID: bondHistory_domain.HistoryBond, relatedID: objectID, other: bondHistory_domain.HistoryBondBound},
		{name: "bono comprado", typeID: bondHistory_domain.HistoryBondBound, relatedID: boundID, other: bondHistory_domain.HistoryBond},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setup(t)
			at := func(hours int) time.Time { return fixtureAt.Add(time.Duration(hours) * time.Hour) }
			// los del listado, guardados desordenados
			middle := msHistory(t, tt.typeID, tt.relatedID, agencyA, userID, "BOND_UPDATED", nil, at(2))
			oldest := msHistory(t, tt.typeID, tt.relatedID, agencyA, userID, "BOND_ADDED", nil, at(0))
			newest := msHistory(t, tt.typeID, tt.relatedID, agencyA, userID, "BOND_UPDATED", nil, at(4))
			// los que no
			msHistory(t, tt.other, tt.relatedID, agencyA, userID, "BOND_UPDATED", nil, at(1))
			msHistory(t, tt.typeID, otherID, agencyA, userID, "BOND_UPDATED", nil, at(1))
			msHistory(t, tt.typeID, tt.relatedID, agencyB, userID, "BOND_UPDATED", nil, at(3))
			removed := msHistory(t, tt.typeID, tt.relatedID, agencyA, userID, "BOND_UPDATED", nil, at(3))
			if _, err := testDB.UpdateOne(context.Background(), historyCollection, bson.M{"_id": removed},
				bson.M{"$set": bson.M{"isRemove": true}}, nil); err != nil {
				t.Fatalf("no se pudo borrar %s: %v", removed, err)
			}

			entries, err := newApp().ListByRelated(adminSession(agencyA), tt.typeID, tt.relatedID)

			if err != nil {
				t.Fatalf("ListByRelated: %v", err)
			}
			if want := []string{newest, middle, oldest}; !slices.Equal(entryIDs(entries), want) {
				t.Fatalf("registros %v, se esperaban %v", entryIDs(entries), want)
			}
			for _, e := range entries {
				if e.TypeID != tt.typeID || e.RelatedID != tt.relatedID || e.AgencyID != agencyA {
					t.Errorf("registro %s con typeId %q, relatedId %q y agencyId %q, se esperaban %q, %q y %q",
						e.ID, e.TypeID, e.RelatedID, e.AgencyID, tt.typeID, tt.relatedID, agencyA)
				}
			}
		})
	}
}

// Los registros que escribió Meteor para un bono legacy se listan junto a los que escribe el ms para el
// mismo relatedId, intercalados por createdAt, con su texto y su autor. CA-08.9
func TestListByRelatedLegacy(t *testing.T) {
	setup(t)
	insert(t, profileCollection, profileDoc(userID, "Laura", "Gómez"), profileDoc(adminID, "Mario", "Rojas"))
	day := func(n int) time.Time { return fixtureAt.AddDate(0, 0, n) }
	insert(t, historyCollection,
		meteorHistoryDoc(meteorSoldID, legacyBoundID, userID, "BOND_BOUND_SOLD", []string{}, day(0), nil),
		meteorHistoryDoc(meteorChangedID, legacyBoundID, adminID, "BOND_BOUND_STATUS_CHANGED", []string{"2", "3"}, day(2),
			bson.M{"changes": nil, "editedProperties": nil}),
	)
	matured := msHistory(t, bondHistory_domain.HistoryBondBound, legacyBoundID, agencyA, "",
		"BOND_BOUND_STATUS_CHANGED", []string{"1", "2"}, day(1))
	unblocked := msHistory(t, bondHistory_domain.HistoryBondBound, legacyBoundID, agencyA, adminID,
		"BOND_BOUND_STATUS_CHANGED", []string{"3", "1"}, day(3))

	entries, err := newApp().ListByRelated(adminSession(agencyA), bondHistory_domain.HistoryBondBound, legacyBoundID)

	if err != nil {
		t.Fatalf("ListByRelated: %v", err)
	}
	want := []struct {
		id   string
		text string
		user bondHistory_domain.HistoryUser
	}{
		{id: unblocked, text: "Estado cambiado de Bloqueado a Iniciado", user: bondHistory_domain.HistoryUser{FirstName: "Mario", LastName: "Rojas"}},
		{id: meteorChangedID, text: "Estado cambiado de Madurado a Bloqueado", user: bondHistory_domain.HistoryUser{FirstName: "Mario", LastName: "Rojas"}},
		{id: matured, text: "Estado cambiado de Iniciado a Madurado", user: bondHistory_domain.HistoryUser{System: true}},
		{id: meteorSoldID, text: "Bono Vendido", user: bondHistory_domain.HistoryUser{FirstName: "Laura", LastName: "Gómez"}},
	}
	if len(entries) != len(want) {
		t.Fatalf("registros %v, se esperaban %d", entryIDs(entries), len(want))
	}
	for i, w := range want {
		e := entries[i]
		if e.ID != w.id || e.Text != w.text || e.User != w.user || e.RelatedID != legacyBoundID {
			t.Errorf("registro %d = %s %q de %+v, se esperaba %s %q de %+v", i, e.ID, e.Text, e.User, w.id, w.text, w.user)
		}
	}
}

// Un historial de 50 registros de 7 autores lee sus perfiles con una sola consulta a profile, y cada
// registro trae el nombre de su autor. CA-08.8
func TestListByRelatedReadsProfilesOnce(t *testing.T) {
	setup(t)
	authors := make([]string, 7)
	for i := range authors {
		authors[i] = shared_domain.NewID()
		insert(t, profileCollection, profileDoc(authors[i], fmt.Sprintf("Autor%d", i), "Pérez"))
	}
	authorOf := map[string]int{}
	for i := range 50 {
		id := msHistory(t, bondHistory_domain.HistoryBond, objectID, agencyA, authors[i%len(authors)],
			"BOND_UPDATED", nil, fixtureAt.Add(time.Duration(i)*time.Minute))
		authorOf[id] = i % len(authors)
	}
	commands.reset()

	entries, err := newApp().ListByRelated(adminSession(agencyA), bondHistory_domain.HistoryBond, objectID)

	if err != nil {
		t.Fatalf("ListByRelated: %v", err)
	}
	if len(entries) != 50 {
		t.Fatalf("ListByRelated devolvió %d registros, se esperaban 50", len(entries))
	}
	if profiles, histories := commands.count("find", profileCollection), commands.count("find", historyCollection); profiles != 1 || histories != 1 {
		t.Errorf("se mandaron %d find a profile y %d a bondHistory, se esperaba uno a cada una", profiles, histories)
	}
	for _, e := range entries {
		if want := fmt.Sprintf("Autor%d", authorOf[e.ID]); e.User.FirstName != want || e.User.System {
			t.Errorf("el registro %s trae el autor %+v, se esperaba %s", e.ID, e.User, want)
		}
	}
}
