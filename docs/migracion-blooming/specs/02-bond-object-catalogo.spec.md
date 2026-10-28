# Spec 02 — Catálogo de bonos (`bondObject`)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondObject` |
| Handlers | `cmd/api/handlers/bondObjects` |
| Colección | `bondObject` (lectura de agregación sobre `bondBound`) |
| Depende de | Spec 00, spec 01 (`bondType`), spec 08 (`BondHistoryPort`) |

## 1. Resumen y alcance

El `bondObject` es el **producto**: define precio, plazo, rendimiento, términos de retiro anticipado, ventana de venta y si el bono es privado. El admin lo administra. El inversionista lo ve por tipo, entre los más comprados y entre los últimos publicados.

Cada compra copia los términos del bondObject en el `bondBound` (spec 04). Por eso editar un bondObject **no** afecta a los bonos ya comprados.

Las gráficas de proyección (`getDataChart`) están en la spec 03.

## 2. Origen legacy

Archivo: `backend/server/model/bond.object.js` (628 líneas).

| Operación legacy | Línea | Tipo | Destino |
|---|---|---|---|
| `bondObjects` | 18 | publicación | **Descartada**: sin filtro de agencia ni permisos; la reemplaza `GET /bond-objects` (RF-02.1) |
| `getBondObjectsByBondTypeId` | 26 | publicación | Fusionada en RF-02.9 |
| `getMostPurchased` | 50 | publicación | Fusionada en RF-02.10 |
| `getLatest` | 84 | publicación | Fusionada en RF-02.11 |
| `bondObject.list` | 105 | método | RF-02.1 |
| `bondObject.listGroupedByType` | 125 | método | RF-02.2 |
| `bondObject.listInvestor` | 151 | método | RF-02.8 |
| `bondObject.get` | 199 | método | RF-02.3 |
| `bondObject.getDataChart` | 221 | método | Spec 03 |
| `bondObject.insert` | 358 | método | RF-02.4 |
| `bondObject.update` | 411 | método | RF-02.5 |
| `bondObject.massiveUpdateOfKeys` | 510 | método | RF-02.6 |
| `bondObject.remove` | 529 | método | RF-02.7 |
| `bondObject.getChart` | 548 | método | **Descartado**: stub que devuelve `{}` |
| `bondObject.getBondObjectsByBondTypeId` | 566 | método | RF-02.9 |
| `bondObject.getMostPurchased` | 591 | método | RF-02.10 |
| `bondObject.getLatest` | 621 | método | RF-02.11 |
| `bondObjectExtraInfo` | 555 | helper | `maturityDateFormatted` en RF-02.9 |
| `isValidDate` | 495 | helper | Lo absorbe el diff de RF-02.5 |

## 3. Requisitos funcionales

- **RF-02.1** — Listado de administración.
  - Devuelve los bondObjects con `isRemove: false`, filtrados opcionalmente por `agencyId` (uno o más, con `$in`).
  - Orden: `name` ascendente.
  - Solo admin.
- **RF-02.2** — Listado agrupado por tipo (admin).
  - Filtro: `isRemove: false` más `AgencyScope(ctx)`. Si la agencia de la sesión es la principal, el admin **puede** filtrar por `agencyId` en el query.
  - En **una sola agregación** se agrupa por `bondTypeId` (`list: $push $$ROOT`) y se hace `$lookup` a `bondType`. Corrige el N+1 del legado.
  - Respuesta: `[{_id: bondTypeId, bondType: {...} | null, list: [...]}]`.
- **RF-02.3** — Detalle (admin).
  - Devuelve el bondObject con `isRemove: false` más `authorizedUser`: los perfiles de `authorizedUserIds`, leídos con `ProfileReader.GetPublic`, **sin** `password` (corrige O9).
  - Si no existe, `ERR_BOND_OBJECT_NOT_FOUND`.
