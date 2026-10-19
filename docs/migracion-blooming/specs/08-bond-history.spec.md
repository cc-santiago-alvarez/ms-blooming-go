# Spec 08 — Historial (`bondHistory`)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondHistory` |
| Handlers | En `cmd/api/handlers/bondObjects` y `cmd/api/handlers/bondBounds` (subrecurso `history`) |
| Colección | `bondHistory` |
| Depende de | Spec 00 |
| La usan | Specs 02, 04 y 05 (escritura con `Add`) |

## 1. Resumen y alcance

Bitácora de eventos sobre productos (`typeId '1'` BOND) y sobre bonos comprados (`typeId '2'` BOND_BOUND). Registra alta, edición con diff, baja, venta de unidades, compra y cambio de estado.

Expone:
- un port de escritura (`Add`), que usan los demás dominios;
- dos listados de lectura con el formato de presentación que espera el front.

**El formato persistido es un contrato externo:** el gateway lo lee directo en `B_BondBound_Audit_details`.

## 2. Origen legacy

Archivo: `backend/server/model/bond.history.js` (125 líneas).

| Operación legacy | Línea | Tipo | Destino |
|---|---|---|---|
| `bondHistories` | 67 | publicación | **Descartada**: filtra por `bondId`, un campo que no existe, y nunca devolvió datos (H1). La reemplazan RF-08.2 y RF-08.3 |
| `bondHistory.listByType` | 94 | método | `GET /bond-objects/{id}/history` y `GET /bond-bounds/{id}/history` (RF-08.2) |
| `Model.bondHistory.add` | 111 | función interna | `BondHistoryPort.Add` (RF-08.1) |
| `bondHistoryFunTemp` | 7 | helper | Presentación (RF-08.4) |
| `Model.bondHistory.type` | 89 | enum | `HistoryType` (§6) |

## 3. Requisitos funcionales

- **RF-08.1** — Registro (`BondHistoryPort.Add(ctx, AddParams)`).
  - Inserta `{_id, relatedId, userId, typeId, agencyId, text, params, changes, editedProperties, createdAt, updatedAt, isRemove: false}`.
  - `userId` y `agencyId` salen de la sesión del `ctx`. En procesos sin sesión (cron, spec 10) se guarda `userId: ""` y el `agencyId` del bono, y la presentación muestra "System".
  - `params` vale `[]` por defecto.
  - Devuelve un error. Cada llamador decide: dentro de una transacción el error aborta (specs 04 y 05); fuera de ella se loguea (spec 02).
- **RF-08.2** — Historial de un producto (admin). `GET /bond-objects/{id}/history` devuelve los registros con `{relatedId: id, typeId: '1', isRemove: false}` y la `agencyId` de la sesión (paridad), ordenados por `createdAt` desc, en el formato de RF-08.4.
- **RF-08.3** — Historial de un bono comprado (dueño o admin). `GET /bond-bounds/{id}/history` primero valida que el bono exista y pertenezca al usuario o que sea admin (spec 06, RF-06.6). Después devuelve `{relatedId: id, typeId: '2', isRemove: false}` con la agencia de la sesión, ordenado por `createdAt` desc, en el formato de RF-08.4.
- **RF-08.4** — Presentación de cada registro (paridad con `bondHistoryFunTemp`):
  - `type = {_id, name}` del enum;
  - `user = {first_name, last_name}` del perfil. Los perfiles se leen **en lote** con `ProfileReader.GetPublic` (corrige H3). Si no existe, `{name: "System"}`;
  - `icon: "mdi-briefcase"`, `color: "#000000"`, `backgroundColor: "#e5e5e5"`;
  - `text = Format(Text[text] o "", params)`. Para `BOND_BOUND_STATUS_CHANGED`, los ids de estado de `params` se traducen a nombres **solo para mostrar**;
  - `createdAtFromNow`: tiempo relativo con el formato de moment en inglés (`"a few seconds ago"`, `"2 days ago"`, `"a month ago"`). Es paridad, porque el backend no configura locale;
  - se omiten `params` y `userId`, y se conservan `typeId`, `relatedId`, `agencyId`, `changes`, `editedProperties` y `createdAt`.
