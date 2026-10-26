package bondBound_domain_test

import (
	"errors"
	"testing"
	"time"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// aDate es el filtro de un día.
func aDate(day string) bondBound_domain.DateFilter {
	return bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterADate, ADate: day}
}

// betweenDates es el filtro de los días de start a end.
func betweenDates(start, end string) bondBound_domain.DateFilter {
	return bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterBetweenDates, StartDate: start, EndDate: end}
}

// aMonth es el filtro de un mes.
func aMonth(month, year int) bondBound_domain.DateFilter {
	return bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAMonth, Month: month, Year: year}
}

// aYear es el filtro de un año.
func aYear(year int) bondBound_domain.DateFilter {
	return bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterAYear, Year: year}
}

// Range da el rango [desde, hasta) del día, los días, el mes o el año en la zona de la sesión, expresado en
// UTC: un instante de la noche local del último día queda dentro aunque en UTC ya sea el día siguiente.
// CA-06.3
func TestDateFilterRange(t *testing.T) {
	tests := []struct {
		name    string
		filter  bondBound_domain.DateFilter
		utcDiff int
		from    string
		to      string
		in      []string
		out     []string
	}{
		{
			name: "aDate en UTC-5", filter: aDate("2026-10-13"), utcDiff: -5,
			from: "2026-10-13T05:00:00Z", to: "2026-10-14T05:00:00Z",
			in:  []string{"2026-10-13T05:00:00Z", "2026-10-14T04:59:00Z"},
			out: []string{"2026-10-13T04:59:00Z", "2026-10-14T05:00:00Z"},
		},
		{name: "aDate en UTC", filter: aDate("2026-10-13"), utcDiff: 0, from: "2026-10-13T00:00:00Z", to: "2026-10-14T00:00:00Z"},
		{name: "aDate en UTC+14", filter: aDate("2026-10-13"), utcDiff: 14, from: "2026-10-12T10:00:00Z", to: "2026-10-13T10:00:00Z"},
		{
			// el 31-oct a las 22:00 en UTC-5 es el 1-nov en UTC
			name: "aMonth en UTC-5", filter: aMonth(10, 2026), utcDiff: -5,
			from: "2026-10-01T05:00:00Z", to: "2026-11-01T05:00:00Z",
			in:  []string{"2026-11-01T03:00:00Z"},
			out: []string{"2026-10-01T03:00:00Z"},
		},
		{name: "aMonth de diciembre", filter: aMonth(12, 2026), utcDiff: -5, from: "2026-12-01T05:00:00Z", to: "2027-01-01T05:00:00Z"},
		{name: "aMonth de febrero bisiesto", filter: aMonth(2, 2028), utcDiff: -5, from: "2028-02-01T05:00:00Z", to: "2028-03-01T05:00:00Z"},
		{
			name: "aYear", filter: aYear(2026), utcDiff: -5,
			from: "2026-01-01T05:00:00Z", to: "2027-01-01T05:00:00Z",
			in:  []string{"2027-01-01T04:59:00Z"},
			out: []string{"2026-01-01T04:59:00Z"},
		},
		{
			name: "betweenDates", filter: betweenDates("2026-10-01", "2026-10-13"), utcDiff: -5,
			from: "2026-10-01T05:00:00Z", to: "2026-10-14T05:00:00Z",
			in:  []string{"2026-10-14T04:59:00Z"},
			out: []string{"2026-10-14T05:00:00Z"},
		},
		{name: "betweenDates de un día", filter: betweenDates("2026-10-13", "2026-10-13"), utcDiff: -5, from: "2026-10-13T05:00:00Z", to: "2026-10-14T05:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := tt.filter.Range(tt.utcDiff)
			if err != nil || r == nil {
				t.Fatalf("rango %v y error %v, se esperaba un rango", r, err)
			}

			from, to := at(t, tt.from), at(t, tt.to)
			if !r.From.Equal(from) || !r.To.Equal(to) || r.From.Location() != time.UTC || r.To.Location() != time.UTC {
				t.Errorf("rango [%v, %v), se esperaba [%v, %v) en UTC", r.From, r.To, from, to)
			}
			for _, s := range tt.in {
				if x := at(t, s); x.Before(r.From) || !x.Before(r.To) {
					t.Errorf("%s fuera del rango [%v, %v), se esperaba dentro", s, r.From, r.To)
				}
			}
			for _, s := range tt.out {
				if x := at(t, s); !x.Before(r.From) && x.Before(r.To) {
					t.Errorf("%s dentro del rango [%v, %v), se esperaba fuera", s, r.From, r.To)
				}
			}
			if err := tt.filter.Validation(); err != nil {
				t.Errorf("validación %v, se esperaba nil", err)
			}
		})
	}
}