- **RF-02.4** — Alta (admin).
  - Crea el bondObject con los campos de §6 y aplica los defaults del legado:
    - `isPrivateBond` y `active`: `false`. `available`: `false`. `authorizedUserIds`: `[]`.
    - `earlyCashTerm`: `initialPenaltyPercentage` 100, `inflectionPoint` 0, `inflectionPenaltyPercentage` 100, `blockWithdrawals` false, `initialDayForWithdraw` 0.
  - Valida (§5.1).
  - Después de guardar: `BondHistory.Add({relatedId, typeId: '1' BOND, text: 'BOND_ADDED'})` y `Activity.Add({description: 'BOND_CREATED', relatedId, agencyId, userId, params: {name, description, amount, bondTypeId}})`. Las dos son best-effort.
  - `agencyId` y `userId` de la actividad son los de la sesión, igual que en el historial. El legado tomaba `agencyId` del body, que puede traer varias agencias.
  - Responde `201` con el documento.
- **RF-02.5** — Edición (admin).
  - Reemplaza todos los campos editables con los mismos defaults y validaciones del alta.
  - Calcula el diff contra el documento anterior con `BuildChanges(old, new)` (§5.2) y registra `BondHistory.Add({relatedId, typeId: '1', text: 'BOND_UPDATED', changes, editedProperties})`.
  - No modifica los `bondBound` existentes.
  - Responde el documento actualizado.
- **RF-02.6** — Actualización masiva de clave (admin).
  - Recibe `{bonds: [id...], key}` y actualiza `key` en los bondObjects con `_id` en la lista e `isRemove: false`.
  - Responde `{matched, modified}`. Corrige O4: el legado nunca respondía.
  - Validaciones: `key` no vacía (`ERR_BOND_OBJECT_KEY_REQUIRED`) y `bonds` con 1–500 ids (`ERR_BOND_OBJECT_IDS_REQUIRED`).
- **RF-02.7** — Baja lógica (admin).
  - Marca `isRemove: true` y registra `BondHistory.Add({relatedId, typeId: '1', text: 'BOND_REMOVE'})`.
  - Los bonos ya comprados no se ven afectados.
  - Si no existe, `ERR_BOND_OBJECT_NOT_FOUND`.
- **RF-02.8** — Inversionistas de un bono (admin).
  - Agrega `bondBound` con `{isRemove: false, bondObjectId, statusId ∈ {'1','2','3','6'}}` más `AgencyScope(ctx)`, agrupa por `userId` con `numOfBounds` y hace `$lookup` a `profile`.
  - Proyección: `investor {first_name, last_name, email, id, avatar, address, avatar_url}`.
  - Respuesta: `[{_id: userId, numOfBounds, investor}]`.
- **RF-02.9** — Bonos de un tipo, para el inversionista.
  - Devuelve los bondObjects con `isRemove: false`, `active: true`, `bondTypeId` y `agencyId` que contiene el `agencyId` de la sesión. Los privados solo aparecen si el usuario está en `authorizedUserIds`. El filtro va en la consulta.
  - Es privado el bondObject cuyo `isPrivateBond` es verdadero con el criterio de los datos legacy de §6, también en la consulta.
  - Orden: `createdAt` descendente.
  - Cada item suma:
    - `maturityDateEstimated = StartOfDay(now, utcDiff) + maturityTimeInDays días`, en ISO;
    - `maturityDateFormatted` con el formato legacy `D MMM YYYY` (por ejemplo `5 Mar 2027`).
- **RF-02.10** — Más comprados.
  - Toma los 3 `bondObjectId` con más `bondBound` (`isRemove: false`, `agencyId` de la sesión).
  - Devuelve esos bondObjects con `isRemove: false`, `agencyId` que contiene la agencia de la sesión y el filtro de privados de RF-02.9, ordenados por cantidad de compras descendente.