- **RF-08.5** — Textos faltantes (H2). Se agregan al catálogo de la spec 00:
  - `BOND_SOLD = "Bono adquirido por un inversionista ({0})"`;
  - `BOND_BOUND_STATUS_CHANGED = "Estado cambiado de {0} a {1}"`.

  Los demás textos (`BOND_ADDED`, `BOND_UPDATED`, `BOND_REMOVE`, `BOND_BOUND_SOLD`) ya existen en el legado.
- **RF-08.6** — Estabilidad del formato persistido.
  - Los valores de `text` se guardan como **clave** (`"BOND_BOUND_STATUS_CHANGED"`), no como texto ya formateado.
  - `params` de un cambio de estado tiene exactamente dos elementos: `[estadoAnterior, estadoNuevo]`. `Add` rechaza cualquier otro largo con `ERR_BOND_HISTORY_INVALID_PARAMS`.
  - `typeId` es `'1'` o `'2'` como string.

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| GET | `/bond-objects/{id}/history` | admin | RF-08.2, RF-08.4 |
| GET | `/bond-bounds/{id}/history` | dueño o admin | RF-08.3, RF-08.4 |

→ `200`

```json
[ { "_id": "…", "relatedId": "…", "typeId": "2", "type": { "_id": "2", "name": "Bono Enlazado" },
    "agencyId": "A", "text": "Estado cambiado de Iniciado a Vendido",
    "user": { "first_name": "Juan", "last_name": "Pérez" },
    "icon": "mdi-briefcase", "color": "#000000", "backgroundColor": "#e5e5e5",
    "createdAt": "2026-10-12T15:00:00Z", "createdAtFromNow": "a day ago" } ]
```

**Cambio de contrato respecto al legado:** `listByType` recibía `{type: 'BOND'|'BOND_BOUND', relatedId}`. Ahora el tipo lo define la ruta.

**Errores:** `ERR_INVALID_ID` (400, id fuera de formato; spec 00, RF-00.19), `ERR_FORBIDDEN` (403), `ERR_BOND_BOUND_NOT_FOUND` (404, solo en RF-08.3) y `ERR_BOND_HISTORY_INVALID_PARAMS` (500 interno, si un llamador manda un `typeId` distinto de `'1'`/`'2'`, `text` o `relatedId` vacíos, o un `BOND_BOUND_STATUS_CHANGED` sin exactamente dos `params`).

## 5. Reglas de negocio

Ver RF-08.4 y RF-08.6. `FromNow(t, now)` replica los umbrales de moment:

| Diferencia | Salida |
|---|---|
| < 45 s | `a few seconds ago` |
| < 90 s | `a minute ago` |
| < 45 min | `N minutes ago` |
| < 90 min | `an hour ago` |
| < 22 h | `N hours ago` |
| < 36 h | `a day ago` |
| < 26 d | `N days ago` |
| < 45 d | `a month ago` |
| < 320 d | `N months ago` |
| < 548 d | `a year ago` |
| resto | `N years ago` |

## 6. Modelo de datos

```go
type HistoryType string
const (
    HistoryBond      HistoryType = "1" // Bono
    HistoryBondBound HistoryType = "2" // Bono Enlazado
)

type BondHistory struct {
    ID               string                    `json:"_id" bson:"_id"`
    RelatedID        string                    `json:"relatedId" bson:"relatedId"`
    UserID           string                    `json:"-" bson:"userId"`
    TypeID           HistoryType               `json:"typeId" bson:"typeId"`
    AgencyID         string                    `json:"agencyId" bson:"agencyId"`
    Text             string                    `json:"text" bson:"text"`
    Params           []string                  `json:"-" bson:"params"`
    Changes          map[string]ChangeEntry    `json:"changes,omitempty" bson:"changes,omitempty"`
    EditedProperties []string                  `json:"editedProperties,omitempty" bson:"editedProperties,omitempty"`
    CreatedAt, UpdatedAt time.Time             // createdAt, updatedAt
    IsRemove         bool                      `json:"isRemove" bson:"isRemove"`
}
type ChangeEntry struct { Old any `json:"old" bson:"old"`; New any `json:"new" bson:"new"` }
```

