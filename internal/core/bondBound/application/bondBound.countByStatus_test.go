package bondBound_application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// withMainInvestor es un context con la sesión de ownerID, sin admin, en la agencia principal A, en UTC-5.
func withMainInvestor() context.Context {
	return session_domain.SetUserSession(context.Background(), &session_domain.UserSession{
		UserID:   ownerID,
		AgencyID: agencyA,
		Agency:   session_domain.SessionAgency{ID: agencyA, IsMain: true},
		UtcDiff:  session_domain.OptionalUtcDiff{Hours: -5, Valid: true},
	})
}

// Un no admin siempre cuenta sus propios bonos: sin userId se usa el suyo, y con el de otro usuario es
// ErrForbidden sin consultar. Un admin cuenta los del userId pedido o, sin él, los de su alcance. El alcance es
// la agencia de la sesión, o todas si es la principal. CA-07.6
func TestCountByStatusScope(t *testing.T) {
	tests := []struct {
		name       string
		ctx        context.Context
		userID     string
		wantUser   string
		wantAgency *string
		wantErr    error
	}{
		{name: "no admin sin userId", ctx: withSession(ownerID, -5), wantUser: ownerID, wantAgency: agency(agencyA)},
		{name: "no admin con su userId", ctx: withSession(ownerID, -5), userID: ownerID, wantUser: ownerID, wantAgency: agency(agencyA)},
		{name: "no admin con userId ajeno", ctx: withSession(ownerID, -5), userID: otherUserID, wantErr: shared_domain.ErrForbidden},
		{name: "no admin de la agencia principal", ctx: withMainInvestor(), wantUser: ownerID},
		{name: "admin sin userId", ctx: withAdmin(), wantUser: "", wantAgency: agency(agencyA)},
		{name: "admin con userId ajeno", ctx: withAdmin(), userID: otherUserID, wantUser: otherUserID, wantAgency: agency(agencyA)},
		{name: "admin de la agencia principal", ctx: withMainAdmin(), wantUser: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			f.repo.statusCount = bondBound_ports.StatusCount{Total: 125000, Count: 120}

			count, err := f.app().CountByStatus(tt.ctx, bondBound_domain.KpiFilters{UserID: tt.userID})

			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) || count != (bondBound_ports.StatusCount{}) {
					t.Fatalf("CountByStatus = %+v, %v, se esperaba %v", count, err, tt.wantErr)
				}
				assertNoQueries(t, f)
				return
			}
			if err != nil {
				t.Fatalf("CountByStatus: %v", err)
			}
			want := []repoQuery{{
				method:     "CountByStatus",
				agencyID:   tt.wantAgency,
				kpiFilters: bondBound_domain.KpiFilters{UserID: tt.wantUser},
				utcDiff:    -5,
			}}
			if !reflect.DeepEqual(f.repo.queries, want) {
				t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, want)
			}
			if wantCount := (bondBound_ports.StatusCount{Total: 125000, Count: 120}); count != wantCount {
				t.Errorf("CountByStatus = %+v, se esperaba %+v", count, wantCount)
			}
		})
	}
}

