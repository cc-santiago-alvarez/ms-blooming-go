# Spec 03 — Proyección y gráficas

| | |
|---|---|
| Estado | Borrador |
| Dominios | `shared_domain` (tipos y funciones puras), `bondObject`, `bondBound` |
| Handlers | `cmd/api/handlers/bondObjects`, `cmd/api/handlers/bondBounds` |
| Depende de | Spec 00, spec 02 (visibilidad de bondObject), spec 04 (repositorio de bondBound) |

## 1. Resumen y alcance

Cálculos de proyección que el front dibuja con amCharts:

- la **simulación diaria** de un bono: inversión, ganancia acumulada y valor con penalidad, día por día hasta el vencimiento;
- las **guías**: la línea del punto de crecimiento (inflexión) y la zona de retiro.

Hay tres usos:
1. el admin previsualiza un bono que todavía no guardó;
2. el inversionista simula un bono del catálogo;
3. el inversionista ve las guías de un bono que ya compró.

Todo el cálculo vive en funciones puras sin I/O (constitución §8).

La cotización de venta anticipada (`preSale`) **no** está aquí; está en la spec 05.

## 2. Origen legacy

| Operación legacy | Archivo:línea | Destino |
|---|---|---|
| `bondObject.getDataChart` | `backend/server/model/bond.object.js:221` | RF-03.1, RF-03.2, RF-03.3, RF-03.4 |
| `bondBound.getChartGuides` | `backend/server/model/bond.bound.js:868` | RF-03.5 |

## 3. Requisitos funcionales

- **RF-03.1** — `SimulateProjection(terms, amount, today, utcDiff) []ChartPoint` (función pura) genera `maturityTimeInDays + 1` puntos, de `i = 0` a `maturityTimeInDays` inclusive, con la fórmula de §5.1.
  - Cada punto lleva `date` (`YYYY-MM-DD` en `utcDiff`) e `inversion`.
  - `ganancia` y `perdida` aparecen desde el día del punto de inflexión.
  - Los valores son `float64` redondeados a 2 decimales. Ningún valor se trunca a entero (corrige O10).
- **RF-03.2** — `BuildObjectGuides(terms, today, utcDiff) []ChartGuide` (función pura) produce las guías del bondObject, siempre relativas a **hoy** (§5.2).
- **RF-03.3** — Vista previa (admin).
  - Recibe los términos de un bondObject sin guardar (`amount`, `maturityTimeInDays`, `maturityRewardsPercentage`, `earlyCashTerm`) y `simulation`.
  - Les aplica los defaults del alta: redondea `amount` a 2 decimales y deja en 100 las penalidades en 0 (spec 02 §5.1). Así la vista previa muestra lo que se guardaría; el legado simulaba los valores sin defaults.
  - Los valida con las reglas de la spec 02 §5.1 que aplican a los términos.
  - Responde `{guides}`, más `data` cuando `simulation=true`.
- **RF-03.4** — Proyección de un bondObject del catálogo (usuario).
  - Carga el bondObject solo si es visible para el usuario con las reglas de RF-02.9 (agencia, activo, privado autorizado). El admin lo ve siempre.
  - Responde `{guides}`, más `data` cuando `simulation=true`.
  - Si no es visible, `ERR_BOND_OBJECT_NOT_FOUND`.
- **RF-03.5** — Guías de un bono comprado (dueño o admin).
  - `BuildBoundGuides(terms, createdAt, utcDiff)` (función pura) usa como base `bondBound.createdAt` y los términos del snapshot `bondBound.earlyCashTerm` y `maturityTimeInDays` (§5.3).
  - Si el bono no existe, `ERR_BOND_BOUND_NOT_FOUND`. Si no es del usuario y el usuario no es admin, `ERR_FORBIDDEN`.

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| POST | `/bond-objects/chart/preview` | admin | RF-03.3 |
| GET | `/bond-objects/{id}/chart?simulation=true` | usuario | RF-03.4 |
| GET | `/bond-bounds/{id}/chart-guides` | dueño o admin | RF-03.5 |

**`POST /bond-objects/chart/preview`**

```json
{ "amount": 1000, "maturityTimeInDays": 365, "maturityRewardsPercentage": 15,
  "earlyCashTerm": { "initialPenaltyPercentage": 100, "inflectionPoint": 90, "inflectionPenaltyPercentage": 75, "initialDayForWithdraw": 30 },
  "simulation": true }
```

→ `200`

