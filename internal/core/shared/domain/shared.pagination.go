package shared_domain

const (
	defaultPage  = 1
	defaultLimit = 10
	maxLimit     = 100
)

// Pagination es la página pedida en un listado.
type Pagination struct {
	Page  int
	Limit int
}

// Normalize deja Page en 1 o más y Limit entre 1 y 100 (10 por defecto).
func (p *Pagination) Normalize() {
	if p.Page < 1 {
		p.Page = defaultPage
	}
	if p.Limit < 1 {
		p.Limit = defaultLimit
	}
	if p.Limit > maxLimit {
		p.Limit = maxLimit
	}
}

// Skip devuelve cuántos documentos se saltan para llegar a la página.
func (p Pagination) Skip() int64 {
	return int64(p.Page-1) * int64(p.Limit)
}

// Paginated es la respuesta de un listado paginado.
type Paginated[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"totalPages"`
}

// NewPaginated arma la respuesta paginada; items nunca queda en nil.
func NewPaginated[T any](items []T, total int64, p Pagination) Paginated[T] {
	if items == nil {
		items = []T{}
	}
	totalPages := 0
	if total > 0 && p.Limit > 0 {
		totalPages = int((total + int64(p.Limit) - 1) / int64(p.Limit))
	}
	return Paginated[T]{
		Items:      items,
		Total:      total,
		Page:       p.Page,
		Limit:      p.Limit,
		TotalPages: totalPages,
	}
}
