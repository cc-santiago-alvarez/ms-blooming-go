# Spec 07 — Reportes y KPIs

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondBound` |
| Handlers | `cmd/api/handlers/bondBounds` |
| Colección | `bondBound` (lectura); genera archivos vía ms-exporter |
| Depende de | Spec 00 (`ExporterPort`, `AgencyReader`), spec 06 (`DateRangeStage`, búsqueda escapada) |

## 1. Resumen y alcance

Salidas para el dashboard de administración:
- dos reportes Excel, que arma **ms-exporter**: el detalle de bonos comprados y el resumen por inversionista;
- el indicador `total` / `count` de bonos por estado.

## 2. Origen legacy

Archivo: `backend/server/model/bond.bound.js`.

| Operación legacy | Línea | Destino |
|---|---|---|
| `bondBound.reportPurchasedBonds` | 940 | `GET /bond-bounds/reports/purchased` (RF-07.1) |
| `bondBound.reportSummaryPurchasedBonds` | 1125 | `GET /bond-bounds/reports/purchased-summary` (RF-07.2) |
| `bondBound.countByStatusId` | 1212 | `GET /bond-bounds/kpis/count-by-status` (RF-07.3) |

Del lado de ms-exporter: `ms-exporter/src/modules/bond/bond.routes.js` y `bond.purchased.service.js`. Las columnas son Key, Usuario, Tipo, Bono, Monto, Fecha de inicio, Fecha de maduración, Tiempo de maduración en días, Ganancia (%) y Estatus.

## 3. Requisitos funcionales

- **RF-07.1** — Reporte de bonos comprados (admin).
  - **Filtro:** `isRemove: false` y la `agencyId` de la sesión (paridad: el reporte **no** abre a todas las agencias aunque la agencia sea la principal).
  - **`statusId`:** `'1'` por defecto; `'0'` incluye todos los estados.
  - Filtro opcional de `userId` (exacto) y búsqueda escapada (`search`) sobre `profile.first_name`, `bondObject.name` y `bondType.name` (paridad de campos).
  - Filtro de fecha de RF-06.12 sobre `createdAt`.
  - **Orden:** `createdAt` desc.
  - **Filas:** `{key, status: <nombre>, amount, maturityTimeInDays, initialDate, maturityDate, maturityRewardsPercentage, profile: {first_name, last_name}, bondType: {name}, bondObject: {name}, createdAt}`.
  - Se envía a `ExporterPort.Excel("purchasedBonds", {currency, bondBounds})`, con `currency = AgencyReader.SymbolCurrency(session.agencyId)`.
- **RF-07.2** — Reporte resumen por inversionista (admin).
  - **Filtro:** `isRemove: false`, `agencyId` de la sesión y `statusId` (`'1'` por defecto; `'0'` todos).
  - Agrupa por `userId` con `total = Σ amount` y hace `$lookup` a `profile` con la proyección `{first_name, last_name}`.
  - Se envía a `ExporterPort.Excel("summaryPurchasedBonds", {currency, bondBounds})`.
- **RF-07.3** — KPI de bonos por estado.
  - **Filtro:** `isRemove: false` más `AgencyScope(ctx)`. `statusId` se aplica solo si es un estado válido (paridad: si no, se cuentan todos).
  - **Usuario:**
    - un admin puede pasar `userId` o no pasarlo (toda la agencia);
    - un no admin siempre recibe sus propios números: `userId` se fuerza al de la sesión, y si manda otro, `ERR_FORBIDDEN`.
  - Filtro de fecha de RF-06.12 sobre `createdAt` y búsqueda escapada sobre `profile.id`, `first_name`, `last_name`, `email` y `bondObject.name`.
  - **Respuesta:** `{total: Σ amount, count, status: <nombre del estado> | ""}`.
- **RF-07.4** — Entrega de archivos.
  - Los reportes responden el binario con `Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet` y `Content-Disposition: attachment; filename="<FILENAME_REPORT>.xlsx"`.
  - El nombre es `reporte-purchased-bonds_<YYYY-MM-DD_HHmmss>` o `reporte-summary-purchased-bonds_<…>`, con la hora en `utcDiff`.
  - Si ms-exporter falla, `502 ERR_EXPORT_FAILED`.

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| GET | `/bond-bounds/reports/purchased` | admin | RF-07.1, RF-07.4 |
| GET | `/bond-bounds/reports/purchased-summary` | admin | RF-07.2, RF-07.4 |
| GET | `/bond-bounds/kpis/count-by-status` | usuario (alcance según RF-07.3) | RF-07.3 |

**Parámetros comunes de query:**
- `statusId`
- `userId`
- `search`
- `filter` ∈ {`aDate`, `betweenDates`, `aMonth`, `aYear`}, con sus valores: `aDate`, o `startDate` y `endDate`, o `month` y `year`

**`GET /bond-bounds/reports/purchased?statusId=0&filter=aMonth&month=10&year=2026`** → `200` con el binario xlsx.

**`GET /bond-bounds/kpis/count-by-status?statusId=1`** → `200`

```json
{ "total": 125000, "count": 120, "status": "Iniciado" }
```

**Cambio de contrato respecto al legado:**
- Los reportes respondían `{fileName, code, document: <Buffer JSON>, headerContentType}`, y el front armaba el archivo. Ahora el endpoint devuelve el archivo directamente.
- El KPI respondía `{success, code, data}`.
- El KPI ya no recibe `utcDiff` del cliente; usa el de la sesión.

**Errores:**
- `ERR_INVALID_ID` (400): un id de path, query o body no es UUID v4 ni Meteor de 17 caracteres (spec 00, RF-00.19).
- `ERR_FORBIDDEN` (403).
- `ERR_INVALID_DATE_FILTER` (400).
- `ERR_INVALID_STATUS` (400): `statusId` de un reporte fuera del enum y distinto de `'0'`.
- `ERR_INVALID_SEARCH` (400): `search` de más de 100 caracteres.
- `ERR_EXPORT_FAILED` (502; el legado devolvía `code -1000` con status 200).

## 5. Reglas de negocio

- **Normalización de `statusId` (reportes):** vacío → `'1'`; `'0'` → sin filtro de estado; otro valor → debe estar en el enum, si no `ERR_INVALID_STATUS`.
- **Normalización de `statusId` (KPI):** vacío o fuera del enum → sin filtro de estado y `status: ""` (paridad).
- **Búsqueda:** como en la spec 06, como máximo 100 caracteres; uno más largo responde `ERR_INVALID_SEARCH`.
- Los rangos de fecha usan `DateRangeStage` (BLM-0102), siempre en la zona de la sesión.

## 6. Modelo de datos

Esta spec no persiste nada.
- Las filas del reporte (`PurchasedReportRow`, `SummaryReportRow`) viven en `bondBound/ports`.
- El payload a ms-exporter se mantiene **idéntico** al del legado, para no tocar ms-exporter.

## 7. Ports y dependencias

- `BondBoundApplicationPorts`: `ReportPurchased`, `ReportSummary` y `CountByStatus`.
- `BondBoundMongoccRepositoryPorts`: `ReportPurchasedRows`, `ReportSummaryRows` y `CountByStatus`.
- `ExporterPort` y `AgencyReader` (spec 00).
- `DateRangeStage` y el helper de búsqueda escapada (spec 06).

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| B11 | `reportSummaryPurchasedBonds` lee `data.data.statusId` (crashea sin `data.data`) y `countByStatusId` exige `data.query`. | Corregir | Llamadas que no responden. | BLM-0110 |
| B12 | El `betweenDates` del KPI compara año, mes y día por separado: del 25-ene al 5-feb no incluye el 30-ene. Además deriva `utcDiff` de la zona del servidor o del cliente. | Corregir | Indicadores incorrectos. | BLM-0111 |
| B34 | Búsqueda con `$regex` sin escapar en los reportes y el KPI (mismo problema que B24 del listado). | Corregir | Seguridad y robustez. | BLM-0111 |
| B29 | Los filtros `aDate` y `betweenDates` del reporte ignoran `utcDiff` (zona del servidor), y `month`/`year` salen de `initialDate` en UTC. | Corregir | Mismo criterio de fechas que el listado. | BLM-0111 |
| B30 | Si falla ms-exporter, se responde `code -1000` con HTTP 200 e incluye el objeto de error interno (`err`). | Corregir | Contrato HTTP y exposición de detalles internos. | BLM-0112 |
| B31 | El KPI sin permisos permite que cualquier sesión consulte totales de la agencia o de otro usuario. | Corregir | Seguridad. | BLM-0113 |

## 9. Criterios de aceptación

- **CA-07.1** (RF-07.1, RF-07.4) — Sin `statusId`, el payload a ms-exporter contiene solo bonos en estado `'1'`, cada fila con `status: "Iniciado"`, y `currency` es el símbolo de la agencia. Con `statusId=0` contiene todos los estados. La respuesta es `200`, con el `Content-Type` de xlsx y un `Content-Disposition` que empieza por `reporte-purchased-bonds_`.
- **CA-07.2** (RF-07.1) — Con `utcDiff -5`, `filter=aMonth&month=10&year=2026` incluye un bono creado el `2026-11-01T03:00Z`. `search=plus` encuentra bonos cuyo `bondObject.name` contiene "Plus".
- **CA-07.3** (RF-07.2) — Si el usuario u1 tiene dos bonos activos de 1000 y 500, el resumen envía `{_id: u1, total: 1500, profile: {first_name, last_name}}`. Un request sin parámetros responde `200` y no falla (B11).
- **CA-07.4** (RF-07.4) — Si ms-exporter responde 500, el endpoint responde `502 ERR_EXPORT_FAILED`, sin detalles internos.
- **CA-07.5** (RF-07.3) — `filter=betweenDates&startDate=2026-01-25&endDate=2026-02-05` cuenta un bono creado el 2026-01-30 (B12).
- **CA-07.6** (RF-07.3) — Un no admin sin `userId` recibe solo sus totales, y con el `userId` de otro recibe `403`. Un admin sin `userId` recibe los totales de su agencia. `statusId=9` cuenta todos los estados y responde `status: ""`.
- **CA-07.7** (RF-07.1, RF-07.2) — Un no admin que llama a los reportes recibe `403`.
- **CA-07.8** (RF-07.1, RF-07.3) — El reporte y el KPI cuentan e incluyen por igual bonos legacy (17 caracteres) y nuevos (UUID), y `userId=<id legacy>` filtra correctamente.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0110 | DTOs `ReportFilters` y `KpiFilters`, con la normalización de `statusId` de §5 y sus errores | impl | F6 | RF-07.1, RF-07.2, RF-07.3 | BLM-0101 | `internal/core/bondBound/domain/bondBound.reports.domain.go`, `bondBound.filters.domain.go` (validación común de userId, búsqueda y fecha, compartida con `ListFilters`) |
| BLM-0111 | Adapters `ReportPurchasedRows`, `ReportSummaryRows` y `CountByStatus` (con `DateRangeStage` y búsqueda escapada) | impl | F6 | RF-07.1, RF-07.2, RF-07.3 | BLM-0110, BLM-0102, BLM-0103 | `internal/core/bondBound/infrastructure/mongocc/bondBound.report*.adapter.go`, `bondBound.countByStatus.adapter.go`, `bondBound.mongocc.repository.go` (`numericAmount`), `internal/core/bondBound/ports/bondBound.ports.go` (métodos del repositorio y filas `PurchasedReportRow`, `SummaryReportRow` y `StatusCount`) |
| BLM-0112 | Casos de uso `ReportPurchased` y `ReportSummary` (moneda, `ExporterPort` y nombre de archivo) | impl | F6 | RF-07.1, RF-07.2, RF-07.4 | BLM-0111, BLM-0016, BLM-0017 | `internal/core/bondBound/application/bondBound.reportPurchased.go`, `bondBound.reportSummary.go`, `bondBound.exportReport.go` (moneda, payload y nombre del archivo), `bondBound.application.go` (dependencias `Agencies` y `Exporter`), `internal/core/bondBound/ports/bondBound.ports.go` (`ReportPurchased`, `ReportSummary` y `ReportFile`), `cmd/api/server/server.config.go`, `server.deps.go`, `server.bondBound.go` (cableado de `ExporterPort` y `AgencyReader`) |
| BLM-0113 | Caso de uso `CountByStatus`, con el alcance por rol | impl | F6 | RF-07.3 | BLM-0111, BLM-0007 | `internal/core/bondBound/application/bondBound.countByStatus.go`, `internal/core/bondBound/ports/bondBound.ports.go` (`CountByStatus`) |
| BLM-0114 | Handlers de reportes (respuesta binaria y headers) y del KPI | impl | F6 | RF-07.1, RF-07.2, RF-07.3, RF-07.4 | BLM-0112, BLM-0113, BLM-0084, BLM-0026 | `cmd/api/handlers/bondBounds/bondBounds.reportPurchased.get.go`, `bondBounds.reportSummary.get.go`, `bondBounds.countByStatus.get.go` |
| BLM-0115 | Tests unitarios: normalización, permisos, fallo de ms-exporter y headers | test | F6 | CA-07.1, CA-07.4, CA-07.6, CA-07.7 | BLM-0114 | `internal/core/bondBound/application/bondBound.report*_test.go`, `bondBound.countByStatus_test.go` |
| BLM-0116 | Tests de integración de las agregaciones (rangos, búsqueda, agrupado) | test | F6 | CA-07.2, CA-07.3, CA-07.5, CA-07.8 | BLM-0111 | `internal/core/bondBound/infrastructure/mongocc/bondBound.report*_integration_test.go` |

## 11. Fuera de alcance y preguntas abiertas

- La variante PDF de los reportes está comentada en el legado y no se porta.
- **Pregunta:** ¿el reporte debería respetar `AgencyScope`, es decir, que la agencia principal vea todas? El legado siempre filtra por la agencia de la sesión.