- **RF-02.11** — Últimos publicados. Devuelve los 3 bondObjects más recientes (`createdAt` desc) con `isRemove: false`, de la agencia de la sesión, con el filtro de privados de RF-02.9.

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| GET | `/bond-objects` | admin | RF-02.1 |
| GET | `/bond-objects/grouped-by-type` | admin | RF-02.2 |
| GET | `/bond-objects/most-purchased` | usuario | RF-02.10 |
| GET | `/bond-objects/latest` | usuario | RF-02.11 |
| GET | `/bond-objects/{id}` | admin | RF-02.3 |
| GET | `/bond-objects/{id}/investors` | admin | RF-02.8 |
| GET | `/bond-types/{bondTypeId}/bond-objects` | usuario | RF-02.9 |
| POST | `/bond-objects` | admin | RF-02.4 |
| PUT | `/bond-objects/{id}` | admin | RF-02.5 |
| PATCH | `/bond-objects/keys` | admin | RF-02.6 |
| DELETE | `/bond-objects/{id}` | admin | RF-02.7 |

**`POST /bond-objects`** (el mismo body sirve para `PUT`)

```json
{
  "name": "Bono Plus 12M", "description": "…", "agencyId": ["A"], "key": "BP12",
  "bondTypeId": "d0c1…", "amount": 1000, "isPrivateBond": false, "authorizedUserIds": [],
  "maturityTimeInDays": 365, "maturityRewardsPercentage": 15,
  "active": true, "available": true,
  "saleInitDate": "2026-11-01T00:00:00Z", "saleEndDate": null,
  "earlyCashTerm": {
    "initialPenaltyPercentage": 100, "inflectionPoint": 90, "inflectionPenaltyPercentage": 75,
    "blockWithdrawals": false, "initialDayForWithdraw": 30
  }
}
```

**`GET /bond-types/{bondTypeId}/bond-objects`** → `200`

```json
[{ "_id": "…", "name": "Bono Plus 12M", "amount": 1000, "maturityTimeInDays": 365, "…": "…",
   "maturityDateEstimated": "2027-10-13T05:00:00Z", "maturityDateFormatted": "13 Oct 2027" }]
```

**`PATCH /bond-objects/keys`**: `{"bonds": ["id1","id2"], "key": "BP12"}` → `200 {"matched": 2, "modified": 2}`.

**Cambio de contrato respecto al legado:**
- `getBondObjectsByBondTypeId`, `getMostPurchased` y `getLatest` devolvían `{success, code, data}`; ahora devuelven el arreglo directo.
- `get` devolvía `authorizedUser` con los perfiles completos; ahora devuelve solo la proyección pública.

**Errores**

| Sentinel | HTTP | Código legacy | Cuándo |
|---|---|---|---|
| `ERR_INVALID_ID` | 400 | — | Un id de path o body no es UUID v4 ni Meteor de 17 caracteres (spec 00, RF-00.19) |
| `ERR_BOND_OBJECT_NOT_FOUND` | 404 | -11 | No existe o está borrado |
| `ERR_BOND_OBJECT_INVALID_BOND_TYPE` | 422 | — | `bondTypeId` inexistente o borrado |
| `ERR_BOND_OBJECT_NAME_REQUIRED` | 400 | — | Falta `name` |
| `ERR_BOND_OBJECT_KEY_REQUIRED` | 400 | — | Falta `key` |
| `ERR_BOND_OBJECT_AGENCY_REQUIRED` | 400 | — | `agencyId` vacío |
| `ERR_BOND_OBJECT_INVALID_AMOUNT` | 400 | — | `amount <= 0` |
| `ERR_BOND_OBJECT_INVALID_MATURITY` | 400 | — | `maturityTimeInDays <= 0` o `maturityRewardsPercentage < 0` |
| `ERR_BOND_OBJECT_INVALID_INFLECTION_POINT` | 400 | — | `inflectionPoint < 0` o `inflectionPoint >= maturityTimeInDays` |
| `ERR_BOND_OBJECT_INVALID_WITHDRAW_DAY` | 400 | — | `initialDayForWithdraw < 0` o `> maturityTimeInDays` |
| `ERR_BOND_OBJECT_INVALID_PERCENTAGE` | 400 | — | Penalidades fuera de 0–100 |
| `ERR_BOND_OBJECT_INVALID_SALE_WINDOW` | 400 | — | `saleEndDate < saleInitDate` |
| `ERR_BOND_OBJECT_IDS_REQUIRED` | 400 | — | `bonds` vacío o con más de 500 ids |

## 5. Reglas de negocio

### 5.1 Validaciones de alta y edición

