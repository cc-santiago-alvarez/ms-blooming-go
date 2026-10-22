package handlers_shared

import (
	"net/http"
	"strconv"
	"strings"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// QueryPagination lee page y limit del query. Un valor que falta o no es un entero queda en 0, y el caso
// de uso lo normaliza a su default, como hacía el legado.
func QueryPagination(r *http.Request) shared_domain.Pagination {
	q := r.URL.Query()
	return shared_domain.Pagination{
		Page:  queryInt(q.Get("page")),
		Limit: queryInt(q.Get("limit")),
	}
}

// queryInt devuelve el entero de s, o 0 si no lo es.
func queryInt(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}
