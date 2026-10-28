# Spec 01 — Tipos de bono (`bondType`)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondType` |
| Handlers | `cmd/api/handlers/bondTypes` |
| Colección | `bondType` |
| Depende de | Spec 00; spec 02 (`BondObjectCounterPort`) |

## 1. Resumen y alcance

Catálogo de tipos de bono: lo administra el admin de Blooming y lo consulta el inversionista para navegar los bonos. Cubre CRUD con soft delete, el listado de administración y el listado de tipos activos de la agencia con su cantidad de bonos.

## 2. Origen legacy

Archivo: `backend/server/model/bond.type.js` (130 líneas).

| Operación legacy | Línea | Tipo | Destino |
|---|---|---|---|
| `bondTypes` | 18 | publicación | Fusionada en `GET /bond-types/active` (RF-01.2) |
| `bondType.list` | 31 | método | `GET /bond-types` (RF-01.1) |
| `bondType.get` | 45 | método | `GET /bond-types/{id}` (RF-01.3) |
| `bondType.insert` | 54 | método | `POST /bond-types` (RF-01.4) |
| `bondType.update` | 71 | método | `PUT /bond-types/{id}` (RF-01.5) |
| `bondType.remove` | 93 | método | `DELETE /bond-types/{id}` (RF-01.6) |
| `bondType.listByActive` | 121 | método | `GET /bond-types/active` (RF-01.2) |
| `bondTypeExtraInfo` | 113 | helper | `numOfBonds` en RF-01.2 |

## 3. Requisitos funcionales

- **RF-01.1** — Listado de administración.
  - Devuelve los tipos con `isRemove: false` más `AgencyScope(ctx)`. Si la agencia de la sesión es la principal, el admin **puede** filtrar por uno o más `agencyId` en el query: se quedan los tipos cuyo `agencyId` contiene **alguno** de ellos (`$in`). Si no lo es, el query no amplía el filtro.
  - Orden: `name` ascendente.
  - Solo admin.
- **RF-01.2** — Listado de tipos activos para el inversionista.
  - Devuelve los tipos con `isRemove: false`, `active: true` y `agencyId` que contiene el `agencyId` de la sesión.
  - Cada tipo incluye `numOfBonds`: la cantidad de `bondObject` con `isRemove: false`, `active: true` y ese `bondTypeId`, **sin filtrar por agencia** (paridad con `bondType: true` del legado). Los privados solo cuentan si el usuario está en `authorizedUserIds`.
  - Se calcula en una sola agregación (`$group` por `bondTypeId`), no con una consulta por tipo.
- **RF-01.3** — Detalle. Devuelve el tipo por `_id` si `isRemove: false`; si no, `ERR_BOND_TYPE_NOT_FOUND`. Solo admin.
- **RF-01.4** — Alta. Crea un tipo con `name`, `description`, `active` y `agencyId []string`.
  - Validaciones: `name` no vacío (`ERR_BOND_TYPE_NAME_REQUIRED`) y al menos un `agencyId` (`ERR_BOND_TYPE_AGENCY_REQUIRED`).
  - `active` es `false` por defecto.
  - Responde el documento creado. Solo admin.
- **RF-01.5** — Edición. Reemplaza `name`, `description`, `active` y `agencyId` del tipo, con las mismas validaciones del alta, y actualiza `updatedAt`.
  - Si no existe o está borrado, `ERR_BOND_TYPE_NOT_FOUND`.
  - Responde el documento actualizado. Solo admin.
- **RF-01.6** — Baja lógica.
  - Si existe algún `bondObject` con `bondTypeId` igual y `isRemove: false`, responde `ERR_BOND_TYPE_HAS_BOND_OBJECTS`. Corrige T1: los bondObjects ya borrados no bloquean.
  - Si no, marca `isRemove: true`. Solo admin.