Son nuevas: el legado no validaba nada.
- `name`, `key` y `bondTypeId` no vacíos; `agencyId` con al menos un elemento.
- `bondTypeId` debe existir con `isRemove: false` (`BondTypeApplicationPorts.GetByID`).
- `amount > 0`, `maturityTimeInDays > 0` y `maturityRewardsPercentage >= 0`.
- `0 <= inflectionPoint < maturityTimeInDays`. Si fueran iguales, las fórmulas de las specs 03 y 05 dividirían por cero (`maturityTimeInDays - inflectionPoint`).
- `0 <= initialDayForWithdraw <= maturityTimeInDays`.
- `initialPenaltyPercentage` e `inflectionPenaltyPercentage` entre 0 y 100.
- Si vienen las dos fechas, `saleEndDate >= saleInitDate`.

**Paridad que se mantiene:** el legado aplica `|| 100` a las penalidades, así que un `0` se guarda como `100`. `Normalize()` replica esa regla (ver la pregunta abierta).

### 5.2 Diff de edición (`BuildChanges`, función pura)

```
changes = {}
para cada campo escalar (name, description, key, bondTypeId, amount, isPrivateBond,
                         maturityTimeInDays, maturityRewardsPercentage, active, available):
  si old != new → changes[campo] = {old, new}
para fechas (saleInitDate, saleEndDate): comparar por instante → changes[campo]
para arreglos (agencyId, authorizedUserIds): comparar como conjuntos → changes[campo] = {old, new}
para earlyCashTerm.<sub>: si old != new → changes["earlyCashTerm.<sub>"] = {old, new}
editedProperties = keys(changes) ordenadas
```

Corrige O2: el legado no detectaba cambios en `earlyCashTerm` ni en las fechas.

## 6. Modelo de datos

```go
// shared_domain: lo comparten bondObject (términos del producto) y bondBound (snapshot al comprar).
type EarlyCashTerm struct {
    InitialPenaltyPercentage    float64 `json:"initialPenaltyPercentage" bson:"initialPenaltyPercentage"`
    InflectionPoint             int     `json:"inflectionPoint" bson:"inflectionPoint"`
    InflectionPenaltyPercentage float64 `json:"inflectionPenaltyPercentage" bson:"inflectionPenaltyPercentage"`
    BlockWithdrawals            bool    `json:"blockWithdrawals" bson:"blockWithdrawals"`
    InitialDayForWithdraw       int     `json:"initialDayForWithdraw" bson:"initialDayForWithdraw"`
}

type BondObject struct {
    ID                        string                  `json:"_id" bson:"_id"`
    Name                      string                  `json:"name" bson:"name"`
    Description               string                  `json:"description" bson:"description"`
    AgencyID                  shared_domain.AgencyIDs `json:"agencyId" bson:"agencyId"`
    Key                       string                  `json:"key" bson:"key"`
    BondTypeID                string                  `json:"bondTypeId" bson:"bondTypeId"`
    Amount                    float64                 `json:"amount" bson:"amount"`
    IsPrivateBond             bool                    `json:"isPrivateBond" bson:"isPrivateBond"`
    AuthorizedUserIDs         []string                `json:"authorizedUserIds" bson:"authorizedUserIds"`
    MaturityTimeInDays        int                     `json:"maturityTimeInDays" bson:"maturityTimeInDays"`
    MaturityRewardsPercentage float64                 `json:"maturityRewardsPercentage" bson:"maturityRewardsPercentage"`
    Active                    bool                    `json:"active" bson:"active"`
    Available                 bool                    `json:"available" bson:"available"`
    SaleInitDate              *time.Time              `json:"saleInitDate" bson:"saleInitDate"`
    SaleEndDate               *time.Time              `json:"saleEndDate" bson:"saleEndDate"`
    EarlyCashTerm             shared_domain.EarlyCashTerm `json:"earlyCashTerm" bson:"earlyCashTerm"`
    CreatedAt, UpdatedAt      time.Time               // json/bson: createdAt, updatedAt
    IsRemove                  bool                    `json:"isRemove" bson:"isRemove"`
}
```

