//go:build load

package load_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	shared_domain "github.com/futurapps/ms-blooming-go/internal/core/shared/domain"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Volumen de la siembra del listado.
const (
	seededBounds   = 100_000
	seededProfiles = 1000
	seededProducts = 10
	// seedBatch es la cantidad de documentos de cada InsertMany.
	seedBatch = 10_000
)

// Medición del listado: requests de calentamiento y medidos por escenario, y clientes concurrentes.
const (
	listWarmup   = 3
	listRequests = 40
	listClients  = 4
	// maxListP95 es el p95 máximo de cada escenario.
	maxListP95 = time.Second
)

// statuses son los estados de los bonos sembrados, en rueda.
var statuses = []string{"1", "2", "3", "4", "5", "6"}

// newestBound es el createdAt del bono más reciente de la siembra; los demás van un minuto antes cada uno.
var newestBound = time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)

// listScenario es una consulta del listado y lo que debe responder.
type listScenario struct {
	name  string
	query string
	total int
	items int
}

// Con 100 000 bonos de la agencia, el p95 de cada escenario del listado de administración (primera página,
// página profunda, búsqueda y filtro por estado), medido desde 4 clientes concurrentes, no pasa de 1 s, y
// cada respuesta trae el total y la cantidad de items esperados. RF-06.1
func TestAdminListP95(t *testing.T) {
	setup(t)
	seedCatalog(t)
	searched, byStatus := seedBounds(t)

	scenarios := []listScenario{
		{name: "primera página", query: "", total: seededBounds, items: 10},
		{name: "página profunda", query: fmt.Sprintf("?page=%d&limit=100", seededBounds/100), total: seededBounds, items: 100},
		{name: "búsqueda", query: "?search=" + url.QueryEscape(searched), total: seededBounds / seededProfiles, items: 10},
		{name: "por estado", query: "?statusId=" + statuses[0], total: byStatus, items: 10},
	}
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			for range listWarmup {
				if _, err := sc.list(); err != nil {
					t.Fatal(err)
				}
			}

			took := make([]time.Duration, listRequests)
			errs := make([]error, listRequests)
			jobs := make(chan int, listRequests)
			for i := range listRequests {
				jobs <- i
			}
			close(jobs)
			var wg sync.WaitGroup
			for range listClients {
				wg.Go(func() {
					for i := range jobs {
						took[i], errs[i] = sc.list()
					}
				})
			}
			wg.Wait()

			for _, err := range errs {
				if err != nil {
					t.Fatal(err)
				}
			}
			logLatencies(t, sc.name, took)
			if p95 := percentile(took, 95); p95 > maxListP95 {
				t.Errorf("p95 de %s = %v, se esperaba como máximo %v", sc.name, p95.Round(time.Millisecond), maxListP95)
			}
		})
	}
}

// list pide el listado del escenario como el admin y verifica la página; devuelve lo que tardó.
func (sc listScenario) list() (time.Duration, error) {
	res, err := send(http.MethodGet, "/bond-bounds"+sc.query, adminToken, nil)
	if err != nil {
		return 0, err
	}
	var page struct {
		Items []json.RawMessage `json:"items"`
		Total int               `json:"total"`
	}
	if err := res.decode(http.StatusOK, &page); err != nil {
		return 0, err
	}
	if page.Total != sc.total || len(page.Items) != sc.items {
		return 0, fmt.Errorf("%s: %d items de %d, se esperaban %d de %d",
			sc.name, len(page.Items), page.Total, sc.items, sc.total)
	}
	return res.took, nil
}

// seedBounds guarda directo en Mongo los productos, los perfiles y los bonos del listado, todos de la
// agencia, con _id UUID y Meteor alternados. Devuelve el nombre de un dueño, para buscarlo, y cuántos bonos
// quedaron en el primer estado.
func seedBounds(t *testing.T) (searched string, byStatus int) {
	t.Helper()
	started := time.Now()
	insert(t, "bondType", typeDoc(otherTypeID, "Renta variable"))

	types := []string{typeID, otherTypeID}
	products := make([]bson.M, seededProducts)
	docs := make([]any, 0, seededProducts)
	for i := range products {
		products[i] = objectDoc(mixedID("Ob", i), types[i%len(types)], fmt.Sprintf("Producto de carga %02d", i), fmt.Sprintf("LP%02d", i))
		docs = append(docs, products[i])
	}
	insertMany(t, "bondObject", docs)

	profiles := make([]string, seededProfiles)
	docs = make([]any, 0, seededProfiles)
	for i := range profiles {
		profiles[i] = mixedID("Pf", i)
		docs = append(docs, profileDoc(profiles[i], fmt.Sprintf("CC-%010d", i), profileName(i), "Carga",
			fmt.Sprintf("inversor%04d@futurapps.test", i+1)))
	}
	insertMany(t, "profile", docs)

	docs = make([]any, 0, seedBatch)
	for i := range seededBounds {
		product := products[i%seededProducts]
		// los productos pares son del primer tipo y los impares del segundo: i/2 no se repite en un tipo
		seq := i/2 + 1
		createdAt := newestBound.Add(-time.Duration(i) * time.Minute)
		status := statuses[i%len(statuses)]
		if status == statuses[0] {
			byStatus++
		}
		docs = append(docs, bson.M{
			"_id":                       mixedID("Bb", i),
			"bondObjectId":              product["_id"],
			"bondTypeId":                product["bondTypeId"],
			"key":                       fmt.Sprintf("%s-%d", product["key"], seq),
			"keySeq":                    seq,
			"token":                     fmt.Sprintf("%06d", i),
			"userId":                    profiles[i%seededProfiles],
			"agencyId":                  agencyID,
			"statusId":                  status,
			"amount":                    price,
			"maturityTimeInDays":        maturityDays,
			"maturityRewardsPercentage": rewards,
			"earlyCashTerm":             earlyCashTerm,
			"initialDate":               createdAt,
			"maturityDate":              createdAt.AddDate(0, 0, maturityDays),
			"createdAt":                 createdAt,
			"updatedAt":                 createdAt,
			"isRemove":                  false,
		})
		if len(docs) == seedBatch {
			insertMany(t, "bondBound", docs)
			docs = docs[:0]
		}
	}
	insertMany(t, "bondBound", docs)

	t.Logf("siembra: %d bonos, %d perfiles y %d productos en %v", seededBounds, seededProfiles, seededProducts,
		time.Since(started).Round(time.Millisecond))
	return profileName(41), byStatus
}

// mixedID es un UUID v4 para los i pares y, para los impares, un id Meteor de 17 caracteres con prefix e i.
func mixedID(prefix string, i int) string {
	if i%2 == 0 {
		return shared_domain.NewID()
	}
	return fmt.Sprintf("%s%0*d", prefix, 17-len(prefix), i)
}

// profileName es el nombre del perfil i: único entre los sembrados.
func profileName(i int) string {
	return fmt.Sprintf("Inversor %04d", i+1)
}

// insertMany guarda docs en collection en una sola operación, sin orden.
func insertMany(t *testing.T, collection string, docs []any) {
	t.Helper()
	if len(docs) == 0 {
		return
	}
	if _, err := testDB.GetCollection(collection).InsertMany(context.Background(), docs, options.InsertMany().SetOrdered(false)); err != nil {
		t.Fatalf("no se pudieron guardar %d documentos en %s: %v", len(docs), collection, err)
	}
}
