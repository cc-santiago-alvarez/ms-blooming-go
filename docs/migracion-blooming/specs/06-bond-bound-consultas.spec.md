# Spec 06 — Consultas de bonos comprados (`bondBound`)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondBound` |
| Handlers | `cmd/api/handlers/bondBounds` |
| Colección | `bondBound` (lookups a `profile`, `bondObject` y `bondType`) |
| Depende de | Spec 00, spec 04 (entidad y repositorio) |

## 1. Resumen y alcance

Lecturas de bonos comprados:
- para el **admin:** listado paginado con búsqueda y filtros de fecha, bonos de un usuario, compradores de un producto y baja lógica;
- para el **inversionista:** sus bonos, los recientes, el detalle y la línea de tiempo.

También define los cálculos derivados que se muestran (acumulado, días transcurridos, "creciendo"/"creció") y el constructor de filtros de fecha que reutiliza la spec 07.

## 2. Origen legacy

Archivo: `backend/server/model/bond.bound.js`.

| Operación legacy | Línea | Tipo | Destino |
|---|---|---|---|
| `bondBounds` | 27 | publicación | **Descartada**: sin filtros ni permisos; la reemplaza RF-06.1 |
| `bondBoundByUserLogged` | 35 | publicación | Fusionada en RF-06.5 |
| `getInvestorsById` | 64 | publicación | Fusionada en RF-06.8 |
| `bondBound.list` | 78 | método | RF-06.1 |
| `bondBound.listStatus` | 261 | método | RF-06.2 |
| `bondBound.listByUserIdAndBondObjectId` | 268 | método | RF-06.3 |
| `bondBound.listByUserlogged` | 283 | método | RF-06.4 |
| `bondBound.get` | 350 | método | RF-06.6 |
| `bondBound.getDetails` | 359 | método | RF-06.7 |
| `bondBound.listByBondObjectId` | 406 | método | RF-06.8 |
| `bondBound.update` | 526 | método | **Descartado**: `$set {}` vacío |
| `bondBound.remove` | 661 | método | RF-06.10 |
| `bondBound.getBondPurchasedForChart` | 1364 | método | RF-06.9 |
| `bondBoundExtraInfo` | 1432 | helper | RF-06.11 |
| `bondBound.byUserLogged` | 1533 | método | RF-06.5 |

## 3. Requisitos funcionales

- **RF-06.1** — Listado de administración paginado.
  - **Filtro base:** `isRemove: false` más `AgencyScope(ctx)`. Filtros opcionales: `statusId` y `userId` (exacto).
  - **Búsqueda de texto** (`search`, el `query.userId` del legado): regex que no distingue mayúsculas, con el texto **escapado**. Busca en `profile.id`, `profile.first_name`, `profile.last_name`, `profile.email`, `bondObject.key`, `bondObject.name`, `bondObject.description` y `bondType.name`.
  - **Filtro de fecha** sobre `createdAt` (RF-06.12).
  - **Lookups:** `profile` (left join), `bondObject` y `bondType` (inner join, paridad: los bonos huérfanos no aparecen).
  - **Orden:** `createdAt` desc.
  - **Paginación** (RF-00.18), con `total` calculado en la misma agregación (`$facet`).
  - **Cada item:** `{_id, key, status: {_id, name}, amount, maturityTimeInDays, initialDate, maturityRewardsPercentage, profile: {first_name, last_name}, bondType: {name}, bondObject: {name}, createdAt, month, year, daysPassed, acumulateAmount}`.
- **RF-06.2** — Estados. Devuelve el enum `[{_id, name}]` de los 6 estados, en orden.
- **RF-06.3** — Bonos de un usuario para un producto (admin). Devuelve los bonos con `isRemove: false`, `userId`, `bondObjectId` y `statusId ∈ {1,2,3,6}`, más `AgencyScope`.
- **RF-06.4** — Mis bonos (paginado).
  - Filtro: `isRemove: false`, `userId` y `agencyId` de la sesión (paridad), y opcionalmente `statusId` (uno o varios).
  - Orden: `createdAt` desc. Paginación real (corrige B10).
  - Cada item: `{_id, bondObject: {name}, amount, key, initialDate, maturityRewardsPercentage, maturityTimeInDays, statusId, earlyCashTerm, maturityDate}` más los derivados de RF-06.11.