```json
{
  "guides": [
    { "date": "2027-01-11", "lineColor": "#CC0000", "lineAlpha": 1, "fillAlpha": 0.2, "fillColor": "#CC0000",
      "dashLength": 10, "inside": true, "labelRotation": 90, "lineThickness": 1.5, "label": "Punto de Crecimiento" },
    { "date": "2026-11-12", "toDate": "2027-10-12", "lineColor": "#4ec74c", "lineAlpha": 1, "fillAlpha": 0.2, "fillColor": "#4ec74c",
      "dashLength": 10, "inside": true, "labelRotation": 0, "position": "top", "label": "Zona de Retiro" }
  ],
  "data": [
    { "date": "2026-10-13", "inversion": 1000 },
    { "date": "2027-01-11", "inversion": 1000, "ganancia": 1000, "perdida": 1000 },
    { "date": "2027-01-12", "inversion": 1000, "ganancia": 1000.55, "perdida": 1000.14 }
  ]
}
```

**`GET /bond-bounds/{id}/chart-guides`** → `200 {"guides": [...]}`. La guía de retiro no tiene `toDate` ni `fillColor`, y usa `labelRotation 90` y `lineThickness 1.5` (paridad).

**Cambio de contrato respecto al legado:**
- `ganancia` y `perdida` eran `string` (`toFixed(2)`) a partir del día siguiente a la inflexión y `number` en el día de inflexión; ahora son siempre `number`.
- `GET /bond-objects/{id}/chart?simulation=true` ahora sí calcula `data`. Antes, sin `preview`, `data` era `[]` (O7).

**Errores:** `ERR_INVALID_ID` (400, id fuera de formato; spec 00, RF-00.19), `ERR_BOND_OBJECT_NOT_FOUND` (404), `ERR_BOND_BOUND_NOT_FOUND` (404), `ERR_FORBIDDEN` (403) y los `ERR_BOND_OBJECT_INVALID_*` de la spec 02 (400) para la vista previa.

## 5. Reglas de negocio

### 5.1 Simulación (paridad con `bond.object.js:221-355`)

```
M  = maturityTimeInDays
R  = maturityRewardsPercentage
P  = earlyCashTerm.inflectionPoint
Pi = earlyCashTerm.initialPenaltyPercentage
Pf = earlyCashTerm.inflectionPenaltyPercentage

si P == 0:  Pf = Pi;  daily = R / M / 100
si no:      daily = R / (M - P) / 100          // P < M garantizado por la validación de la spec 02

current = 0
para i en 0..M:
  punto = {date: fecha(StartOfDay(today, utcDiff) + i días), inversion: amount}
  si i == P:  punto.ganancia = amount; punto.perdida = amount
  si i >  P:
     current += amount * daily
     penalty  = current * (Pf / 100) - current
     punto.ganancia = Round2(current + amount)
     punto.perdida  = Round2(|penalty| + amount)
  data.append(punto)
```

### 5.2 Guías del bondObject (base: hoy)

```
G1 = {date: hoy + P, lineColor "#CC0000", lineAlpha 1, fillAlpha 0.2, fillColor "#CC0000", dashLength 10,
      inside true, labelRotation 90, lineThickness (P > 0 ? 1.5 : 0), label (P > 0 ? "Punto de Crecimiento" : "")}
si initialDayForWithdraw > 0:
G2 = {date: hoy + initialDayForWithdraw, toDate: hoy + (M - 1), lineColor "#4ec74c", lineAlpha 1, fillAlpha 0.2,
      fillColor "#4ec74c", dashLength 10, inside true, labelRotation 0, position "top", label "Zona de Retiro"}
```

### 5.3 Guías del bondBound (base: `createdAt`)

```
G1 = igual que 5.2 con date = createdAt + P
si initialDayForWithdraw > 0:
G2 = {date: createdAt + initialDayForWithdraw, lineColor "#4ec74c", lineAlpha 1, fillAlpha 0.2, dashLength 10,
      inside true, labelRotation 90, label "Zona de Retiro", lineThickness 1.5}
```

Las fechas se formatean en la zona `utcDiff` de la sesión. El legado usaba la zona del servidor; ver O8.

## 6. Modelo de datos

