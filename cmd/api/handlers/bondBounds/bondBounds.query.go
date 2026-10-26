package bondBounds_handlers

import (
	"net/url"

	handlers_shared "github.com/futurapps/ms-blooming-go/cmd/api/handlers/shared"
	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// queryDateFilter lee el filtro de fecha del query: filter, aDate, startDate, endDate, month y year. Un mes o
// año que no es entero queda en 0 y lo rechaza la validación del caso de uso.
func queryDateFilter(q url.Values) bondBound_domain.DateFilter {
	return bondBound_domain.DateFilter{
		Kind:      bondBound_domain.DateFilterKind(q.Get("filter")),
		ADate:     q.Get("aDate"),
		StartDate: q.Get("startDate"),
		EndDate:   q.Get("endDate"),
		Month:     handlers_shared.QueryInt(q.Get("month")),
		Year:      handlers_shared.QueryInt(q.Get("year")),
	}
}