- **RF-06.5** — Mis bonos recientes.
  - Los 5 bonos más recientes del usuario con `isRemove: false` y `statusId ∈ {1,2}`. Sin filtro de agencia, igual que el legado.
  - Campos: `{_id, amount, key, initialDate, maturityRewardsPercentage, maturityTimeInDays, earlyCashTerm, bondObjectId, statusId}`, más `bondObject: {name, earlyCashTerm}` (con `$lookup`, no una consulta por item) y los derivados de RF-06.11.
- **RF-06.6** — Bono por id (dueño o admin). Devuelve el documento si `isRemove: false`. Si no existe, `ERR_BOND_BOUND_NOT_FOUND`; si es ajeno y el usuario no es admin, `ERR_FORBIDDEN`.
- **RF-06.7** — Detalle (dueño o admin).
  - El bono, más `bondObject` con proyección pública (sin `authorizedUserIds`) y `bondType`, `status: {_id, name}`, `daysPassed` y `acumulateAmount`.
  - Se omiten `statusId`, `userId`, `bondObjectId` y `bondTypeId` (paridad).
  - Fuera del alcance de agencia de la sesión (`AgencyScope`) es `ERR_BOND_BOUND_NOT_FOUND`; de otro usuario y sin admin, `ERR_FORBIDDEN`.
- **RF-06.8** — Compradores de un producto (admin). Devuelve los bonos con `isRemove: false`, `bondObjectId` y `AgencyScope`, más `profile` con la proyección pública de `ProfileReader` (corrige B15).
- **RF-06.9** — Línea de tiempo de compras.
  - Devuelve `[{_id, text: "<bondObject.name> (<key>)", start, end}]`, con `start == end == initialDate` en formato `%Y-%m-%d %H:%M`, en la zona `utcDiff`.
  - Sin `userId`, devuelve los del usuario de la sesión. Con un `userId` distinto exige admin.
- **RF-06.10** — Baja lógica (admin) de un bono del alcance de agencia de la sesión (`AgencyScope`). Marca `isRemove: true`; si no existe o es de otra agencia, `ERR_BOND_BOUND_NOT_FOUND`.
- **RF-06.11** — Derivados (funciones puras), con `today = StartOfDay(now, u)` y `start = StartOfDay(initialDate, u)`:
  - `daysPassed = DaysBetween(start, today)`.
  - `acumulateAmount`: si `daysPassed <= M` y `statusId == '1'`, vale `Round2((amount × (1 + R/100)) / M × daysPassed)`; si no, `0` (paridad de fórmula).
  - `growing = today >= start + P días && today < start + (M + 1) días`.
  - `grew = today >= start + (M + 1) días`.
  - `initialDateFormatted` y `maturityDateFormatted`: formato `D MMM YYYY` en `u`.
- **RF-06.12** — Filtros de fecha comunes (`filter`), siempre en la zona `utcDiff` de la sesión:
  - `aDate` (`aDate=YYYY-MM-DD`): `[StartOfDay(aDate), StartOfDay(aDate) + 1 día)`;
  - `betweenDates` (`startDate`, `endDate`): `[StartOfDay(startDate), StartOfDay(endDate) + 1 día)`;
  - `aMonth` (`month`, `year`): el rango del mes calendario;
  - `aYear` (`year`): el rango del año.

  Un `betweenDates` con `startDate` posterior a `endDate` es inválido.

  Se implementan como **rangos sobre el campo de fecha**, no como `$month`/`$year` en UTC (corrige B23). El rango `[desde, hasta)` lo calcula `DateFilter.Range(u)` en el dominio. La spec 07 los reutiliza sobre `createdAt` o `initialDate`.

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| GET | `/bond-bounds` | admin | RF-06.1 |
| GET | `/bond-bounds/statuses` | usuario | RF-06.2 |
| GET | `/bond-bounds/me` | usuario | RF-06.4 |
| GET | `/bond-bounds/me/recent` | usuario | RF-06.5 |
| GET | `/bond-bounds/timeline` | usuario (admin con `userId` ajeno) | RF-06.9 |
| GET | `/bond-bounds/{id}` | dueño o admin | RF-06.6 |
| GET | `/bond-bounds/{id}/details` | dueño o admin | RF-06.7 |
| DELETE | `/bond-bounds/{id}` | admin | RF-06.10 |
| GET | `/users/{userId}/bond-bounds?bondObjectId=` | admin | RF-06.3 |
| GET | `/bond-objects/{id}/bond-bounds` | admin | RF-06.8 |

