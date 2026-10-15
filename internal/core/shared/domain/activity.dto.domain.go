package shared_domain

// Activity son los datos de una actividad de Blooming. Description es la clave del texto, que se
// formatea con Params al leerla.
type Activity struct {
	Description string
	RelatedID   string
	AgencyID    string
	UserID      string
	Params      map[string]any
}
