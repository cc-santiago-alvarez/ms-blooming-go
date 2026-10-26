package bondBound_domain

import (
	"strings"
	"time"
	"unicode/utf8"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

// DateFilterKind es el tipo de filtro de fecha de los listados y reportes.
type DateFilterKind string

const (
	DateFilterADate        DateFilterKind = "aDate"
	DateFilterBetweenDates DateFilterKind = "betweenDates"
	DateFilterAMonth       DateFilterKind = "aMonth"
	DateFilterAYear        DateFilterKind = "aYear"
)

// maxSearchLength es el largo máximo, en caracteres, del texto de búsqueda.
const maxSearchLength = 100

// DateFilter es el filtro de fecha del query; Kind vacío no filtra. Las fechas son YYYY-MM-DD y se leen
// como días de la zona de la sesión.
type DateFilter struct {
	Kind      DateFilterKind
	ADate     string
	StartDate string
	EndDate   string
	Month     int
	Year      int
}

// DateRange es un rango [From, To) de instantes en UTC.
type DateRange struct {
	From time.Time
	To   time.Time
}

// Normalize recorta el tipo y las fechas.
func (f *DateFilter) Normalize() {
	f.Kind = DateFilterKind(strings.TrimSpace(string(f.Kind)))
	f.ADate = strings.TrimSpace(f.ADate)
	f.StartDate = strings.TrimSpace(f.StartDate)
	f.EndDate = strings.TrimSpace(f.EndDate)
}

// Validation exige un tipo conocido y, según el tipo, fechas YYYY-MM-DD con startDate no posterior a
// endDate, un mes de 1 a 12 o un año de 1 a 9999.
func (f DateFilter) Validation() error {
	_, err := f.Range(0)
	return err
}

// Range devuelve el rango del filtro en la zona utcDiff, o nil si no filtra: el día de aDate, los días de
// startDate a endDate, el mes o el año. Un filtro inválido es ErrInvalidDateFilter.
func (f DateFilter) Range(utcDiff int) (*DateRange, error) {
	zone := time.FixedZone("", utcDiff*3600)
	var from, to time.Time
	switch f.Kind {
	case "":
		return nil, nil
	case DateFilterADate:
		day, err := time.ParseInLocation(time.DateOnly, f.ADate, zone)
		if err != nil {
			return nil, ErrInvalidDateFilter
		}
		from, to = day, day.AddDate(0, 0, 1)
	case DateFilterBetweenDates:
		start, startErr := time.ParseInLocation(time.DateOnly, f.StartDate, zone)
		end, endErr := time.ParseInLocation(time.DateOnly, f.EndDate, zone)
		if startErr != nil || endErr != nil || end.Before(start) {
			return nil, ErrInvalidDateFilter
		}
		from, to = start, end.AddDate(0, 0, 1)
	case DateFilterAMonth:
		if f.Month < 1 || f.Month > 12 || !validYear(f.Year) {
			return nil, ErrInvalidDateFilter
		}
		from = time.Date(f.Year, time.Month(f.Month), 1, 0, 0, 0, 0, zone)
		to = from.AddDate(0, 1, 0)
	case DateFilterAYear:
		if !validYear(f.Year) {
			return nil, ErrInvalidDateFilter
		}
		from = time.Date(f.Year, time.January, 1, 0, 0, 0, 0, zone)
		to = from.AddDate(1, 0, 0)
	default:
		return nil, ErrInvalidDateFilter
	}
	return &DateRange{From: from.UTC(), To: to.UTC()}, nil
}

// validYear indica si el año está entre 1 y 9999.
func validYear(year int) bool {
	return year >= 1 && year <= 9999
}

// ListFilters son los filtros del listado de administración. El estado y el usuario son exactos, Search se
// busca en el perfil, el producto y el tipo, y Date se aplica sobre createdAt; los vacíos no filtran.
type ListFilters struct {
	StatusID BondBoundStatus
	UserID   string
	Search   string
	Date     DateFilter
}

// Normalize recorta el estado, el usuario, la búsqueda y el filtro de fecha.
func (f *ListFilters) Normalize() {
	f.StatusID = BondBoundStatus(strings.TrimSpace(string(f.StatusID)))
	f.UserID = strings.TrimSpace(f.UserID)
	f.Search = strings.TrimSpace(f.Search)
	f.Date.Normalize()
}

// Validation exige un estado conocido, un userId válido, una búsqueda de hasta 100 caracteres y un filtro
// de fecha válido. Los vacíos no se validan.
func (f ListFilters) Validation() error {
	if f.StatusID != "" && !f.StatusID.IsValid() {
		return ErrInvalidStatus
	}
	return validateQueryFilters(f.UserID, f.Search, f.Date)
}

// validateQueryFilters exige un userId válido si viene, una búsqueda de hasta 100 caracteres y un filtro de
// fecha válido.
func validateQueryFilters(userID, search string, date DateFilter) error {
	switch {
	case userID != "" && !shared_domain.IsValidID(userID):
		return shared_domain.ErrInvalidID
	case utf8.RuneCountInString(search) > maxSearchLength:
		return ErrInvalidSearch
	}
	return date.Validation()
}

// MineFilters son los filtros de los bonos del usuario de la sesión; sin estados no filtra por estado.
type MineFilters struct {
	StatusIDs []BondBoundStatus
}

// Normalize recorta los estados y quita los vacíos y los repetidos; si no queda ninguno, deja nil.
func (f *MineFilters) Normalize() {
	var statuses []BondBoundStatus
	seen := make(map[BondBoundStatus]bool, len(f.StatusIDs))
	for _, s := range f.StatusIDs {
		s = BondBoundStatus(strings.TrimSpace(string(s)))
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		statuses = append(statuses, s)
	}
	f.StatusIDs = statuses
}

// Validation exige que cada estado sea uno de los conocidos.
func (f MineFilters) Validation() error {
	for _, s := range f.StatusIDs {
		if !s.IsValid() {
			return ErrInvalidStatus
		}
	}
	return nil
}