**`GET /bond-bounds?page=1&limit=10&statusId=1&search=juan&filter=aMonth&month=10&year=2026`** → `200`

```json
{ "items": [ { "_id": "…", "key": "BP12-58", "status": { "_id": "1", "name": "Iniciado" }, "amount": 1000,
               "maturityTimeInDays": 365, "initialDate": "…", "maturityRewardsPercentage": 15,
               "profile": { "first_name": "Juan", "last_name": "Pérez" }, "bondType": { "name": "Plus" },
               "bondObject": { "name": "Bono Plus 12M" }, "createdAt": "…", "month": 10, "year": 2026,
               "daysPassed": 30, "acumulateAmount": 94.52 } ],
  "total": 25, "page": 1, "limit": 10, "totalPages": 3 }
```

**`GET /bond-bounds/me?statusId=1&statusId=2&page=1&limit=5`** → `{items, total, page, limit, totalPages}`. Cada item es el de RF-06.4.

**Cambio de contrato respecto al legado:**
- `list` respondía `{bondBound, total}`; `listByUserlogged` usaba `skip`/`limitBondResults` sin total.
- `byUserLogged` y `getBondPurchasedForChart` respondían `{success, code, data}`.
- `month` y `year` ahora se calculan en la zona de la sesión.

**Errores:**
- `ERR_INVALID_ID` (400): un id de path, query o body no es UUID v4 ni Meteor de 17 caracteres (spec 00, RF-00.19).
- `ERR_BOND_BOUND_NOT_FOUND` (404), `ERR_FORBIDDEN` (403).
- `ERR_INVALID_DATE_FILTER` (400): `filter` desconocido, o fechas, mes o año inválidos.
- `ERR_INVALID_STATUS` (400): `statusId` fuera del enum.
- `ERR_INVALID_SEARCH` (400): `search` de más de 100 caracteres.

## 5. Reglas de negocio

Ver RF-06.11 y RF-06.12.

**Búsqueda segura:** el texto pasa por `regexp.QuoteMeta` antes de armar el `$regex`. Se aceptan como máximo 100 caracteres; uno más largo responde `ERR_INVALID_SEARCH`.

**Ejemplo de `acumulateAmount`:** `amount 1000`, `R 10`, `M 100`, estado `'1'` y día 50 dan `(1100 / 100) × 50 = 550`.

## 6. Modelo de datos

Usa `BondBound` (spec 04). DTOs de salida: `BondBoundAdminItem`, `BondBoundWithProfile` (RF-06.8), `BondBoundMineItem`, `BondBoundDetail` y `TimelineItem`, en `bondBound/ports`, porque combinan datos de otros dominios. El estado sale como `StatusRef` (`{_id, name}`), del dominio. `BondBoundMineItem` sirve a RF-06.4 y RF-06.5, con `bondObjectId` y `maturityDate` en los dos; `bondObject.earlyCashTerm` solo sale en RF-06.5. El producto del detalle es `BondObjectPublic`, sin `authorizedUserIds`.