- RF-01.1 y RF-01.3 a RF-01.6 aplican el alcance de agencia de la sesión (constitución §6; corrige T5). La agencia principal no tiene restricción. Para un admin de otra agencia:
  - un tipo cuyo `agencyId` no contiene su agencia no existe: no aparece en el listado, y el detalle, la edición y la baja responden `ERR_BOND_TYPE_NOT_FOUND`;
  - el alta y la edición exigen `agencyId` igual a `[agencia de la sesión]`, y la edición y la baja, un tipo solo de su agencia (todos sus `agencyId` son la de la sesión). Si no, `ERR_FORBIDDEN`: un tipo compartido con otras agencias lo edita o borra solo la agencia principal;
  - los errores se evalúan en este orden: validación (`400`), alcance (`404`), permiso (`403`) y conflicto (`409`).
- **RF-01.7** — Compatibilidad del campo `agencyId`. En los documentos legacy puede ser un `string` o un `[]string`.
  - El tipo de dominio `shared_domain.AgencyIDs` decodifica los dos formatos como `[]string`, y siempre persiste `[]string`.
  - Los filtros usan `{agencyId: X}` o `{agencyId: {$in: [...]}}`, que sirven para los dos formatos.

## 4. Contrato API

Prefijo `/api/v1`. Todos requieren `x-secret` y `x-token`.

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| GET | `/bond-types` | admin | RF-01.1 |
| GET | `/bond-types/active` | usuario | RF-01.2 |
| GET | `/bond-types/{id}` | admin | RF-01.3 |
| POST | `/bond-types` | admin | RF-01.4 |
| PUT | `/bond-types/{id}` | admin | RF-01.5 |
| DELETE | `/bond-types/{id}` | admin | RF-01.6 |

Un admin de una agencia que no es la principal opera solo dentro de su agencia (§3).

**`GET /bond-types?agencyId=A&agencyId=B`** → `200`

```json
[{ "_id": "d0c1…", "name": "Bono Plus", "description": "…", "active": true, "agencyId": ["A"], "createdAt": "2026-01-10T15:00:00Z", "updatedAt": "…", "isRemove": false }]
```

**`GET /bond-types/active`** → `200`. Mismo objeto que el anterior, más `"numOfBonds": 4`.

**`POST /bond-types`**

```json
{ "name": "Bono Plus", "description": "Renta fija 12 meses", "active": true, "agencyId": ["A", "B"] }
```

→ `201` con el documento creado. **`PUT /bond-types/{id}`** recibe el mismo body y responde `200` con el documento. **`DELETE /bond-types/{id}`** → `200 true`.

**Cambio de contrato respecto al legado:**
- `listByActive` devolvía `{success, code, data}`; ahora se devuelve el arreglo directo.
- `insert` devolvía solo el id; ahora devuelve el documento.
- `update` y `remove` devolvían `1`.
- Un admin de una agencia que no es la principal solo ve, edita y borra los tipos de su agencia, y solo crea tipos para ella (T5); el `agencyId` del listado solo filtra para la agencia principal.

**Errores**

| Sentinel | HTTP | Código legacy | Cuándo |
|---|---|---|---|
| `ERR_INVALID_ID` | 400 | — | Un id de path o body no es UUID v4 ni Meteor de 17 caracteres (spec 00, RF-00.19) |
| `ERR_BOND_TYPE_NOT_FOUND` | 404 | -11 | No existe, `isRemove: true` o fuera del alcance de agencia de la sesión |
| `ERR_BOND_TYPE_NAME_REQUIRED` | 400 | — | `name` vacío |
| `ERR_BOND_TYPE_AGENCY_REQUIRED` | 400 | — | `agencyId` vacío |
| `ERR_BOND_TYPE_HAS_BOND_OBJECTS` | 409 | -13 | Baja con bonos activos asignados |
| `ERR_FORBIDDEN` | 403 | — | No es admin (spec 00), o un admin de una agencia que no es la principal escribe fuera de su agencia |

## 5. Reglas de negocio

```
numOfBonds(type, session) =
  count(bondObject {
    isRemove: false, active: true, bondTypeId: type._id,
    $or: [ {isPrivateBond: {$ne: true}}, {authorizedUserIds: session.userId} ]
  })
```

