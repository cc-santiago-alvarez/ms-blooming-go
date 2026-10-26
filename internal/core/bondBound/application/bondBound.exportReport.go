package bondBound_application

import (
	"context"
	"time"

	bondBound_ports "github.com/futurapps/ms-blooming-go/internal/core/bondBound/ports"
	session_domain "github.com/futurapps/ms-blooming-go/internal/core/session/domain"
	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
)

const (
	// reportTimestampLayout es la hora del nombre de un reporte: YYYY-MM-DD_HHmmss.
	reportTimestampLayout = "2006-01-02_150405"
	// reportExtension es la extensión de los reportes de ms-exporter.
	reportExtension = ".xlsx"
)

// reportPayload es el body que reciben los reportes de bonos en ms-exporter, igual al del legado.
type reportPayload struct {
	Currency   string `json:"currency"`
	BondBounds any    `json:"bondBounds"`
}

// exportReport genera en ms-exporter el Excel report con las filas y la moneda de la agencia, y lo nombra con
// fileName y la hora actual en la zona de la sesión.
func (app *BondBoundApplication) exportReport(ctx context.Context, agencyID, report, fileName string, rows any) (bondBound_ports.ReportFile, error) {
	currency, err := app.Agencies.SymbolCurrency(ctx, agencyID)
	if err != nil {
		return bondBound_ports.ReportFile{}, err
	}

	zone := time.FixedZone("", session_domain.UtcDiff(ctx)*3600)
	stamp := shared_domain.Now().In(zone).Format(reportTimestampLayout)
	name := shared_domain.Format(shared_domain.Text(shared_domain.TextFilenameReport), fileName, stamp) + reportExtension

	content, err := app.Exporter.Excel(ctx, report, reportPayload{Currency: currency, BondBounds: rows})
	if err != nil {
		return bondBound_ports.ReportFile{}, err
	}
	return bondBound_ports.ReportFile{Name: name, Content: content}, nil
}