// Sin tipo de filtro no hay rango, aunque vengan fechas. CA-06.3
func TestDateFilterRangeWithoutFilter(t *testing.T) {
	f := bondBound_domain.DateFilter{ADate: "2026-10-13", StartDate: "x", Month: 13}

	r, err := f.Range(-5)

	if r != nil || err != nil {
		t.Errorf("rango %v y error %v, se esperaban nil", r, err)
	}
	if err := f.Validation(); err != nil {
		t.Errorf("validación %v, se esperaba nil", err)
	}
}

// Un tipo desconocido, una fecha mal formada o imposible, un startDate posterior a endDate, o un mes o año
// fuera de rango son ERR_INVALID_DATE_FILTER. CA-06.3
func TestDateFilterRangeInvalid(t *testing.T) {
	tests := []struct {
		name   string
		filter bondBound_domain.DateFilter
	}{
		{name: "startDate posterior a endDate", filter: betweenDates("2026-10-14", "2026-10-13")},
		{name: "betweenDates sin endDate", filter: betweenDates("2026-10-13", "")},
		{name: "tipo desconocido", filter: bondBound_domain.DateFilter{Kind: "aWeek", ADate: "2026-10-13"}},
		{name: "aDate vacía", filter: aDate("")},
		{name: "aDate con otro formato", filter: aDate("13/10/2026")},
		{name: "aDate imposible", filter: aDate("2026-02-30")},
		{name: "mes 0", filter: aMonth(0, 2026)},
		{name: "mes 13", filter: aMonth(13, 2026)},
		{name: "aMonth con año 0", filter: aMonth(10, 0)},
		{name: "aYear con año 10000", filter: aYear(10000)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := tt.filter.Range(-5)

			if r != nil || !errors.Is(err, bondBound_domain.ErrInvalidDateFilter) {
				t.Errorf("rango %v y error %v, se esperaba ERR_INVALID_DATE_FILTER", r, err)
			}
			if err := tt.filter.Validation(); !errors.Is(err, bondBound_domain.ErrInvalidDateFilter) {
				t.Errorf("validación %v, se esperaba ERR_INVALID_DATE_FILTER", err)
			}
		})
	}
}

// Normalize recorta el tipo y las fechas del query antes de validar. CA-06.3
func TestDateFilterNormalize(t *testing.T) {
	f := bondBound_domain.DateFilter{Kind: " aDate ", ADate: " 2026-10-13 ", StartDate: " ", EndDate: "\t"}

	f.Normalize()

	want := bondBound_domain.DateFilter{Kind: bondBound_domain.DateFilterADate, ADate: "2026-10-13"}
	if f != want {
		t.Fatalf("filtro %+v, se esperaba %+v", f, want)
	}
	r, err := f.Range(-5)
	if err != nil || r == nil || !r.From.Equal(at(t, "2026-10-13T05:00:00Z")) || !r.To.Equal(at(t, "2026-10-14T05:00:00Z")) {
		t.Errorf("rango %v y error %v, se esperaba [2026-10-13T05:00Z, 2026-10-14T05:00Z)", r, err)
	}
}

// El listado admin rechaza un betweenDates con startDate posterior a endDate y recorta su filtro de fecha.
// CA-06.3
func TestListFiltersDateValidation(t *testing.T) {
	t.Run("startDate posterior a endDate", func(t *testing.T) {
		f := bondBound_domain.ListFilters{Date: betweenDates("2026-10-14", "2026-10-13")}

		if err := f.Validation(); !errors.Is(err, bondBound_domain.ErrInvalidDateFilter) {
			t.Errorf("validación %v, se esperaba ERR_INVALID_DATE_FILTER", err)
		}
	})

	t.Run("fechas con espacios", func(t *testing.T) {
		f := bondBound_domain.ListFilters{Date: bondBound_domain.DateFilter{
			Kind: " betweenDates ", StartDate: " 2026-10-01 ", EndDate: " 2026-10-13 ",
		}}

		f.Normalize()

		if f.Date != betweenDates("2026-10-01", "2026-10-13") {
			t.Errorf("filtro de fecha %+v, se esperaba recortado", f.Date)
		}
		if err := f.Validation(); err != nil {
			t.Errorf("validación %v, se esperaba nil", err)
		}
	})
}