## 6. Modelo de datos

```go
type BondType struct {
    ID          string                 `json:"_id" bson:"_id"`
    AgencyID    shared_domain.AgencyIDs `json:"agencyId" bson:"agencyId"`
    Name        string                 `json:"name" bson:"name"`
    Description string                 `json:"description" bson:"description"`
    Active      bool                   `json:"active" bson:"active"`
    CreatedAt   time.Time              `json:"createdAt" bson:"createdAt"`
    UpdatedAt   time.Time              `json:"updatedAt" bson:"updatedAt"`
    IsRemove    bool                   `json:"isRemove" bson:"isRemove"`
}
```

Índices (`EnsureIndexes`): `{isRemove: 1, active: 1, agencyId: 1}`.

## 7. Ports y dependencias

- `BondTypeApplicationPorts`: `List`, `ListActive`, `GetByID`, `Create`, `Update`, `Remove`. La spec 02 también usa `GetByID` y `GetManyByIDs`, para validar `bondTypeId` y armar el agrupado. `GetByID` aplica el alcance de agencia de la sesión (§3), y así la spec 02 valida que el `bondTypeId` esté en el alcance. `GetManyByIDs` no lo aplica: lo usa también la maduración (spec 10), sin sesión.
- `BondTypeMongoccRepositoryPorts`: `Find`, `FindOneByID`, `FindManyByIDs`, `InsertOne`, `UpdateOne`, `SoftDelete`, `EnsureIndexes`.
- `BondObjectCounterPort` se declara en `bondType_ports` y lo implementa la aplicación de `bondObject`:
  - `CountActiveByType(ctx, bondTypeIDs []string, userID string) (map[string]int, error)`
  - `ExistsNotRemovedByType(ctx, bondTypeID string) (bool, error)`

  Se conecta **después** de inicializar los dos dominios con `linkBondObjectToBondType(deps)` en `cmd/api/server`, igual que `linkBillingAccountToClosureRequest` en petty-cash. Así se evita el ciclo de inicialización: `bondObject` necesita `bondType` y viceversa.

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| T1 | `remove` cuenta los bondObjects borrados como bloqueantes. | Corregir | Un tipo sin bonos vivos no se podía borrar nunca. | BLM-0049 |
| T2 | `numOfBonds` cuenta bondObjects de todas las agencias, pero el listado del inversionista filtra por agencia. | Mantener | Paridad visual; queda como pregunta abierta. | — |
| T3 | `get` no filtra `isRemove` y devuelve `undefined` si no existe. | Corregir | Respuesta 404 explícita. | BLM-0042 |
| T4 | `update` escribe `undefined` (`null`) en los campos que no llegan. | Corregir | La validación exige el body completo. | BLM-0043 |
| O1 | `agencyId` se filtra con `$all` en tipos y con `$in: [agencyId]` en objetos (arreglo dentro de arreglo). | Corregir | Filtro único `$in` con `AgencyIDs` (RF-01.7). | BLM-0039 |
| T5 | El legado no valida la agencia en ningún método de administración de bondType, y la migración lo mantuvo: un admin de una agencia que no es la principal lista (`GET /bond-types` solo filtra por el `agencyId` del query), lee, edita y borra tipos de otras agencias, y crea o edita tipos para otras agencias. Contradice la constitución §6. Lo encontró la revisión de rutas de la matriz de autorización (BLM-0127). | Corregir | Aislamiento entre agencias. | BLM-0158 |

## 9. Criterios de aceptación