```go
// shared_domain
type EarlyCashTerm struct { /* ver spec 02 §6 */ }
type ChartPoint struct {
    Date      string   `json:"date"`
    Inversion float64  `json:"inversion"`
    Ganancia  *float64 `json:"ganancia,omitempty"`
    Perdida   *float64 `json:"perdida,omitempty"`
}
type ChartGuide struct {
    Date          string   `json:"date"`
    ToDate        string   `json:"toDate,omitempty"`
    LineColor     string   `json:"lineColor"`
    LineAlpha     float64  `json:"lineAlpha"`
    FillAlpha     float64  `json:"fillAlpha"`
    FillColor     string   `json:"fillColor,omitempty"`
    DashLength    int      `json:"dashLength"`
    Inside        bool     `json:"inside"`
    LabelRotation int      `json:"labelRotation"`
    LineThickness *float64 `json:"lineThickness,omitempty"`
    Position      string   `json:"position,omitempty"`
    Label         string   `json:"label"`
}
```

Esta spec no persiste nada.

## 7. Ports y dependencias

- `BondObjectApplicationPorts.ChartPreview` y `ChartByID` (spec 02). Para la visibilidad, `ChartByID` reutiliza el filtro de RF-02.9.
- `BondBoundApplicationPorts.ChartGuides` (spec 04/06). Usa `FindOneByID` del repositorio de `bondBound` (BLM-0077).
- `shared.StartOfDay` y `shared.Round2` (spec 00).

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| O10 | `getDataChart` y `getChartGuides` aplican `parseInt` sobre `amount` y porcentajes: un bono de 1000.50 con 7.5% se simula como 1000 con 7%. | Corregir | Cálculos de dinero. | BLM-0029 |
| O7 | `getDataChart` con `simulation` pero sin `preview` devuelve `data: []`. | Corregir | La proyección por id es útil para el inversionista; se calcula siempre que `simulation=true`. | BLM-0060 |
| O8 | `getChartGuides` formatea las fechas con la zona del servidor; `getDataChart` usa `session.utcDiff`. | Corregir | Las fechas se ven igual en las dos gráficas. | BLM-0030 |
| B17 | `getChartGuides` no valida que el bono exista (crashea) ni que sea del usuario. | Corregir | Llamada que no responde, y exposición de datos ajenos. | BLM-0078 |

## 9. Criterios de aceptación

