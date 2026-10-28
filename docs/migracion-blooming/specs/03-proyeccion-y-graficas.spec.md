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
- la **evolución de un bono comprado**: lo acumulado hasta hoy, con y sin penalidad, y lo proyectado hasta el vencimiento;
- las **guías**: la línea del punto de crecimiento (inflexión) y la zona de retiro.

Hay cuatro usos:
1. el admin previsualiza un bono que todavía no guardó;
2. el inversionista simula un bono del catálogo;
3. el inversionista ve las guías de un bono que ya compró;
4. el inversionista ve la evolución de un bono que ya compró, con sus guías en la misma respuesta.

Además de los métodos de Meteor, esta spec reemplaza los dos feeds de ms-feed (`GET /bond/:id/feed` y `GET /bound/:id/feed`), que el front llama directo desde el navegador.

Todo el cálculo vive en funciones puras sin I/O (constitución §8).

La cotización de venta anticipada (`preSale`) **no** está aquí; está en la spec 05.

## 2. Origen legacy

| Operación legacy | Archivo:línea | Destino |
|---|---|---|
| `bondObject.getDataChart` | `backend/server/model/bond.object.js:221` | RF-03.1, RF-03.2, RF-03.3, RF-03.4 |
| `bondBound.getChartGuides` | `backend/server/model/bond.bound.js:868` | RF-03.5 |
| `bondController.feed` de ms-feed (`GET /bond/:id/feed`) | `ms-feed/src/components/bond/index.js:5-63` | RF-03.4 (misma fórmula que §5.1) |
| `boundController.feed` de ms-feed (`GET /bound/:id/feed`) | `ms-feed/src/components/bound/index.js:14-122` | RF-03.6, RF-03.7 |

El front pedía a ms-feed la `data` de las dos gráficas: `getDataChart` solo la calcula con `preview` (O7) y `getChartGuides` devuelve solo guías.

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
  - Carga el bondObject solo si es visible para el usuario con las reglas de RF-02.9 (agencia, activo, privado autorizado). Un admin lo ve aunque esté inactivo o sea privado, dentro del alcance de agencia de la sesión (spec 02): si su agencia no es la principal, el `agencyId` del bondObject debe contenerla (corrige O12).
  - Responde `{guides}`, más `data` cuando `simulation=true`.
  - Si no es visible, `ERR_BOND_OBJECT_NOT_FOUND`.
- **RF-03.5** — Guías de un bono comprado (dueño o admin).
  - `BuildBoundGuides(terms, createdAt, utcDiff)` (función pura) usa como base `bondBound.createdAt` y los términos del snapshot `bondBound.earlyCashTerm` y `maturityTimeInDays` (§5.3).
  - Si el bono no existe o está fuera del alcance de agencia de la sesión (`AgencyScope`), `ERR_BOND_BOUND_NOT_FOUND`, también para el dueño. Si no es del usuario y el usuario no es admin, `ERR_FORBIDDEN`. El alcance se valida antes que el dueño, como en el detalle (spec 06, RF-06.7; corrige B38).
- **RF-03.6** — `BoundProjection(terms, amount, initialDate, now, utcDiff) []BoundChartPoint` (función pura) genera la evolución de un bono comprado con la fórmula de §5.4: `maturityTimeInDays + 1` puntos, de `i = 0` a `maturityTimeInDays` inclusive, desde `StartOfDay(initialDate, utcDiff)`.
  - Los días hasta hoy llevan `inversion` (lo acumulado) y `perdida` (lo acumulado con la penalidad). Los días posteriores a la inflexión que todavía no llegaron llevan `ganancia` (lo proyectado). Los días hasta la inflexión llevan además `ganancia` igual a `amount`.
  - El punto de hoy lleva `atMoment`, `perdidaAtMoment` y `bulletClass: "lastBullet"`; el último lleva `gananciaFinal`.
  - Los valores son `float64` redondeados a 2 decimales, sin truncar porcentajes (corrige O10). Las fechas y los días transcurridos se calculan en `utcDiff` (corrige O8).
