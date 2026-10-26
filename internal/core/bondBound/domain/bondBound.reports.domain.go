package bondBound_domain

import "strings"

// ReportAllStatuses es el statusId de los reportes que incluye todos los estados.
const ReportAllStatuses BondBoundStatus = "0"

// ReportFilters son los filtros de los reportes de bonos comprados. Tras Normalize, StatusID es el estado por
// el que se filtra, o ReportAllStatuses para todos. El usuario es exacto, Search se busca en el dueño, el
// producto y el tipo, y Date se aplica sobre createdAt; los vacíos no filtran.
type ReportFilters struct {
	StatusID BondBoundStatus
	UserID   string
	Search   string
	Date     DateFilter
}

// Normalize recorta los filtros y deja el estado iniciado si no viene.
func (f *ReportFilters) Normalize() {
	f.StatusID = BondBoundStatus(strings.TrimSpace(string(f.StatusID)))
	if f.StatusID == "" {
		f.StatusID = StatusInitiated
	}
	f.UserID = strings.TrimSpace(f.UserID)
	f.Search = strings.TrimSpace(f.Search)
	f.Date.Normalize()
}

// AllStatuses indica si el reporte incluye todos los estados.
func (f ReportFilters) AllStatuses() bool {
	return f.StatusID == ReportAllStatuses
}

// Validation exige un estado conocido o ReportAllStatuses, un userId válido, una búsqueda de hasta 100
// caracteres y un filtro de fecha válido.
func (f ReportFilters) Validation() error {
	if !f.AllStatuses() && !f.StatusID.IsValid() {
		return ErrInvalidStatus
	}
	return validateQueryFilters(f.UserID, f.Search, f.Date)
}

// KpiFilters son los filtros del KPI de bonos por estado. Tras Normalize, StatusID es un estado conocido o
// vacío, que no filtra. El usuario es exacto, Search se busca en el dueño y el producto, y Date se aplica
// sobre createdAt; los vacíos no filtran.
type KpiFilters struct {
	StatusID BondBoundStatus
	UserID   string
	Search   string
	Date     DateFilter
}

// Normalize recorta los filtros y vacía el estado si no es uno de los conocidos.
func (f *KpiFilters) Normalize() {
	f.StatusID = BondBoundStatus(strings.TrimSpace(string(f.StatusID)))
	if !f.StatusID.IsValid() {
		f.StatusID = ""
	}
	f.UserID = strings.TrimSpace(f.UserID)
	f.Search = strings.TrimSpace(f.Search)
	f.Date.Normalize()
}

// Validation exige un userId válido, una búsqueda de hasta 100 caracteres y un filtro de fecha válido; el
// estado no se valida.
func (f KpiFilters) Validation() error {
	return validateQueryFilters(f.UserID, f.Search, f.Date)
}