El repositorio devuelve filas con el bono completo y lo que traen los lookups (`BondBoundAdminRow`, `BondBoundWithObject`, `BondBoundDetailRow`) y `TimelineItem` ya armado. La aplicación arma los DTOs de salida con los derivados de RF-06.11.

Índices: los de la spec 04 §6 cubren estos filtros.

## 7. Ports y dependencias

- `BondBoundApplicationPorts`: `List`, `Statuses`, `ListByUserAndObject`, `ListMine`, `ListRecent`, `GetByID`, `GetDetails`, `ListByObject`, `Timeline`, `Remove`.
- `BondBoundMongoccRepositoryPorts`: un adapter por consulta, más `DateRangeStage(filter, field, u)`, que arma el `$match` con el rango de `DateFilter.Range(u)` y reutiliza la spec 07.
- `ProfileReader` (spec 00), para la proyección pública en RF-06.8.

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| B10 | `listByUserlogged` con `skip` salta siempre 5, y `limitBondResults` reordena después de saltar. | Corregir | Paginación real. | BLM-0105 |
| B15 | `listByBondObjectId` y `getInvestorsById` devuelven el `profile` completo, con el hash del password. Tampoco filtran `isRemove` ni agencia. | Corregir | Seguridad. | BLM-0104 |
| B23 | `aMonth` y `aYear` usan `$month`/`$year` en UTC: una compra del 31-oct a las 22:00 (UTC-5) cae en noviembre. | Corregir | Consistencia con la zona de la sesión. | BLM-0102 |
| B24 | La búsqueda usa el texto del usuario como `$regex` sin escapar (regex inválida que hace fallar la agregación, o ReDoS). | Corregir | Seguridad y robustez. | BLM-0103 |
| B25 | `getBondPurchasedForChart` formatea con `%G` (año ISO por semana): las fechas de fines de diciembre salen con el año siguiente. También usa UTC. | Corregir | Dato incorrecto. | BLM-0103 |
| B26 | `get`, `getDetails`, `getInvestorsById` y `getBondPurchasedForChart` no validan dueño ni permisos: cualquier sesión lee bonos ajenos. | Corregir | Seguridad. | BLM-0105 |
| B27 | `getDetails` devuelve el `bondObject` completo, incluido `authorizedUserIds`. | Corregir | Exposición de ids de usuarios. | BLM-0105 |

## 9. Criterios de aceptación

- **CA-06.1** (RF-06.1) — Con 25 bonos de la agencia A y una sesión admin no principal de A, `GET /bond-bounds?page=2&limit=10` devuelve 10 items ordenados por `createdAt` desc, con `total 25` y `totalPages 3`. Los bonos de la agencia B no aparecen.
- **CA-06.2** (RF-06.1) — `search=JUAN` encuentra bonos por `first_name`, `email`, `bondObject.key` o `bondType.name`, sin distinguir mayúsculas. `search=a.*(` no produce error y se busca literalmente. Un `search` de 101 caracteres responde `400 ERR_INVALID_SEARCH`.
- **CA-06.3** (RF-06.12) — Con `utcDiff -5`:
  - `filter=aDate&aDate=2026-10-13` incluye `createdAt 2026-10-14T04:59Z` y excluye `2026-10-14T05:00Z`;
  - `filter=aMonth&month=10&year=2026` incluye `2026-11-01T03:00Z`;
  - `filter=betweenDates` con `startDate` posterior a `endDate` responde `400 ERR_INVALID_DATE_FILTER`.