Índices: `{relatedId: 1, typeId: 1, agencyId: 1, isRemove: 1, createdAt: -1}` y `{text: 1}` (para la auditoría del gateway).

## 7. Ports y dependencias

- `bondHistory_ports.BondHistoryPort`: `Add(ctx, AddParams) error`. Es la única dependencia que importan las specs 02, 04, 05 y 10.
- `BondHistoryApplicationPorts`: `ListByRelated(ctx, typeID, relatedID)`, que devuelve `[]HistoryEntry` con el formato de RF-08.4.
- `BondHistoryMongoccRepositoryPorts`: `InsertOne`, `FindByRelated`, `EnsureIndexes`.
- `ProfileReader` y catálogo `Text`/`Format` (spec 00).
- RF-08.3 lo orquesta la aplicación de `bondBound` (control de dueño) llamando a `BondHistoryApplicationPorts.ListByRelated`. Así `bondHistory` no depende de `bondBound`.

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| H1 | La publicación `bondHistories` filtra por `bondId`, que no existe, y nunca devuelve datos. | Corregir (se descarta la publicación y se usan los endpoints nuevos) | Funcionalidad rota. | BLM-0063 |
| H2 | `BOND_SOLD` y `BOND_BOUND_STATUS_CHANGED` no existen en `Text` y se muestran vacíos. | Corregir | Historial ilegible. | BLM-0036 |
| H3 | Una consulta de `profile` por registro (N+1). | Corregir | Rendimiento. | BLM-0037 |
| H4 | `listByType` con un `type` desconocido lanza `TypeError`. | Corregir | Llamada que no responde; ahora el tipo lo define la ruta. | BLM-0063 |
| H5 | `listByType` no valida permisos: cualquier sesión lee el historial de cualquier producto o bono de su agencia. | Corregir | Seguridad. | BLM-0063, BLM-0099 |

## 9. Criterios de aceptación