- **RF-03.7** — Gráfica de un bono comprado (dueño o admin).
  - Responde `{guides, data}`: las guías de RF-03.5 y la evolución de RF-03.6 a la fecha actual, en el `utcDiff` de la sesión.
  - Mismo control que RF-03.5: si el bono no existe, está borrado o fuera del alcance de agencia, `ERR_BOND_BOUND_NOT_FOUND`; si no es del usuario y el usuario no es admin, `ERR_FORBIDDEN` (corrige B35 y B38).

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| POST | `/bond-objects/chart/preview` | admin | RF-03.3 |
| GET | `/bond-objects/{id}/chart?simulation=true` | usuario | RF-03.4 |
| GET | `/bond-bounds/{id}/chart-guides` | dueño o admin | RF-03.5 |
| GET | `/bond-bounds/{id}/chart` | dueño o admin | RF-03.7 |

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

**`GET /bond-bounds/{id}/chart`** → `200`. Con los datos de CA-03.11 (se muestran 5 de los 11 puntos):

```json
{
  "guides": [
    { "date": "2026-01-06", "lineColor": "#CC0000", "label": "Punto de Crecimiento", "…": "…" }
  ],
  "data": [
    { "date": "2026-01-01", "inversion": 1000, "perdida": 1000, "ganancia": 1000 },
    { "date": "2026-01-07", "inversion": 1020, "perdida": 1010 },
    { "date": "2026-01-08", "inversion": 1040, "perdida": 1020, "atMoment": 1040, "perdidaAtMoment": 1020, "bulletClass": "lastBullet" },
    { "date": "2026-01-09", "ganancia": 1060 },
    { "date": "2026-01-11", "ganancia": 1100, "gananciaFinal": 1100 }
  ]
}
```

Las guías son las mismas que devuelve `/chart-guides`.

**Cambio de contrato respecto al legado:**
- `ganancia` y `perdida` eran `string` (`toFixed(2)`) a partir del día siguiente a la inflexión y `number` en el día de inflexión; ahora son siempre `number`.
- `GET /bond-objects/{id}/chart?simulation=true` ahora sí calcula `data`. Antes, sin `preview`, `data` era `[]` (O7).

**Cambio de contrato respecto a ms-feed:**
- `GET /bound/:id/feed` → `GET /bond-bounds/{id}/chart`:
  - la respuesta es un objeto `{guides, data}` y no un arreglo; las guías, que el front pedía a `bondBound.getChartGuides`, vienen en la misma respuesta;
  - todos los valores son `number` con 2 decimales. En ms-feed `atMoment`, `perdidaAtMoment`, `gananciaFinal` y, después de la inflexión, `ganancia` y `perdida` eran `string`, e `inversion` venía sin redondear;
  - las fechas están en el `utcDiff` de la sesión; ms-feed usaba la zona del servidor;
  - exige sesión: un bono ajeno responde `403` y uno inexistente o borrado, `404`. ms-feed respondía a cualquiera, aunque el bono estuviera borrado, y con `500 BOND_BOUND_FEED_ERROR:TRY_AGAIN_LATER` si no existía.
- `GET /bond/:id/feed` → `GET /bond-objects/{id}/chart?simulation=true` (RF-03.4): la `data` viene junto con las guías, con números, en `utcDiff` y con la visibilidad de RF-02.9.

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

### 5.4 Evolución del bono comprado (paridad con `ms-feed/src/components/bound/index.js:14-122`)