- **CA-06.4** (RF-06.11) — `acumulateAmount` con `amount 1000`, `R 10`, `M 100`, día 50 y estado `'1'` es `550`. Con estado `'2'` es `0`, y con `daysPassed 101` también es `0`.
- **CA-06.5** (RF-06.11) — Con `P 30` y `M 100`: el día 29 da `growing false`; el día 30, `growing true`; el día 100, `growing true` y `grew false`; el día 101, `growing false` y `grew true`.
- **CA-06.6** (RF-06.2) — `GET /bond-bounds/statuses` devuelve 6 items: `[{1, Iniciado}, …, {6, Canjeado}]`.
- **CA-06.7** (RF-06.3) — `GET /users/u1/bond-bounds?bondObjectId=X` devuelve solo los estados 1, 2, 3 y 6. Un no admin recibe `403`.
- **CA-06.8** (RF-06.4) — `GET /bond-bounds/me?statusId=1&statusId=2&page=2&limit=5` con 12 bonos propios devuelve los items 6 a 10, cada uno con `bondObject.name`, `initialDateFormatted`, `growing` y `grew`.
- **CA-06.9** (RF-06.5) — `GET /bond-bounds/me/recent` devuelve como máximo 5 bonos en estados 1 o 2, ordenados por `createdAt` desc, cada uno con `bondObject.name` y `bondObject.earlyCashTerm`.
- **CA-06.10** (RF-06.6, RF-06.7) — Para un no admin, `GET /bond-bounds/{id}` y `/details` de un bono ajeno responden `403`, y de un id inexistente `404`. El detalle propio no tiene `statusId`, `userId`, `bondObjectId`, `bondTypeId` ni `bondObject.authorizedUserIds`.
- **CA-06.11** (RF-06.8) — `GET /bond-objects/X/bond-bounds` devuelve `profile` sin `password` y excluye los bonos borrados.
- **CA-06.12** (RF-06.9) — Un bono con `initialDate 2026-12-31T20:00Z` y `utcDiff -5` da `start "2026-12-31 15:00"`. Un no admin que pide `?userId=otro` recibe `403`.
- **CA-06.13** (RF-06.10) — `DELETE /bond-bounds/{id}` deja `isRemove: true` y el bono desaparece de RF-06.1 y RF-06.4. Un admin de otra agencia, no principal, recibe `404`.
- **CA-06.14** (RF-06.4, RF-06.7) — Con bonos propios legacy (17 caracteres) y nuevos (UUID), `GET /bond-bounds/me` los lista juntos, y `GET /bond-bounds/{idLegacy}/details` resuelve los lookups a un `bondObject` y un `bondType` legacy.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0101 | Dominio: funciones puras `Accumulated`, `Progress` y `FormatShortDate`; DTOs `ListFilters` y `DateFilter` con `Normalize`/`Validation` y el rango `DateFilter.Range` | impl | F6 | RF-06.1, RF-06.11, RF-06.12 | BLM-0075, BLM-0009 | `internal/core/bondBound/domain/bondBound.derived.go`, `bondBound.filters.domain.go`, `bondBound.errors.go` y `bondBound.httpStatus.go` (`ERR_INVALID_DATE_FILTER`, `ERR_INVALID_STATUS`, `ERR_INVALID_SEARCH`) |
| BLM-0102 | `$match` de rango de fecha `DateRangeStage(filter, field, u)` sobre `DateFilter.Range`, reutilizable por la spec 07 | impl | F6 | RF-06.12 | BLM-0101 | `internal/core/bondBound/infrastructure/mongocc/bondBound.dateRange.go` |
| BLM-0103 | Adapters de consulta: listado admin (`$facet`, lookups, búsqueda escapada), por usuario y producto, míos paginados, recientes, detalle, por producto, línea de tiempo (`%Y` en `u`) y `softDelete` | impl | F6 | RF-06.1, RF-06.3, RF-06.4, RF-06.5, RF-06.7, RF-06.8, RF-06.9, RF-06.10 | BLM-0102, BLM-0077 | `internal/core/bondBound/infrastructure/mongocc/bondBound.list*.adapter.go`, `bondBound.findDetails.adapter.go`, `bondBound.timeline.adapter.go`, `bondBound.softDelete.adapter.go`, `bondBound.mongocc.repository.go` (lookups, zona horaria y página con `$facet`), `bondBound.search.go` (búsqueda escapada, reutilizada por la spec 07), `internal/core/bondBound/ports/bondBound.ports.go` (métodos del repositorio y filas) |
| BLM-0104 | Casos de uso admin: `List`, `ListByUserAndObject`, `ListByObject` (perfil público) y `Remove` | impl | F6 | RF-06.1, RF-06.3, RF-06.8, RF-06.10 | BLM-0103, BLM-0017 | `internal/core/bondBound/application/bondBound.list.go`, `bondBound.listByUserAndObject.go`, `bondBound.listByObject.go`, `bondBound.remove.go`, `internal/core/bondBound/ports/bondBound.ports.go` (`List`, `ListByUserAndObject`, `ListByObject`, `Remove`, `BondBoundAdminItem`, `BondBoundWithProfile`), `internal/core/bondBound/domain/bondBound.status.domain.go` (`StatusRef`) |
| BLM-0105 | Casos de uso del inversionista: `Statuses`, `ListMine`, `ListRecent`, `GetByID`, `GetDetails` y `Timeline`, con control de dueño | impl | F6 | RF-06.2, RF-06.4, RF-06.5, RF-06.6, RF-06.7, RF-06.9, RF-06.11 | BLM-0103, BLM-0007 | `internal/core/bondBound/application/bondBound.statuses.go`, `bondBound.listMine.go`, `bondBound.listRecent.go`, `bondBound.getById.go`, `bondBound.getDetails.go`, `bondBound.timeline.go`, `bondBound.application.go` (`inAgencyScope`, que también usa `bondBound.remove.go`), `internal/core/bondBound/ports/bondBound.ports.go` (métodos, `BondBoundMineItem`, `BondBoundDetail`, `BondObjectPublic`), `internal/core/bondBound/domain/bondBound.status.domain.go` (`StatusRefs`), `bondBound.filters.domain.go` (`MineFilters`) |
| BLM-0106 | Handlers y rutas, incluidas `/users/{userId}/bond-bounds` y `/bond-objects/{id}/bond-bounds` | impl | F6 | RF-06.1, RF-06.2, RF-06.3, RF-06.4, RF-06.5, RF-06.6, RF-06.7, RF-06.8, RF-06.9, RF-06.10 | BLM-0104, BLM-0105, BLM-0084, BLM-0026 | `cmd/api/handlers/bondBounds/bondBounds.*.get.go`, `bondBounds.remove.delete.go`, `bondBounds.handler.go` (rutas), `bondBounds.query.go` (`queryDateFilter`, que reutiliza la spec 07), `cmd/api/handlers/shared/handlers.pagination.go` (`QueryInt` exportado) |
| BLM-0107 | Tests de tabla de los derivados y de los rangos de fecha | test | F6 | CA-06.3, CA-06.4, CA-06.5 | BLM-0101, BLM-0102 | `internal/core/bondBound/domain/bondBound.derived_test.go`, `bondBound.filters_test.go`, `internal/core/bondBound/infrastructure/mongocc/bondBound.dateRange_test.go` |
| BLM-0108 | Tests unitarios de casos de uso y handlers (permisos, dueño, 404) | test | F6 | CA-06.6, CA-06.7, CA-06.10, CA-06.12 | BLM-0106 | `internal/core/bondBound/application/bondBound.list*_test.go`, `bondBound.get*_test.go` |
| BLM-0109 | Tests de integración de las agregaciones (paginación, búsqueda escapada, scope de agencia, proyecciones) | test | F6 | CA-06.1, CA-06.2, CA-06.8, CA-06.9, CA-06.11, CA-06.13, CA-06.14 | BLM-0103 | `internal/core/bondBound/infrastructure/mongocc/bondBound.list*_integration_test.go` |

## 11. Fuera de alcance y preguntas abiertas

- **Pregunta:** ¿se debe impedir borrar (soft delete) un bono en estado `'1'`, que tiene dinero invertido? El legado lo permite.
- **Pregunta:** RF-06.4 mantiene el filtro por la agencia de la sesión (paridad). Si un usuario cambia de agencia, deja de ver sus bonos anteriores. ¿Es intencional?
- La fórmula de `acumulateAmount` prorratea capital más interés; se mantiene tal cual porque el front la muestra así.
