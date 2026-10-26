package bondBound_domain_test

import (
	"testing"

	bondBound_domain "github.com/futurapps/ms-blooming-go/internal/core/bondBound/domain"
)

// Encabezado y cierre de la tabla de crons.js del legado.
const (
	legacyMaturityHead = `<table style="width: 100%; text-align: center"><thead><tr><th>Bono</th><th>Tipo</th><th>Nombre</th></tr></thead><tbody>`
	legacyMaturityTail = `</tbody></table>`
)

// BuildMaturityTable arma la tabla del aviso de maduración igual que crons.js del legado: una fila por bono
// en el orden recibido, con la clave resaltada, y los textos escapados. CA-10.6
func TestBuildMaturityTable(t *testing.T) {
	tests := []struct {
		name string
		rows []bondBound_domain.MaturityRow
		want string
	}{
		{
			name: "tres bonos",
			rows: []bondBound_domain.MaturityRow{
				{Key: "BP12-1", Type: "Renta fija", Name: "Bono Plus 12M"},
				{Key: "BP12-2", Type: "Renta fija", Name: "Bono Plus 12M"},
				{Key: "BF6-1", Type: "Renta variable", Name: "Bono Flex 6M"},
			},
			// salida del template de crons.js con los mismos datos
			want: legacyMaturityHead +
				`<tr><td style="color: #4dc3c3">BP12-1</td><td>Renta fija</td><td>Bono Plus 12M</td></tr>` +
				`<tr><td style="color: #4dc3c3">BP12-2</td><td>Renta fija</td><td>Bono Plus 12M</td></tr>` +
				`<tr><td style="color: #4dc3c3">BF6-1</td><td>Renta variable</td><td>Bono Flex 6M</td></tr>` +
				legacyMaturityTail,
		},
		{
			name: "sin bonos",
			want: legacyMaturityHead + legacyMaturityTail,
		},
		{
			name: "textos con HTML",
			rows: []bondBound_domain.MaturityRow{{Key: "BP12-<1>", Type: "Renta & fija", Name: `Bono "Plus"`}},
			want: legacyMaturityHead +
				`<tr><td style="color: #4dc3c3">BP12-&lt;1&gt;</td><td>Renta &amp; fija</td><td>Bono &#34;Plus&#34;</td></tr>` +
				legacyMaturityTail,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := bondBound_domain.BuildMaturityTable(tt.rows); got != tt.want {
				t.Errorf("BuildMaturityTable =\n%s\nse esperaba\n%s", got, tt.want)
			}
		})
	}
}