- **CA-01.1** (RF-01.1) — Dados los tipos `T1{agencyId:[A]}`, `T2{agencyId:[B]}` y `T3{agencyId:[A], isRemove:true}`, cuando un admin de la agencia principal pide `GET /bond-types?agencyId=A`, entonces recibe solo `T1`.
- **CA-01.2** (RF-01.1, RF-01.3, RF-01.4) — Dada una sesión sin `adminBlooming`, cuando llama a `GET /bond-types`, `GET /bond-types/{id}` o `POST /bond-types`, entonces recibe `403 ERR_FORBIDDEN`.
- **CA-01.3** (RF-01.2) — Dada una sesión de la agencia A y el tipo `T1` con tres bondObjects activos (uno privado sin el usuario autorizado) y uno inactivo, cuando pide `GET /bond-types/active`, entonces `T1.numOfBonds == 2`.
- **CA-01.4** (RF-01.3) — Dado un tipo con `isRemove: true`, `GET /bond-types/{id}` responde `404 ERR_BOND_TYPE_NOT_FOUND`.
- **CA-01.5** (RF-01.4) — Un `POST` válido responde `201` con un `_id` UUID, `isRemove: false` y `createdAt` en UTC. Sin `name` responde `400 ERR_BOND_TYPE_NAME_REQUIRED`.
- **CA-01.6** (RF-01.5) — Un `PUT` sobre un tipo existente actualiza los cuatro campos y `updatedAt`. Sobre un id inexistente responde `404`.
- **CA-01.7** (RF-01.6) — Si el tipo tiene un bondObject con `isRemove: false`, `DELETE` responde `409 ERR_BOND_TYPE_HAS_BOND_OBJECTS`. Si solo tiene bondObjects borrados, responde `200` y el tipo queda con `isRemove: true`.
- **CA-01.8** (RF-01.7) — Un documento legacy con `agencyId: "A"` (string) se devuelve como `["A"]` y aparece en el filtro `agencyId=A`.
- **CA-01.9** (RF-01.3, RF-01.5, RF-01.6) — Dado un tipo legacy con `_id` Meteor de 17 caracteres (`kQ3xT9mPz2LwA7bNc`), `GET`, `PUT` y `DELETE /bond-types/kQ3xT9mPz2LwA7bNc` funcionan igual que con un tipo nuevo; después del `PUT` el documento conserva su `_id`. Un tipo nuevo creado a continuación tiene `_id` UUID.
- **CA-01.10** (RF-01.1, RF-01.3) — Dados `T1{agencyId:[A]}`, `T2{agencyId:[B]}` y `T4{agencyId:[A, B]}`, para un admin de A que no es la principal:
  - `GET /bond-types` y `GET /bond-types?agencyId=B` devuelven `T1` y `T4`;
  - `GET /bond-types/{T2}` responde `404 ERR_BOND_TYPE_NOT_FOUND`, y `GET /bond-types/{T4}`, `200`.

  Un admin de la agencia principal recibe los tres en el listado y `200` en los tres detalles.
- **CA-01.11** (RF-01.4, RF-01.5, RF-01.6) — Con los tipos de CA-01.10, para un admin de A que no es la principal:
  - `POST` con `agencyId: [A, B]` o `[B]` responde `403 ERR_FORBIDDEN`, y con `[A]`, `201`; sin `name` responde `400` aunque el `agencyId` sea `[B]`;
  - `PUT` y `DELETE` sobre `T2` responden `404 ERR_BOND_TYPE_NOT_FOUND`, y sobre `T4`, `403 ERR_FORBIDDEN`, también si `T4` tiene bondObjects vivos;
  - `PUT` sobre `T1` con `agencyId: [A, B]` responde `403`, y con `[A]`, `200`.

  Ninguna respuesta `4xx` escribe nada. Un admin de la agencia principal crea, edita y borra los tres.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0039 | Dominio: entidad `BondType`, tipo `AgencyIDs` (string o arreglo), DTOs con `Normalize`/`Validation`, errores y `httpStatus` | impl | F3 | RF-01.4, RF-01.5, RF-01.7 | BLM-0009 | `internal/core/bondType/domain/*.go`, `internal/core/shared/domain/shared.agencyIds.go` |
