package bondBound_domain

import (
	"fmt"
	"html"
	"strings"
)

// MaturityResult es el resumen de una corrida de maduración: bonos madurados, usuarios avisados y bonos que
// fallaron.
type MaturityResult struct {
	Matured int `json:"matured"`
	Users   int `json:"users"`
	Failed  int `json:"failed"`
}

// MaturityRow es un bono madurado en la tabla del aviso: su clave, el nombre del tipo y el del producto.
type MaturityRow struct {
	Key  string
	Type string
	Name string
}

// maturityTableHead abre la tabla del aviso de maduración con sus encabezados.
const maturityTableHead = `<table style="width: 100%; text-align: center"><thead><tr><th>Bono</th><th>Tipo</th><th>Nombre</th></tr></thead><tbody>`

// maturityRowHTML es la fila de un bono, con la clave resaltada.
const maturityRowHTML = `<tr><td style="color: #4dc3c3">%s</td><td>%s</td><td>%s</td></tr>`

// BuildMaturityTable arma la tabla HTML del aviso de maduración con una fila por bono, en el orden recibido.
// Los textos se escapan.
func BuildMaturityTable(rows []MaturityRow) string {
	var b strings.Builder
	b.WriteString(maturityTableHead)
	for _, r := range rows {
		fmt.Fprintf(&b, maturityRowHTML, html.EscapeString(r.Key), html.EscapeString(r.Type), html.EscapeString(r.Name))
	}
	b.WriteString("</tbody></table>")
	return b.String()
}