// Un estado conocido se filtra y responde su nombre; vacío o fuera del enum, incluido '0', cuenta todos los
// estados y responde status "". El total y la cantidad son los del repositorio. CA-07.6
func TestCountByStatusStatus(t *testing.T) {
	tests := []struct {
		name       string
		statusID   bondBound_domain.BondBoundStatus
		wantFilter bondBound_domain.BondBoundStatus
		wantName   string
	}{
		{name: "sin statusId", statusID: "", wantFilter: "", wantName: ""},
		{name: "iniciado", statusID: "1", wantFilter: bondBound_domain.StatusInitiated, wantName: "Iniciado"},
		{name: "madurado con espacios", statusID: " 2 ", wantFilter: bondBound_domain.StatusFinalized, wantName: "Madurado"},
		{name: "estado desconocido", statusID: "9", wantFilter: "", wantName: ""},
		{name: "cero", statusID: "0", wantFilter: "", wantName: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			f.repo.statusCount = bondBound_ports.StatusCount{Total: 125000, Count: 120}

			count, err := f.app().CountByStatus(withAdmin(), bondBound_domain.KpiFilters{StatusID: tt.statusID})
			if err != nil {
				t.Fatalf("CountByStatus: %v", err)
			}

			if len(f.repo.queries) != 1 || f.repo.queries[0].kpiFilters.StatusID != tt.wantFilter {
				t.Errorf("consultas %+v, se esperaba el estado %q", f.repo.queries, tt.wantFilter)
			}
			want := bondBound_ports.StatusCount{Total: 125000, Count: 120, Status: tt.wantName}
			if count != want {
				t.Errorf("CountByStatus = %+v, se esperaba %+v", count, want)
			}
		})
	}
}

// La búsqueda y el filtro de fecha llegan recortados al repositorio, que consulta en la zona de la sesión.
// CA-07.6
func TestCountByStatusFiltersAndZone(t *testing.T) {
	f := newFakes()
	filters := bondBound_domain.KpiFilters{
		Search: " plus ",
		Date:   bondBound_domain.DateFilter{Kind: " betweenDates ", StartDate: " 2026-01-25 ", EndDate: " 2026-02-05 "},
	}

	if _, err := f.app().CountByStatus(withSession(ownerID, 14), filters); err != nil {
		t.Fatalf("CountByStatus: %v", err)
	}

	want := []repoQuery{{
		method:   "CountByStatus",
		agencyID: agency(agencyA),
		kpiFilters: bondBound_domain.KpiFilters{
			UserID: ownerID,
			Search: "plus",
			Date:   bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: "2026-01-25", EndDate: "2026-02-05"},
		},
		utcDiff: 14,
	}}
	if !reflect.DeepEqual(f.repo.queries, want) {
		t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, want)
	}
}

// Un userId fuera de formato es ErrInvalidID antes de revisar el permiso, y una búsqueda de más de 100
// caracteres o un filtro de fecha inválido son el error de la validación; ninguno consulta. CA-07.6
func TestCountByStatusInvalidFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters bondBound_domain.KpiFilters
		wantErr error
	}{
		{name: "userId inválido", filters: bondBound_domain.KpiFilters{UserID: "abc"}, wantErr: shared_domain.ErrInvalidID},
		{name: "búsqueda de 101 caracteres", filters: bondBound_domain.KpiFilters{Search: strings.Repeat("a", 101)}, wantErr: bondBound_domain.ErrInvalidSearch},
		{
			name:    "startDate posterior a endDate",
			filters: bondBound_domain.KpiFilters{Date: bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: "2026-02-05", EndDate: "2026-01-25"}},
			wantErr: bondBound_domain.ErrInvalidDateFilter,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()

			_, err := f.app().CountByStatus(withSession(ownerID, -5), tt.filters)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CountByStatus = %v, se esperaba %v", err, tt.wantErr)
			}
			assertNoQueries(t, f)
		})
	}
}

// Sin sesión es un error y no consulta, y un error del repositorio se devuelve sin cambiarlo. CA-07.6
func TestCountByStatusErrors(t *testing.T) {
	t.Run("sin sesión", func(t *testing.T) {
		f := newFakes()

		if _, err := f.app().CountByStatus(context.Background(), bondBound_domain.KpiFilters{}); err == nil {
			t.Fatal("CountByStatus sin sesión no devolvió error")
		}
		assertNoQueries(t, f)
	})

	t.Run("repositorio", func(t *testing.T) {
		f := newFakes()
		f.repo.err = errors.New("mongo caído")

		if _, err := f.app().CountByStatus(withAdmin(), bondBound_domain.KpiFilters{}); !errors.Is(err, f.repo.err) {
			t.Fatalf("CountByStatus = %v, se esperaba el error del repositorio", err)
		}
	})
}