- **Datos legacy:** los numéricos pueden venir como string (eran inputs de formulario sin validar). El decoder BSON acepta `string|int|double` en `amount`, `maturityTimeInDays`, `maturityRewardsPercentage` y en los campos de `earlyCashTerm` (tipo `shared_domain.FlexNumber`).
- **`isPrivateBond` legacy:** hay bondObjects que guardan ahí un string, por ejemplo un id de Meteor (O11). Se lee con el criterio de verdad de JS, como lo evalúa el legado (`bond.object.js:574`, `isPrivateBond && …`):
  - `false`, `null`, ausente, `""` y `0` son `false`;
  - cualquier otro valor es `true`, también `"false"` y `"0"`, que en JS son strings no vacíos.

  El filtro de privados de la consulta (RF-02.9) usa el mismo criterio: un bondObject es público solo si `isPrivateBond` está en `[false, null, "", 0]` (`null` también encuentra el campo ausente). Al editar el bondObject, el valor se guarda como booleano.
- **Índices:** `{isRemove: 1, agencyId: 1, bondTypeId: 1, active: 1}`, `{isRemove: 1, agencyId: 1, createdAt: -1}`.

## 7. Ports y dependencias

- `BondObjectApplicationPorts`:
  - `List`, `ListGroupedByType`, `GetByID`, `Create`, `Update`, `UpdateKeys`, `Remove`, `ListInvestors`, `ListByType`, `MostPurchased`, `Latest`.
  - `FindVisibleForPurchase(ctx, id)`, que usa la spec 04.
  - `GetByIDIncludingRemoved(ctx, id)`, que usan las specs 05 y 10 para el nombre del producto en los mensajes.
- `BondObjectMongoccRepositoryPorts`, con un adapter por método, más `EnsureIndexes`. Incluye `CountActiveByType(ctx, bondTypeIDs, userID)` y `ExistsNotRemovedByType(ctx, bondTypeID)`, que cuentan sobre `bondObject` para `BondObjectCounterPort` de la spec 01.
- `BondBoundStatsPort`: adapter de **solo lectura** dentro de `bondObject/infrastructure/mongocc/`, que agrega sobre la colección `bondBound`:
  - `TopPurchasedObjectIDs(ctx, agencyID, limit)`
  - `InvestorsByObject(ctx, objectID, agencyScope)`

  No escribe en `bondBound`.
- `BondTypeApplicationPorts` (spec 01): `GetByID` y `GetManyByIDs`.
- `BondHistoryPort.Add` (spec 08), `ActivityPort` y `ProfileReader` (spec 00).
- Implementa `bondType_ports.BondObjectCounterPort` (spec 01): la aplicación delega en `CountActiveByType` y `ExistsNotRemovedByType` del repositorio.

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| O2 | El diff de `update` no detecta cambios en `earlyCashTerm` ni en fechas. | Corregir | Auditoría incompleta de cambios de términos financieros. | BLM-0053 |
| O3 | Los privados se filtran en el método `getBondObjectsByBondTypeId`, pero no en la publicación ni en `getMostPurchased`/`getLatest`. | Corregir | Exposición de bonos privados a usuarios no autorizados. | BLM-0055 |
| O4 | `massiveUpdateOfKeys` nunca responde, y el cliente queda colgado. | Corregir | Llamada que no responde. | BLM-0054 |
| O5 | `listGroupedByType` hace N+1 consultas a `bondType` y filtra por `data.agencyId` en lugar de la sesión. | Corregir | Rendimiento y scoping. | BLM-0052 |
| O6 | Sin validaciones en alta y edición: `inflectionPoint == maturityTimeInDays` produce división por cero en las fórmulas. | Corregir | Cálculos de dinero inválidos. | BLM-0045 |
| O9 | `get` devuelve los perfiles completos de `authorizedUser`, incluido el hash del password. | Corregir | Seguridad. | BLM-0052 |
| O11 | Algunos bondObjects legacy guardan un string en `isPrivateBond` (en la base local, ids de Meteor como `"A8DjCnAETjXarQ2YA"`). El legado los trata como privados, porque un string no vacío es verdadero. Go fallaba al decodificarlos: cualquier endpoint que los lee respondía 500, incluido el listado de su tipo. Además, su filtro `isPrivateBond: {$ne: true}` los dejaba pasar como públicos. Lo encontró el arnés de paridad (BLM-0125). | Paridad: criterio de verdad de JS (§6) | Visibilidad de productos privados y endpoints que fallan. | BLM-0154 |