```
M  = maturityTimeInDays
R  = maturityRewardsPercentage
P  = earlyCashTerm.inflectionPoint
Pf = earlyCashTerm.inflectionPenaltyPercentage

daily   = R / (M - P) / 100          // con P == 0 no se reemplaza Pf por Pi, a diferencia de §5.1 (igual que QuoteSale, spec 05)
start   = StartOfDay(initialDate, utcDiff)
elapsed = DaysBetween(initialDate, now, utcDiff)    // días transcurridos; puede pasar de M
current = 0
para i en 0..M:
  punto = {date: fecha(start + i días)}
  si i > P:  current += amount * daily
  valor = Round2(current + amount)
  penal = Round2(|current * (Pf / 100) - current| + amount)
  si i <= P:                punto.ganancia = Round2(amount)
  si i <= elapsed:          punto.inversion = valor;  punto.perdida = penal
  si i >  P y i > elapsed:  punto.ganancia = valor
  si i == elapsed:          punto.atMoment = valor;  punto.perdidaAtMoment = penal;  punto.bulletClass = "lastBullet"
  si i == M:                punto.gananciaFinal = valor
  data.append(punto)
```

- Un bono vencido (`elapsed > M`) no tiene punto de hoy: ningún punto lleva `atMoment` ni `bulletClass` (paridad).
- Con `P >= M` nunca se suma interés: `daily` no se usa y la serie queda plana en `amount`.
- La base de `data` es `initialDate`, como en `QuoteSale`; la de las guías (§5.3) es `createdAt`. En la compra los dos valen lo mismo (spec 04), así que la serie y las guías quedan alineadas.
- `current` se acumula día por día igual que en JavaScript (sin FMA), como en `SimulateProjection`.

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
type BoundChartPoint struct {
    Date            string   `json:"date"`
    Inversion       *float64 `json:"inversion,omitempty"`
    Ganancia        *float64 `json:"ganancia,omitempty"`
    Perdida         *float64 `json:"perdida,omitempty"`
    AtMoment        *float64 `json:"atMoment,omitempty"`
    PerdidaAtMoment *float64 `json:"perdidaAtMoment,omitempty"`
    BulletClass     string   `json:"bulletClass,omitempty"`
    GananciaFinal   *float64 `json:"gananciaFinal,omitempty"`
}

