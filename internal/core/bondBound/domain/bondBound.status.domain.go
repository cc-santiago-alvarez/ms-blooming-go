package bondBound_domain

// BondBoundStatus es el statusId de un bono comprado.
type BondBoundStatus string

const (
	StatusInitiated BondBoundStatus = "1" // Iniciado
	StatusFinalized BondBoundStatus = "2" // Madurado
	StatusBlocked   BondBoundStatus = "3" // Bloqueado
	StatusUnbound   BondBoundStatus = "4" // Desligado
	StatusSold      BondBoundStatus = "5" // Vendido
	StatusTook      BondBoundStatus = "6" // Canjeado
)

// IsValid indica si el estado es uno de los conocidos.
func (s BondBoundStatus) IsValid() bool {
	return s.Name() != ""
}

// Name devuelve el nombre legacy del estado, o "" si no existe.
func (s BondBoundStatus) Name() string {
	switch s {
	case StatusInitiated:
		return "Iniciado"
	case StatusFinalized:
		return "Madurado"
	case StatusBlocked:
		return "Bloqueado"
	case StatusUnbound:
		return "Desligado"
	case StatusSold:
		return "Vendido"
	case StatusTook:
		return "Canjeado"
	}
	return ""
}