## 9. Criterios de aceptación

- **CA-02.1** (RF-02.1) — Dados los bondObjects `"Zeta"` y `"Alfa"` de la agencia A y uno de B, `GET /bond-objects?agencyId=A` devuelve `[Alfa, Zeta]`.
- **CA-02.2** (RF-02.2) — Dados dos tipos con bonos, `GET /bond-objects/grouped-by-type` devuelve 2 grupos, cada uno con su `bondType` completo, y el repositorio ejecuta una sola agregación.
- **CA-02.3** (RF-02.3) — Dado un bondObject con `authorizedUserIds: [u1]`, `GET /bond-objects/{id}` devuelve `authorizedUser[0]` con `first_name` y `email`, y sin `password`.
- **CA-02.4** (RF-02.4) — Un `POST` sin `isPrivateBond` ni `earlyCashTerm.inflectionPenaltyPercentage` guarda `isPrivateBond: false` e `inflectionPenaltyPercentage: 100`, y crea un `bondHistory` (`typeId '1'`, `text 'BOND_ADDED'`) y una `activity` (`description 'BOND_CREATED'`).
- **CA-02.5** (RF-02.4) — Un `bondTypeId` inexistente responde `422 ERR_BOND_OBJECT_INVALID_BOND_TYPE`. `inflectionPoint: 365` con `maturityTimeInDays: 365` responde `400 ERR_BOND_OBJECT_INVALID_INFLECTION_POINT`.
- **CA-02.6** (RF-02.5) — Un `PUT` que cambia `amount` 1000→2000 y `earlyCashTerm.inflectionPoint` 0→30 genera un `bondHistory` `BOND_UPDATED` con `changes` en `amount` y `earlyCashTerm.inflectionPoint`, y `editedProperties` con esas dos claves.
- **CA-02.7** (RF-02.5) — Dado un `bondBound` comprado del bondObject X con `amount 1000`, después de un `PUT` que cambia el `amount` de X a 2000, el `bondBound` sigue con `amount 1000`.
- **CA-02.8** (RF-02.6) — `PATCH /bond-objects/keys` con 3 ids, uno de ellos borrado, responde `{matched: 2, modified: 2}`.
- **CA-02.9** (RF-02.7) — `DELETE` deja `isRemove: true` y crea el history `BOND_REMOVE`. Un segundo `DELETE` responde `404`.
- **CA-02.10** (RF-02.8) — Dado un usuario con bonos del objeto X en estados `1`, `5` y `6`, `GET /bond-objects/X/investors` devuelve `numOfBounds: 2`, sin `password` en `investor`.
- **CA-02.11** (RF-02.9) — Para un usuario de la agencia A no autorizado en el privado P, el listado del tipo excluye P, los bonos inactivos y los de la agencia B, e incluye `maturityDateFormatted`. Si el usuario está en `authorizedUserIds` de P, P aparece.
- **CA-02.12** (RF-02.10) — Dados 4 bondObjects con 5, 3, 8 y 1 compras, `most-purchased` devuelve los de 8, 5 y 3 compras, en ese orden. Si el de 8 es privado y el usuario no está autorizado, no aparece.
- **CA-02.13** (RF-02.11) — `latest` devuelve como máximo 3 bonos, ordenados por `createdAt` desc, solo de la agencia de la sesión.
- **CA-02.14** (RF-02.4, RF-02.5, RF-02.7) — Si `BondHistory.Add` falla, el alta, la edición o la baja responden con éxito igual, y el error queda en el log.
- **CA-02.15** (RF-02.3, RF-02.9) — Dado un bondObject legacy (`_id` de 17 caracteres) cuyo `bondTypeId` también es legacy, `GET /bond-objects/{id}` lo devuelve y `GET /bond-types/{bondTypeId}/bond-objects` lo lista junto a los bondObjects nuevos (UUID) del mismo tipo.
- **CA-02.16** (RF-02.9) — El decoder lee `isPrivateBond` así:
  - `false`, `null`, ausente, `""` y `0` dan `false`;
  - `true`, `"A8DjCnAETjXarQ2YA"`, `"false"` y `1` dan `true`.

  Los demás campos del documento se leen igual que con un booleano.