// bondBound_domain
type BondBoundChart struct {
    Guides []shared_domain.ChartGuide      `json:"guides"`
    Data   []shared_domain.BoundChartPoint `json:"data,omitempty"` // vacío en /chart-guides
}
```

Esta spec no persiste nada.

## 7. Ports y dependencias

- `BondObjectApplicationPorts.ChartPreview` y `ChartByID` (spec 02). Para la visibilidad, `ChartByID` reutiliza el filtro de RF-02.9 y, para el admin, el alcance de agencia de la spec 02.
- `BondBoundApplicationPorts.ChartGuides` (spec 04/06). Usa `FindOneByID` del repositorio de `bondBound` (BLM-0077).
- `BondBoundApplicationPorts.Chart` (RF-03.7). Reutiliza el control de alcance de agencia y de dueño o admin de `ChartGuides` (`findOwnedOrAdmin`).
- `shared.StartOfDay`, `shared.DaysBetween` y `shared.Round2` (spec 00).

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| O10 | `getDataChart` y `getChartGuides` aplican `parseInt` sobre `amount` y porcentajes: un bono de 1000.50 con 7.5% se simula como 1000 con 7%. Los feeds de ms-feed hacen lo mismo: `/bond/:id/feed` con el monto y los porcentajes, y `/bound/:id/feed` con los porcentajes y el punto de inflexión. | Corregir | Cálculos de dinero. | BLM-0029, BLM-0143 |
| O7 | `getDataChart` con `simulation` pero sin `preview` devuelve `data: []`. | Corregir | La proyección por id es útil para el inversionista; se calcula siempre que `simulation=true`. | BLM-0060 |
| O8 | `getChartGuides` formatea las fechas con la zona del servidor; `getDataChart` usa `session.utcDiff`. Los feeds de ms-feed también usan la zona del servidor, para las fechas y para los días transcurridos. | Corregir | Las fechas se ven igual en las dos gráficas. | BLM-0030, BLM-0143 |
| B17 | `getChartGuides` no valida que el bono exista (crashea) ni que sea del usuario. | Corregir | Llamada que no responde, y exposición de datos ajenos. | BLM-0078 |
| B35 | `GET /bound/:id/feed` de ms-feed no tiene autenticación (`oAuth` es un stub) y tiene CORS `*`: con el `_id` cualquiera lee el monto y la evolución de un bono ajeno, aunque esté borrado. Si el bono no existe, responde `500`. | Corregir | Exposición de datos ajenos. | BLM-0145 |

## 9. Criterios de aceptación

- **CA-03.1** (RF-03.1) — Con `amount 1000`, `M 10`, `R 10`, `P 0`, `Pi 100` y `Pf 100`, la simulación tiene 11 puntos: el día 0 tiene `ganancia 1000` y `perdida 1000`, el día 1 `ganancia 1010` y `perdida 1000`, y el día 10 `ganancia 1100` y `perdida 1000`.
- **CA-03.2** (RF-03.1) — Con `amount 1000`, `M 10`, `R 10`, `P 5` y `Pf 50`: los días 0–4 no tienen `ganancia` ni `perdida`; el día 5 da `1000/1000`, el día 6 `1020/1010` y el día 10 `1100/1050`.
- **CA-03.3** (RF-03.1) — Con `amount 1000.50` y `R 7.5`, ningún valor se calcula con 1000 ni con 7 (comparado contra la fórmula en float).
- **CA-03.4** (RF-03.2) — Con `P 0` e `initialDayForWithdraw 0` hay una sola guía, con `lineThickness 0` y `label ""`.
- **CA-03.5** (RF-03.2) — Con hoy `2026-10-13`, `utcDiff -5`, `initialDayForWithdraw 30` y `M 365`, la guía de retiro tiene `date "2026-11-12"` y `toDate "2027-10-12"`.
- **CA-03.6** (RF-03.3) — Sin `adminBlooming`, el preview responde `403`. Con `inflectionPoint >= maturityTimeInDays` responde `400`. Con `simulation: false` la respuesta no tiene la clave `data`.
- **CA-03.7** (RF-03.4) — La proyección de un bono privado que no autoriza al usuario responde `404 ERR_BOND_OBJECT_NOT_FOUND`; un admin de una agencia del bono, o de la principal, sí la recibe.
- **CA-03.8** (RF-03.5) — Las guías de un bono de otro usuario responden `403` para un no admin. Para el dueño, con `createdAt 2026-01-01T15:00Z`, `P 30` e `initialDayForWithdraw 10`, `G1.date` es `"2026-01-31"`, `G2.date` es `"2026-01-11"`, y `G2` no tiene `toDate`.
- **CA-03.9** (RF-03.4) — `GET /bond-objects/{id}/chart?simulation=true` sobre un bondObject legacy (`_id` de 17 caracteres) devuelve la misma proyección que sobre uno nuevo con los mismos términos.
- **CA-03.10** (RF-03.3) — Con `amount 1000.004`, `M 10`, `R 100`, `P 0` y las dos penalidades en `0`, el día 10 de la vista previa da `ganancia 2000` y `perdida 1000`: se simuló con `amount 1000` y penalidades de 100. Sin el redondeo la ganancia daría `2000.01`, y sin el default la pérdida daría `2000`.
- **CA-03.11** (RF-03.6) — Con `amount 1000`, `M 10`, `R 10`, `P 5`, `Pf 50`, `initialDate 2026-01-01T15:00Z`, hoy `2026-01-08T15:00Z` y `utcDiff -5` (7 días transcurridos), la evolución tiene 11 puntos, del `2026-01-01` al `2026-01-11`:
  - días 0 a 5: `inversion 1000`, `perdida 1000` y `ganancia 1000`;
  - día 6: `inversion 1020` y `perdida 1010`, sin `ganancia`;
  - día 7 (hoy): `inversion 1040`, `perdida 1020`, `atMoment 1040`, `perdidaAtMoment 1020` y `bulletClass "lastBullet"`;
  - días 8 a 10: `ganancia 1060`, `1080` y `1100`, sin `inversion` ni `perdida`; el día 10 lleva además `gananciaFinal 1100`.
- **CA-03.12** (RF-03.6) — Con `P 0`, `Pi 100`, `Pf 50` y 2 días transcurridos (el resto como en CA-03.11), el día 1 da `inversion 1010` y `perdida 1005`: se usa `Pf` y no `Pi`, que daría `1000`.
- **CA-03.13** (RF-03.6) — Con los términos de CA-03.11 y 15 días transcurridos (bono vencido), los 11 puntos llevan `inversion` y `perdida`, ninguno lleva `atMoment`, `perdidaAtMoment` ni `bulletClass`, y el día 10 da `inversion 1100`, `perdida 1050` y `gananciaFinal 1100`.
- **CA-03.14** (RF-03.6) — Con `amount 1000.50` y `R 7.5`, ningún valor se calcula con 1000 ni con 7 (comparado contra la fórmula en float), y todos los valores son números con 2 decimales, ninguno `string`. Con `initialDate 2026-03-10T03:00Z` y `utcDiff -5`, el día 0 es `"2026-03-09"`.
- **CA-03.15** (RF-03.6) — En al menos 30 casos calculados con el código de ms-feed (`bound/index.js:14-122`, con Node y moment, la zona del proceso en `utcDiff`, `now` fijo y `parseInt` reemplazado por `Number` en los porcentajes), coinciden las fechas, el día de `lastBullet` y todos los valores. Los `string` del legado se comparan como números.
- **CA-03.16** (RF-03.7) — `GET /bond-bounds/{id}/chart`:
  - el dueño recibe `200` con `M + 1` puntos en `data` y las mismas guías que `/chart-guides`, y un admin que no es el dueño también;
  - otro usuario recibe `403 ERR_FORBIDDEN`, y un bono inexistente o borrado, `404 ERR_BOND_BOUND_NOT_FOUND`;
  - un id fuera de formato responde `400 ERR_INVALID_ID` sin leer el repositorio;
  - un `_id` legacy de 17 caracteres responde igual que uno nuevo.
- **CA-03.17** (RF-03.4) — `GET /bond-objects/{id}/chart` de un bondObject de la agencia A, activo y público, responde `404 ERR_BOND_OBJECT_NOT_FOUND` a un admin de B que no es la principal. Un admin de la agencia principal lo recibe aunque esté inactivo o sea privado.
- **CA-03.18** (RF-03.5, RF-03.7) — Dado un bono de la agencia B, `/chart-guides` y `/chart` responden `404 ERR_BOND_BOUND_NOT_FOUND` a un admin de A que no es la principal y a un usuario de A que no es el dueño, sin `403`. Un admin de la agencia principal recibe `200`.

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
| BLM-0078 | Caso de uso `ChartGuides` en bondBound, con control de dueño o admin | impl | F5 | RF-03.5 | BLM-0030, BLM-0077, BLM-0007 | `internal/core/bondBound/application/bondBound.chartGuides.go`, `bondBound.application.go`, `internal/core/bondBound/ports/bondBound.ports.go` (`ChartGuides`), `internal/core/bondBound/domain/bondBound.domain.go` (`ProjectionTerms`), `bondBound.dto.domain.go` (`BondBoundChart`), `bondBound.errors.go` y `bondBound.httpStatus.go` (`ERR_BOND_BOUND_NOT_FOUND`), `internal/core/shared/domain/shared.errors.go` y `shared.httpStatus.go` (`ERR_FORBIDDEN`) |
| BLM-0085 | Handler y ruta `GET /bond-bounds/{id}/chart-guides` | impl | F5 | RF-03.5 | BLM-0078, BLM-0084, BLM-0026 | `cmd/api/handlers/bondBounds/bondBounds.chartGuides.get.go`, `bondBounds.handler.go` (ruta) |
| BLM-0086 | Tests del caso de uso y del handler de guías de bondBound (dueño, admin, 404) | test | F5 | CA-03.8 | BLM-0085 | `internal/core/bondBound/application/bondBound.chartGuides_test.go`, `bondBound.fakes_test.go` (`FindOneByID` en memoria), `cmd/api/handlers/bondBounds/bondBounds.chartGuides.get_test.go`, `bondBounds.testServer_test.go` (`ChartGuides` en `fakeApp`) |
| BLM-0143 | `shared_domain`: `BoundChartPoint` y función pura `BoundProjection` (evolución del bono comprado) | impl | F5 | RF-03.6 | BLM-0029 | `internal/core/shared/domain/shared.boundProjection.go`, `shared.chart.go` (`BoundChartPoint`) |
| BLM-0144 | Tests de tabla de `BoundProjection`, con casos calculados con el código de ms-feed (el comentario del test dice cómo se generaron) | test | F5 | CA-03.11, CA-03.12, CA-03.13, CA-03.14, CA-03.15 | BLM-0143 | `internal/core/shared/domain/shared.boundProjection_test.go`, `testdata/legacy-bound-projection.json` |
| BLM-0145 | Caso de uso `Chart` en bondBound: guías y evolución, con control de dueño o admin | impl | F5 | RF-03.7 | BLM-0143, BLM-0078 | `internal/core/bondBound/application/bondBound.chart.go`, `internal/core/bondBound/ports/bondBound.ports.go` (`Chart`), `internal/core/bondBound/domain/bondBound.dto.domain.go` (`BondBoundChart.Data`) |
| BLM-0146 | Handler y ruta `GET /bond-bounds/{id}/chart` | impl | F5 | RF-03.7 | BLM-0145, BLM-0085 | `cmd/api/handlers/bondBounds/bondBounds.chart.get.go`, `bondBounds.handler.go` (ruta) |
| BLM-0147 | Tests del caso de uso y del handler de la gráfica de bondBound (dueño, admin, 403, 404, id legacy) | test | F5 | CA-03.16 | BLM-0146 | `internal/core/bondBound/application/bondBound.chart_test.go`, `cmd/api/handlers/bondBounds/bondBounds.chart.get_test.go`, `bondBounds.testServer_test.go` (`Chart` en `fakeApp`) |

BLM-0028, BLM-0029, BLM-0030 y BLM-0031 van en la fase F2. BLM-0060, BLM-0061 y BLM-0062 van en la F3 (catálogo). BLM-0078, BLM-0085 y BLM-0086 van en la F5, cuando ya existe el repositorio de `bondBound` (BLM-0077). La evolución del bono comprado, que sale de ms-feed (BLM-0143 a BLM-0147), también va en la F5. Ver `tasks.md`.

El alcance de agencia de RF-03.4 (O12) va en BLM-0160 y BLM-0161 (spec 02); el de RF-03.5 y RF-03.7 (B38), en BLM-0162 y BLM-0163 (spec 06).

## 11. Fuera de alcance y preguntas abiertas

- La variable `totalProfit` y el cálculo de `maximum` del legado están muertos o comentados y no se portan.
- **Pregunta:** ¿el front usa `labelRotation` y `lineThickness` tal cual, o se pueden simplificar las guías? Por ahora se mantienen idénticas.
- **Pregunta:** ¿el front usa `bulletClass` (la clase del punto de hoy en amCharts)? Por ahora se mantiene por paridad.
- **Pregunta:** la evolución de un bono vendido o canjeado sigue acumulando hasta hoy, como en ms-feed. ¿Debería cortarse en `soldDate`/`tookDate`? Por ahora se mantiene la paridad.