- **CA-03.1** (RF-03.1) — Con `amount 1000`, `M 10`, `R 10`, `P 0`, `Pi 100` y `Pf 100`, la simulación tiene 11 puntos: el día 0 tiene `ganancia 1000` y `perdida 1000`, el día 1 `ganancia 1010` y `perdida 1000`, y el día 10 `ganancia 1100` y `perdida 1000`.
- **CA-03.2** (RF-03.1) — Con `amount 1000`, `M 10`, `R 10`, `P 5` y `Pf 50`: los días 0–4 no tienen `ganancia` ni `perdida`; el día 5 da `1000/1000`, el día 6 `1020/1010` y el día 10 `1100/1050`.
- **CA-03.3** (RF-03.1) — Con `amount 1000.50` y `R 7.5`, ningún valor se calcula con 1000 ni con 7 (comparado contra la fórmula en float).
- **CA-03.4** (RF-03.2) — Con `P 0` e `initialDayForWithdraw 0` hay una sola guía, con `lineThickness 0` y `label ""`.
- **CA-03.5** (RF-03.2) — Con hoy `2026-10-13`, `utcDiff -5`, `initialDayForWithdraw 30` y `M 365`, la guía de retiro tiene `date "2026-11-12"` y `toDate "2027-10-12"`.
- **CA-03.6** (RF-03.3) — Sin `adminBlooming`, el preview responde `403`. Con `inflectionPoint >= maturityTimeInDays` responde `400`. Con `simulation: false` la respuesta no tiene la clave `data`.
- **CA-03.7** (RF-03.4) — La proyección de un bono privado que no autoriza al usuario responde `404 ERR_BOND_OBJECT_NOT_FOUND`; un admin sí la recibe.
- **CA-03.8** (RF-03.5) — Las guías de un bono de otro usuario responden `403` para un no admin. Para el dueño, con `createdAt 2026-01-01T15:00Z`, `P 30` e `initialDayForWithdraw 10`, `G1.date` es `"2026-01-31"`, `G2.date` es `"2026-01-11"`, y `G2` no tiene `toDate`.
- **CA-03.9** (RF-03.4) — `GET /bond-objects/{id}/chart?simulation=true` sobre un bondObject legacy (`_id` de 17 caracteres) devuelve la misma proyección que sobre uno nuevo con los mismos términos.
- **CA-03.10** (RF-03.3) — Con `amount 1000.004`, `M 10`, `R 100`, `P 0` y las dos penalidades en `0`, el día 10 de la vista previa da `ganancia 2000` y `perdida 1000`: se simuló con `amount 1000` y penalidades de 100. Sin el redondeo la ganancia daría `2000.01`, y sin el default la pérdida daría `2000`.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0028 | `shared_domain`: `EarlyCashTerm` (decodifica numéricos legacy con `FlexNumber`), `FlexNumber`, `ChartPoint` y `ChartGuide` | impl | F2 | RF-03.1, RF-03.2, RF-03.5 | BLM-0009 | `internal/core/shared/domain/shared.earlyCashTerm.go`, `shared.flexNumber.go`, `shared.chart.go` |
| BLM-0029 | Función pura `SimulateProjection` | impl | F2 | RF-03.1 | BLM-0028 | `internal/core/shared/domain/shared.projection.go` |
| BLM-0030 | Funciones puras `BuildObjectGuides` y `BuildBoundGuides` | impl | F2 | RF-03.2, RF-03.5 | BLM-0028 | `internal/core/shared/domain/shared.chartGuides.go` |
| BLM-0031 | Tests de tabla de `SimulateProjection` y de las guías, con casos calculados con el código legacy | test | F2 | CA-03.1, CA-03.2, CA-03.3, CA-03.4, CA-03.5 | BLM-0029, BLM-0030 | `internal/core/shared/domain/shared.projection_test.go`, `shared.chartGuides_test.go`, `testdata/legacy-projection.json`, `legacy-guides.json` |
| BLM-0060 | Casos de uso `ChartPreview` y `ChartByID` en bondObject, con visibilidad de RF-02.9 | impl | F3 | RF-03.3, RF-03.4 | BLM-0029, BLM-0030, BLM-0055 | `internal/core/bondObject/application/bondObject.chartPreview.go`, `bondObject.chartById.go`, `internal/core/bondObject/ports/bondObject.ports.go` (`ChartPreview` y `ChartByID`), `internal/core/bondObject/domain/bondObject.dto.domain.go` (`ChartPreviewParams`, `BondObjectChart` y validación de términos compartida con el alta), `bondObject.domain.go` (`ProjectionTerms`) |
| BLM-0061 | Handlers y rutas de bondObject: `POST /bond-objects/chart/preview` y `GET /bond-objects/{id}/chart` | impl | F3 | RF-03.3, RF-03.4 | BLM-0060, BLM-0056, BLM-0026 | `cmd/api/handlers/bondObjects/bondObjects.chartPreview.post.go`, `bondObjects.chart.get.go`, `bondObjects.handler.go` (registro de las rutas) |
| BLM-0062 | Tests de casos de uso y handlers de la proyección de bondObject (permisos, visibilidad, validación) | test | F3 | CA-03.6, CA-03.7, CA-03.9, CA-03.10 | BLM-0061 | `internal/core/bondObject/application/bondObject.chart*_test.go`, `bondObject.fakes_test.go` (`FindVisible` en memoria), `cmd/api/handlers/bondObjects/bondObjects.chart*_test.go`, `bondObjects.testServer_test.go` (`ChartPreview` y `ChartByID` en `fakeApp`), `internal/core/bondObject/infrastructure/mongocc/bondObject.chart_integration_test.go` |
| BLM-0078 | Caso de uso `ChartGuides` en bondBound, con control de dueño o admin | impl | F5 | RF-03.5 | BLM-0030, BLM-0077, BLM-0007 | `internal/core/bondBound/application/bondBound.chartGuides.go` |
| BLM-0085 | Handler y ruta `GET /bond-bounds/{id}/chart-guides` | impl | F5 | RF-03.5 | BLM-0078, BLM-0084, BLM-0026 | `cmd/api/handlers/bondBounds/bondBounds.chartGuides.get.go` |
| BLM-0086 | Tests del caso de uso y del handler de guías de bondBound (dueño, admin, 404) | test | F5 | CA-03.8 | BLM-0085 | `internal/core/bondBound/application/bondBound.chartGuides_test.go` |

BLM-0028, BLM-0029, BLM-0030 y BLM-0031 van en la fase F2. BLM-0060, BLM-0061 y BLM-0062 van en la F3 (catálogo). BLM-0078, BLM-0085 y BLM-0086 van en la F5, cuando ya existe el repositorio de `bondBound` (BLM-0077). Ver `tasks.md`.

## 11. Fuera de alcance y preguntas abiertas

- La variable `totalProfit` y el cálculo de `maximum` del legado están muertos o comentados y no se portan.
- **Pregunta:** ¿el front usa `labelRotation` y `lineThickness` tal cual, o se pueden simplificar las guías? Por ahora se mantienen idénticas.