- **CA-02.17** (RF-02.3, RF-02.9) — Dados dos bondObjects del mismo tipo, P con `isPrivateBond: "A8DjCnAETjXarQ2YA"` y Q con `isPrivateBond: ""`:
  - el detalle de P responde `200` con `isPrivateBond: true`;
  - para un usuario no autorizado, el listado del tipo responde `200` con Q y sin P;
  - si el usuario está en `authorizedUserIds` de P, P aparece.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0045 | Dominio: `BondObject` (con `FlexNumber` para numéricos legacy), DTOs con defaults legacy en `Normalize()` y validaciones de §5.1, errores y `httpStatus` | impl | F3 | RF-02.4, RF-02.5 | BLM-0009, BLM-0039, BLM-0028 | `internal/core/bondObject/domain/*.go` |
| BLM-0046 | Ports: `BondObjectApplicationPorts`, `BondObjectMongoccRepositoryPorts`, `BondBoundStatsPort` | impl | F3 | RF-02.1, RF-02.2, RF-02.3, RF-02.4, RF-02.5, RF-02.6, RF-02.7, RF-02.8, RF-02.9, RF-02.10, RF-02.11 | BLM-0045 | `internal/core/bondObject/ports/bondObject.ports.go`, `internal/core/bondObject/domain/bondObject.dto.domain.go` (tipos de salida y `VisibleFilter`) |
| BLM-0047 | Repositorio mongocc (`find`, agregación agrupada con `$lookup`, filtro de privados, `updateMany` de claves, `softDelete`, conteos por tipo) y `EnsureIndexes` | impl | F3 | RF-02.1, RF-02.2, RF-02.6, RF-02.7, RF-02.9, RF-02.11, RF-01.2, RF-01.6 | BLM-0046 | `internal/core/bondObject/infrastructure/mongocc/bondObject.mongocc.repository.go`, `*.adapter.go` |
| BLM-0048 | Adapter de solo lectura `BondBoundStatsPort` e implementación de `BondObjectCounterPort` para la spec 01 | impl | F3 | RF-02.8, RF-02.10, RF-01.2, RF-01.6 | BLM-0047 | `internal/core/bondObject/infrastructure/mongocc/bondObject.bondBoundStats.adapter.go`, `internal/core/bondObject/application/bondObject.application.go`, `bondObject.counter.go` |
| BLM-0052 | Casos de uso admin de consulta: `List`, `ListGroupedByType` (sin N+1), `GetByID` con `authorizedUser` seguro y `GetByIDIncludingRemoved` | impl | F3 | RF-02.1, RF-02.2, RF-02.3 | BLM-0047, BLM-0017 | `internal/core/bondObject/application/bondObject.list.go`, `bondObject.listGroupedByType.go`, `bondObject.getById.go`, `bondObject.getByIdIncludingRemoved.go`, `bondObject.application.go` (dependencia `ProfileReader`), `cmd/api/server/server.bondObject.go` (cableado del `ProfileReader`) |
| BLM-0053 | Función pura `BuildChanges(old, new)` con escalares, fechas, arreglos y `earlyCashTerm.*` | impl | F3 | RF-02.5 | BLM-0045 | `internal/core/bondObject/domain/bondObject.changes.go` |
| BLM-0054 | Casos de uso `Create`, `Update`, `UpdateKeys` y `Remove`, con history y activity best-effort | impl | F3 | RF-02.4, RF-02.5, RF-02.6, RF-02.7 | BLM-0047, BLM-0053, BLM-0035, BLM-0013, BLM-0042 | `internal/core/bondObject/application/bondObject.create.go`, `bondObject.update.go`, `bondObject.updateKeys.go`, `bondObject.remove.go`, `bondObject.history.go`, `bondObject.application.go` (dependencias), `bondObject.getById.go` (`findNotRemoved`), `cmd/api/server/server.bondObject.go`, `server.deps.go`, `server.config.go` (cableado de `ActivityPort`) |
| BLM-0055 | Casos de uso `ListInvestors`, `ListByType` (privados y `maturityDateFormatted`), `MostPurchased`, `Latest` y `FindVisibleForPurchase` | impl | F3 | RF-02.8, RF-02.9, RF-02.10, RF-02.11 | BLM-0048 | `internal/core/bondObject/application/bondObject.listInvestors.go`, `bondObject.listByType.go`, `bondObject.mostPurchased.go`, `bondObject.latest.go`, `bondObject.findVisibleForPurchase.go`, `bondObject.application.go` (dependencia `BondBoundStatsPort` y aserción del port), `internal/core/bondObject/domain/bondObject.maturity.go`, `cmd/api/server/server.bondObject.go` (cableado) |
| BLM-0056 | Handlers, registro de rutas y dependencias de `server.bondObject.go` | impl | F3 | RF-02.1, RF-02.2, RF-02.3, RF-02.4, RF-02.5, RF-02.6, RF-02.7, RF-02.8, RF-02.9, RF-02.10, RF-02.11 | BLM-0052, BLM-0054, BLM-0055, BLM-0007, BLM-0026 | `cmd/api/handlers/bondObjects/*.go`, `cmd/api/server/server.bondObject.go`, `server.go` (registro de handlers) |
| BLM-0057 | Tests unitarios de casos de uso y handlers con fakes | test | F3 | CA-02.3, CA-02.4, CA-02.5, CA-02.6, CA-02.9, CA-02.14 | BLM-0056 | `internal/core/bondObject/application/*_test.go`, `cmd/api/handlers/bondObjects/*_test.go` |
| BLM-0058 | Tests de integración del repositorio y de las agregaciones | test | F3 | CA-01.3, CA-02.1, CA-02.2, CA-02.7, CA-02.8, CA-02.10, CA-02.11, CA-02.12, CA-02.13, CA-02.15 | BLM-0047, BLM-0048 | `internal/core/bondObject/infrastructure/mongocc/*_integration_test.go` |
| BLM-0059 | Test de tabla de `BuildChanges` | test | F3 | CA-02.6 | BLM-0053 | `internal/core/bondObject/domain/bondObject.changes_test.go` |
| BLM-0154 | `isPrivateBond` legacy con el criterio de verdad de JS: el decoder de `bondObject` acepta cualquier tipo y el filtro de privados de la consulta trata como público solo `[false, null, "", 0]` | impl | F3 | RF-02.3, RF-02.9 | BLM-0045, BLM-0047 | `internal/core/bondObject/domain/bondObject.bson.go`, `internal/core/bondObject/infrastructure/mongocc/bondObject.mongocc.repository.go` (`visibleTo`) |
| BLM-0155 | Tests del decoder con cada valor de `isPrivateBond` y de integración del detalle y del listado del tipo con un `isPrivateBond` string | test | F3 | CA-02.16, CA-02.17 | BLM-0154 | `internal/core/bondObject/domain/bondObject.bson_test.go`, `internal/core/bondObject/infrastructure/mongocc/bondObject.findVisible_integration_test.go` |

## 11. Fuera de alcance y preguntas abiertas

- **Pregunta:** ¿la regla legacy `|| 100`, que convierte una penalidad 0% en 100%, es intencional? Por ahora se mantiene.
- **Pregunta:** ¿`latest` y `most-purchased` deberían ocultar los bonos inactivos o fuera de la ventana de venta? El legado no lo hace.
- **Pregunta:** ¿qué idioma usa `maturityDateFormatted`? El legado usa el locale de moment del servidor; por ahora se asume inglés (`Oct`).
- La historia de `massiveUpdateOfKeys` (`BOND_UPDATED_KEY`) está comentada en el legado y no se registra.
