//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"slices"
	"testing"
	"time"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// La línea de tiempo trae las compras no borradas del usuario por fecha inicial ascendente, como
// "<producto> (<clave>)" y la fecha inicial en la zona de la sesión con el año calendario, como inicio y fin:
// el 30 de diciembre de 2025 sigue en 2025 aunque su semana ISO sea de 2026. Sin producto, el texto queda
// vacío. CA-06.12
func TestTimeline(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	at := func(s string) time.Time {
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("fecha de prueba inválida %q: %v", s, err)
		}
		return v
	}
	newYearEve := boundDoc(nextBoundID(), typeID, "BP12-1", bson.M{"initialDate": at("2026-12-31T20:00:00Z")})
	isoWeek := boundDoc(nextBoundID(), typeID, "BP12-2", bson.M{"initialDate": at("2025-12-30T15:00:00Z")})
	legacy := boundDoc(legacyBoundID, legacyTypeID, "BC06-3", bson.M{"bondObjectId": legacyObjectID, "initialDate": at("2027-01-01T03:00:00Z")})
	noObject := boundDoc(nextBoundID(), typeID, "BX-4", bson.M{"bondObjectId": missingID, "initialDate": at("2028-01-01T12:00:00Z")})
	insert(t, boundCollection, newYearEve, isoWeek, legacy, noObject,
		boundDoc(nextBoundID(), typeID, "BP12-5", bson.M{"isRemove": true, "initialDate": at("2026-06-01T12:00:00Z")}),
		boundDoc(nextBoundID(), typeID, "BP12-6", bson.M{"userId": user2ID, "initialDate": at("2026-06-01T12:00:00Z")}))
	item := func(doc bson.M, text, date string) bondBound_ports.TimelineItem {
		return bondBound_ports.TimelineItem{ID: doc["_id"].(string), Text: text, Start: date, End: date}
	}

	tests := []struct {
		name    string
		utcDiff int
		want    []bondBound_ports.TimelineItem
	}{
		{name: "UTC-5", utcDiff: -5, want: []bondBound_ports.TimelineItem{
			item(isoWeek, "Bono Plus 12M (BP12-2)", "2025-12-30 10:00"),
			item(newYearEve, "Bono Plus 12M (BP12-1)", "2026-12-31 15:00"),
			item(legacy, "Bono Clásico 6M (BC06-3)", "2026-12-31 22:00"),
			item(noObject, "", "2028-01-01 07:00"),
		}},
		{name: "UTC+14", utcDiff: 14, want: []bondBound_ports.TimelineItem{
			item(isoWeek, "Bono Plus 12M (BP12-2)", "2025-12-31 05:00"),
			item(newYearEve, "Bono Plus 12M (BP12-1)", "2027-01-01 10:00"),
			item(legacy, "Bono Clásico 6M (BC06-3)", "2027-01-01 17:00"),
			item(noObject, "", "2028-01-02 02:00"),
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items, err := repo.Timeline(context.Background(), userID, tt.utcDiff)
			if err != nil {
				t.Fatalf("Timeline: %v", err)
			}

			if !slices.Equal(items, tt.want) {
				t.Errorf("línea de tiempo %+v, se esperaba %+v", items, tt.want)
			}
		})
	}

	t.Run("usuario sin compras", func(t *testing.T) {
		items, err := repo.Timeline(context.Background(), missingID, -5)

		if err != nil || items == nil || len(items) != 0 {
			t.Errorf("Timeline = %+v, %v, se esperaba una lista vacía", items, err)
		}
	})
}