| BLM-0040 | Ports: `BondTypeApplicationPorts`, `BondTypeMongoccRepositoryPorts`, `BondObjectCounterPort` | impl | F3 | RF-01.1, RF-01.2, RF-01.3, RF-01.4, RF-01.5, RF-01.6 | BLM-0039 | `internal/core/bondType/ports/bondType.ports.go` |
| BLM-0041 | Repositorio mongocc (un archivo por método) y `EnsureIndexes` | impl | F3 | RF-01.1, RF-01.2, RF-01.3, RF-01.5, RF-01.6 | BLM-0040 | `internal/core/bondType/infrastructure/mongocc/bondType.mongocc.repository.go`, `*.adapter.go` |
| BLM-0042 | Casos de uso `List`, `ListActive` (con `numOfBonds`), `GetByID` y `GetManyByIDs` | impl | F3 | RF-01.1, RF-01.2, RF-01.3 | BLM-0041 | `internal/core/bondType/application/bondType.application.go`, `bondType.list.go`, `bondType.listActive.go`, `bondType.getById.go`, `bondType.getManyByIds.go` |
| BLM-0043 | Casos de uso `Create` y `Update` | impl | F3 | RF-01.4, RF-01.5 | BLM-0041 | `internal/core/bondType/application/bondType.create.go`, `bondType.update.go` |
| BLM-0044 | Tests de integración del repositorio (filtros por agencia, `AgencyIDs` legacy e ids Meteor) | test | F3 | CA-01.1, CA-01.8, CA-01.9 | BLM-0041 | `internal/core/bondType/infrastructure/mongocc/*_integration_test.go` |
| BLM-0049 | Caso de uso `Remove` con guarda de bondObjects vivos y conexión `linkBondObjectToBondType` | impl | F3 | RF-01.6 | BLM-0041, BLM-0042, BLM-0043, BLM-0048 | `internal/core/bondType/application/bondType.remove.go`, `bondType.application.go` (aserción de `BondTypeApplicationPorts`), `cmd/api/server/server.bondType.go`, `server.bondObject.go` (inicialización mínima), `server.deps.go`, `server.go` |
| BLM-0050 | Handlers y registro de rutas (con `RequireBloomingAdmin` donde corresponde) | impl | F3 | RF-01.1, RF-01.2, RF-01.3, RF-01.4, RF-01.5, RF-01.6 | BLM-0042, BLM-0043, BLM-0049, BLM-0007, BLM-0026 | `cmd/api/handlers/bondTypes/*.go`, `cmd/api/server/server.bondType.go`, `server.go` (registro), `cmd/api/handlers/shared/handlers.body.go` (`DecodeBody`) |
| BLM-0051 | Tests unitarios de casos de uso y handlers con fakes | test | F3 | CA-01.1, CA-01.2, CA-01.3, CA-01.4, CA-01.5, CA-01.6, CA-01.7 | BLM-0050 | `internal/core/bondType/application/*_test.go`, `cmd/api/handlers/bondTypes/*_test.go` |
| BLM-0158 | Alcance de agencia en bondType: el listado filtra por `AgencyScope` en lugar del `agencyId` del query, el detalle fuera del alcance es `ERR_BOND_TYPE_NOT_FOUND`, y el alta, la edición y la baja de un admin de otra agencia exigen tipos solo de su agencia | impl | F3 | RF-01.1, RF-01.3, RF-01.4, RF-01.5, RF-01.6 | BLM-0156, BLM-0042, BLM-0043, BLM-0049 | `internal/core/bondType/application/bondType.list.go`, `bondType.getById.go`, `bondType.create.go`, `bondType.update.go`, `bondType.remove.go`, `internal/core/bondType/ports/bondType.ports.go` (comentarios) |
| BLM-0159 | Tests de alcance de agencia de bondType (admin de otra agencia, de la principal y tipos compartidos) y ajuste de los tests que usaban un context sin sesión | test | F3 | CA-01.1, CA-01.10, CA-01.11 | BLM-0158 | `internal/core/bondType/application/*_test.go` |

## 11. Fuera de alcance y preguntas abiertas

- **Pregunta (T2):** ¿`numOfBonds` debería filtrar por la agencia del usuario, para coincidir con el listado de bonos de la spec 02 (RF-02.9)?
- Por ahora no se valida que el `name` sea único por agencia, como en el legado.
