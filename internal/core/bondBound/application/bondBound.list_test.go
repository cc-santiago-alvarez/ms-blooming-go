package bondBound_application_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// adminRow es el bono de hace 50 días como lo trae el listado de administración, comprado en octubre de
// 2026 en la zona de la sesión.
func adminRow() bondBound_ports.BondBoundAdminRow {
	return bondBound_ports.BondBoundAdminRow{
		Bound:      *boundOnDay(50),
		Profile:    &bondBound_ports.ProfileName{FirstName: "Juan", LastName: "Pérez"},
		BondObject: bondBound_ports.BondObjectRef{Name: "Bono Plus 12M"},
		BondType:   bondBound_ports.BondTypeRef{Name: "Plus"},
		Month:      10,
		Year:       2026,
	}
}

// El listado consulta con los filtros recortados, la página pedida, la zona de la sesión y su alcance: su
// agencia, o todas si es la principal. Cada ítem trae el estado con su nombre, el dueño, el producto, el
// tipo, el mes y el año y lo acumulado a hoy; con 25 bonos y 10 por página son 3 páginas. CA-06.1
func TestListQueryAndItems(t *testing.T) {
	tests := []struct {
		name      string
		ctx       context.Context
		wantScope *string
	}{
		{name: "admin de la agencia A", ctx: withAdmin(), wantScope: agency(agencyA)},
		{name: "admin de la agencia principal", ctx: withMainAdmin()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()
			row := adminRow()
			f.repo.adminPage = shared_domain.Paginated[bondBound_ports.BondBoundAdminRow]{Items: []bondBound_ports.BondBoundAdminRow{row}, Total: 25}
			filters := bondBound_domain.ListFilters{
				StatusID: " 1 ",
				UserID:   " " + ownerID + " ",
				Search:   " juan ",
				Date:     bondBound_domain.DateFilter{Kind: " aMonth ", Month: 10, Year: 2026},
			}

			page, err := f.app().List(tt.ctx, filters, shared_domain.Pagination{Page: 2, Limit: 10})
			if err != nil {
				t.Fatalf("List: %v", err)
			}

			wantQuery := []repoQuery{{
				method:   "ListAdmin",
				agencyID: tt.wantScope,
				filters: bondBound_domain.ListFilters{
					StatusID: bondBound_domain.StatusInitiated,
					UserID:   ownerID,
					Search:   "juan",
					Date:     bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAMonth, Month: 10, Year: 2026},
				},
				page:    shared_domain.Pagination{Page: 2, Limit: 10},
				utcDiff: -5,
			}}
			if !reflect.DeepEqual(f.repo.queries, wantQuery) {
				t.Errorf("consultas %+v, se esperaba %+v", f.repo.queries, wantQuery)
			}
			if page.Total != 25 || page.Page != 2 || page.Limit != 10 || page.TotalPages != 3 {
				t.Errorf("total %d, page %d, limit %d y totalPages %d, se esperaban 25, 2, 10 y 3",
					page.Total, page.Page, page.Limit, page.TotalPages)
			}

			b := row.Bound
			// 1150 / 365 × 50, redondeado
			wantItem := bondBound_ports.BondBoundAdminItem{
				ID:                        b.ID,
				Key:                       b.Key,
				Status:                    bondBound_domain.StatusRef{ID: "1", Name: "Iniciado"},
				Amount:                    1000,
				MaturityTimeInDays:        365,
				InitialDate:               b.InitialDate,
				MaturityRewardsPercentage: 15,
				Profile:                   row.Profile,
				BondType:                  row.BondType,
				BondObject:                row.BondObject,
				CreatedAt:                 b.CreatedAt,
				Month:                     10,
				Year:                      2026,
				DaysPassed:                50,
				AcumulateAmount:           157.53,
			}
			if len(page.Items) != 1 || !reflect.DeepEqual(page.Items[0], wantItem) {
				t.Errorf("ítems %+v, se esperaba [%+v]", page.Items, wantItem)
			}
		})
	}
}

