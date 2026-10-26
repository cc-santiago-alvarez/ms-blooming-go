//go:build integration

package bondBound_mongocc_repository_test

import (
	"context"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// El resumen suma por usuario los montos de los bonos no borrados de la agencia pedida, del mayor al menor,
// con el nombre del perfil, o sin profile si no tiene. Dos bonos iniciados de 1000 y "500" (string, como
// algunos legacy) dan 1500, con _id Meteor y UUID por igual; '0' suma todos los estados y '2' solo los
// madurados. CA-07.3, CA-07.8
func TestReportSummaryRowsGroupsByUser(t *testing.T) {
	repo := setup(t)
	seedCatalog(t)
	noProfileID := shared_domain.NewID()
	insert(t, boundCollection,
		boundAt(shared_domain.NewID(), 0, nil),
		boundAt(legacyBoundID, 1, bson.M{"amount": "500"}),
		boundAt(legacyBound2ID, 2, bson.M{"statusId": "2", "amount": 200.0}),
		boundAt(boundID, 3, bson.M{"userId": user2ID, "amount": 700.0}),
		boundAt(nextBoundID(), 4, bson.M{"userId": noProfileID, "amount": 300.0}),
		boundAt(nextBoundID(), 5, bson.M{"isRemove": true, "amount": 9999.0}),
		boundAt(nextBoundID(), 6, bson.M{"agencyId": agencyB, "amount": 7777.0}))
	carlos := &bondBound_ports.ProfileName{FirstName: "Carlos", LastName: "Ramírez"}
	lucia := &bondBound_ports.ProfileName{FirstName: "Lucía", LastName: "Torres"}

	tests := []struct {
		name     string
		statusID bondBound_domain.BondBoundStatus
		want     []bondBound_ports.SummaryReportRow
	}{
		{name: "iniciados", statusID: bondBound_domain.StatusInitiated, want: []bondBound_ports.SummaryReportRow{
			{UserID: userID, Total: 1500, Profile: carlos},
			{UserID: user2ID, Total: 700, Profile: lucia},
			{UserID: noProfileID, Total: 300},
		}},
		{name: "todos los estados", statusID: bondBound_domain.ReportAllStatuses, want: []bondBound_ports.SummaryReportRow{
			{UserID: userID, Total: 1700, Profile: carlos},
			{UserID: user2ID, Total: 700, Profile: lucia},
			{UserID: noProfileID, Total: 300},
		}},
		{name: "madurados", statusID: bondBound_domain.StatusFinalized, want: []bondBound_ports.SummaryReportRow{
			{UserID: userID, Total: 200, Profile: carlos},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows, err := repo.ReportSummaryRows(context.Background(), agencyA, bondBound_domain.ReportFilters{StatusID: tt.statusID})
			if err != nil {
				t.Fatalf("ReportSummaryRows: %v", err)
			}

			assertJSON(t, rows, tt.want)
		})
	}
}

// Sin bonos, el resumen es una lista vacía, no null, y no falla. CA-07.3
func TestReportSummaryRowsEmpty(t *testing.T) {
	repo := setup(t)

	rows, err := repo.ReportSummaryRows(context.Background(), agencyA, bondBound_domain.ReportFilters{StatusID: bondBound_domain.StatusInitiated})
	if err != nil {
		t.Fatalf("ReportSummaryRows: %v", err)
	}

	assertJSON(t, rows, []bondBound_ports.SummaryReportRow{})
}