- **CA-08.1** (RF-08.1, RF-08.6) — `Add({relatedId: b1, typeId: '2', text: 'BOND_BOUND_STATUS_CHANGED', params: ['1','5']})` persiste exactamente esos valores, más `userId` y `agencyId` de la sesión, `_id` UUID, timestamps e `isRemove: false`. Con `params: nil` guarda `[]`.
- **CA-08.2** (RF-08.1) — Un `Add` hecho dentro de una transacción que después se revierte no deja documento.
- **CA-08.3** (RF-08.2) — `GET /bond-objects/X/history` (admin) devuelve solo registros `typeId '1'` de X en la agencia de la sesión, ordenados por `createdAt` desc. Un no admin recibe `403`.
- **CA-08.4** (RF-08.3) — `GET /bond-bounds/B/history` devuelve los registros `typeId '2'` de B si el usuario es el dueño. Otro usuario no admin recibe `403`, y un id inexistente da `404`.
- **CA-08.5** (RF-08.4) — Un registro con `userId` de un perfil existente devuelve `user: {first_name, last_name}`. Con `userId ""` devuelve `{name: "System"}`. La salida no contiene `params` ni `userId`.
- **CA-08.6** (RF-08.4, RF-08.5) — `BOND_BOUND_STATUS_CHANGED` con `['1','5']` se muestra como `"Estado cambiado de Iniciado a Vendido"`, `BOND_ADDED` como `"Bono Añadido"`, y una clave desconocida como `""`.
- **CA-08.7** (RF-08.4) — `FromNow` con una diferencia de 30 s da `"a few seconds ago"`, con 2 días `"2 days ago"` y con 400 días `"a year ago"`.
- **CA-08.8** (RF-08.4) — Un historial de 50 registros con 7 usuarios distintos hace una sola lectura de `profile`.
- **CA-08.9** (RF-08.2, RF-08.3) — Los registros históricos con `relatedId` legacy (17 caracteres), escritos por Meteor, se listan junto a los escritos por el ms para el mismo `relatedId`, en orden de `createdAt`.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0032 | Dominio: `BondHistory`, `HistoryType`, `ChangeEntry`, `AddParams` (con `Validation`) y errores | impl | F2 | RF-08.1, RF-08.6 | BLM-0009 | `internal/core/bondHistory/domain/*.go`, `internal/core/shared/domain/shared.text.go` (claves `BOND_SOLD` y `BOND_BOUND_STATUS_CHANGED`) |
| BLM-0033 | Ports: `BondHistoryPort`, `BondHistoryApplicationPorts` y `BondHistoryMongoccRepositoryPorts` | impl | F2 | RF-08.1, RF-08.2, RF-08.3 | BLM-0032 | `internal/core/bondHistory/ports/bondHistory.ports.go`, `internal/core/bondHistory/domain/bondHistory.dto.domain.go` (tipo de respuesta `HistoryEntry`) |
| BLM-0034 | Repositorio: `InsertOne`, `FindByRelated` y `EnsureIndexes` | impl | F2 | RF-08.1, RF-08.2, RF-08.3 | BLM-0033 | `internal/core/bondHistory/infrastructure/mongocc/bondHistory.mongocc.repository.go`, `*.adapter.go` |
| BLM-0035 | Caso de uso `Add`, expuesto como `BondHistoryPort` (sesión o modo sistema) e inicialización en `server.bondHistory.go` | impl | F2 | RF-08.1, RF-08.6 | BLM-0034, BLM-0005 | `internal/core/bondHistory/application/bondHistory.add.go`, `cmd/api/server/server.bondHistory.go` |
| BLM-0036 | Textos faltantes en el catálogo y helper `FromNow` con los umbrales de moment | impl | F2 | RF-08.4, RF-08.5 | BLM-0018 | `internal/core/shared/domain/shared.text.go`, `shared.fromNow.go` |
| BLM-0037 | Caso de uso `ListByRelated` con presentación (perfiles en lote, `Format`, nombres de estado y `FromNow`) | impl | F2 | RF-08.2, RF-08.3, RF-08.4 | BLM-0034, BLM-0017, BLM-0036 | `internal/core/bondHistory/application/bondHistory.listByRelated.go`, `bondHistory.present.go` |
| BLM-0038 | Tests unitarios de la presentación, los textos y `FromNow` | test | F2 | CA-08.5, CA-08.6, CA-08.7 | BLM-0037, BLM-0036 | `internal/core/bondHistory/application/bondHistory.present_test.go`, `internal/core/shared/domain/shared.fromNow_test.go` |
| BLM-0063 | Handler `GET /bond-objects/{id}/history` (admin) | impl | F3 | RF-08.2 | BLM-0037, BLM-0056, BLM-0026 | `cmd/api/handlers/bondObjects/bondObjects.history.get.go` |
| BLM-0099 | Caso de uso `HistoryOf` en bondBound (control de dueño y delegación) y handler `GET /bond-bounds/{id}/history` | impl | F5 | RF-08.3 | BLM-0037, BLM-0077, BLM-0084, BLM-0026 | `internal/core/bondBound/application/bondBound.history.go`, `cmd/api/handlers/bondBounds/bondBounds.history.get.go` |
| BLM-0100 | Tests de integración: `Add` (con transacción), listados, permisos y lectura en lote | test | F5 | CA-08.1, CA-08.2, CA-08.3, CA-08.4, CA-08.8, CA-08.9 | BLM-0063, BLM-0099 | `internal/core/bondHistory/infrastructure/mongocc/*_integration_test.go`, `cmd/api/handlers/*/*.history_test.go` |

BLM-0032 a BLM-0036 y BLM-0038 van en la fase F2, porque el catálogo (F3) ya escribe historial. BLM-0063 va en la F3, junto con los handlers de bondObject. BLM-0099 y BLM-0100 van en la F5, porque necesitan el repositorio de `bondBound`. Ver `tasks.md`.

## 11. Fuera de alcance y preguntas abiertas

- **Pregunta:** ¿los textos propuestos para `BOND_SOLD` y `BOND_BOUND_STATUS_CHANGED` son los correctos? Hay que validarlos con negocio.
- **Pregunta:** ¿`createdAtFromNow` debería estar en español? Por paridad sale en inglés.