// La página se normaliza antes de consultar: 1 o más, y de 1 a 100 ítems con 10 por defecto. CA-06.1
func TestListNormalizesPage(t *testing.T) {
	tests := []struct {
		name string
		in   shared_domain.Pagination
		want shared_domain.Pagination
	}{
		{name: "sin página ni límite", in: shared_domain.Pagination{}, want: shared_domain.Pagination{Page: 1, Limit: 10}},
		{name: "límite excesivo", in: shared_domain.Pagination{Page: 3, Limit: 500}, want: shared_domain.Pagination{Page: 3, Limit: 100}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()

			page, err := f.app().List(withAdmin(), bondBound_domain.ListFilters{}, tt.in)
			if err != nil {
				t.Fatalf("List: %v", err)
			}

			if len(f.repo.queries) != 1 || f.repo.queries[0].page != tt.want {
				t.Errorf("consultas %+v, se esperaba la página %+v", f.repo.queries, tt.want)
			}
			if page.Page != tt.want.Page || page.Limit != tt.want.Limit || page.Items == nil {
				t.Errorf("página %+v, se esperaba %+v sin ítems", page, tt.want)
			}
		})
	}
}

// Un estado desconocido, un userId fuera de formato, una búsqueda de más de 100 caracteres o un filtro de
// fecha inválido son el error de la validación y no consultan; una búsqueda de 100 caracteres es válida.
// CA-06.2, CA-06.3
func TestListInvalidFilters(t *testing.T) {
	tests := []struct {
		name    string
		filters bondBound_domain.ListFilters
		wantErr error
	}{
		{name: "estado desconocido", filters: bondBound_domain.ListFilters{StatusID: "9"}, wantErr: bondBound_domain.ErrInvalidStatus},
		{name: "userId inválido", filters: bondBound_domain.ListFilters{UserID: "abc"}, wantErr: shared_domain.ErrInvalidID},
		{name: "búsqueda de 101 caracteres", filters: bondBound_domain.ListFilters{Search: strings.Repeat("a", 101)}, wantErr: bondBound_domain.ErrInvalidSearch},
		{name: "búsqueda de 101 letras con tilde", filters: bondBound_domain.ListFilters{Search: strings.Repeat("á", 101)}, wantErr: bondBound_domain.ErrInvalidSearch},
		{
			name:    "startDate posterior a endDate",
			filters: bondBound_domain.ListFilters{Date: bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: "2026-10-14", EndDate: "2026-10-13"}},
			wantErr: bondBound_domain.ErrInvalidDateFilter,
		},
		{name: "tipo de filtro desconocido", filters: bondBound_domain.ListFilters{Date: bondBound_domain.DateFilter{Kind: "aWeek"}}, wantErr: bondBound_domain.ErrInvalidDateFilter},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFakes()

			page, err := f.app().List(withAdmin(), tt.filters, shared_domain.Pagination{Page: 1, Limit: 10})

			if !errors.Is(err, tt.wantErr) || page.Items != nil {
				t.Fatalf("List = %+v, %v, se esperaba %v", page, err, tt.wantErr)
			}
			assertNoQueries(t, f)
		})
	}

	t.Run("búsqueda de 100 caracteres", func(t *testing.T) {
		f := newFakes()

		_, err := f.app().List(withAdmin(), bondBound_domain.ListFilters{Search: strings.Repeat("á", 100)}, shared_domain.Pagination{})

		if err != nil || len(f.repo.queries) != 1 {
			t.Fatalf("List = %v con consultas %+v, se esperaba una consulta", err, f.repo.queries)
		}
	})
}

// Un error del repositorio se devuelve sin cambiarlo. CA-06.1
func TestListRepositoryError(t *testing.T) {
	f := newFakes()
	f.repo.err = errors.New("mongo caído")

	_, err := f.app().List(withAdmin(), bondBound_domain.ListFilters{}, shared_domain.Pagination{})

	if !errors.Is(err, f.repo.err) {
		t.Fatalf("List = %v, se esperaba el error del repositorio", err)
	}
}
