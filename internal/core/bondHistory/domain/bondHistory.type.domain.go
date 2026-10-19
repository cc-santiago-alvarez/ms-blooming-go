package bondHistory_domain

// HistoryType es el typeId de un registro: un producto o un bono comprado.
type HistoryType string

const (
	HistoryBond      HistoryType = "1"
	HistoryBondBound HistoryType = "2"
)

// IsValid indica si el tipo es uno de los conocidos.
func (t HistoryType) IsValid() bool {
	return t == HistoryBond || t == HistoryBondBound
}

// Name devuelve el nombre del tipo que muestra el historial, o "" si no existe.
func (t HistoryType) Name() string {
	switch t {
	case HistoryBond:
		return "Bono"
	case HistoryBondBound:
		return "Bono Enlazado"
	}
	return ""
}
