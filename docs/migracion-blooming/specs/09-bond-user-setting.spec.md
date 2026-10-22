# Spec 09 — Preferencias del inversionista (`bondUserSetting`)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondUserSetting` |
| Handlers | `cmd/api/handlers/bondUserSettings` |
| Colección | `bondUserSetting` (lectura de agregación sobre `bondBound` y `balanceAccount`) |
| Depende de | Spec 00 (`BalancePort.GetAccount`, `ProfileReader`) |
| La usa | Spec 05 (cuenta destino de la venta y el canje) |

## 1. Resumen y alcance

Cada inversionista tiene **un** documento de preferencias con:
- su idioma;
- el bono seleccionado en la UI;
- **la cuenta donde se acreditan las ventas y los canjes** (`selectedAccountId`).

El documento se crea solo la primera vez que el usuario entra. También sirve como resumen del portafolio: total invertido y total madurado.

Mientras el módulo de saldo siga en Meteor, Meteor necesita actualizar `selectedAccountId` cuando se crea una cuenta. Para eso hay un endpoint interno.

## 2. Origen legacy

Archivo: `backend/server/model/bond.userSetting.js` (257 líneas). El consumidor externo es `backend/server/model/balance.account.js:492-500` (`balanceAccount.insert` → `bondUserSetting.setSelectedAccount`).

| Operación legacy | Línea | Tipo | Destino |
|---|---|---|---|
| `bondUserSettings` | 18 | publicación | Fusionada en `GET /bond-user-settings/me` (RF-09.1) |
| `bondUserSetting.list` | 27 | método | RF-09.6 |
| `bondUserSetting.getByUserLogged` | 35 | método | RF-09.1, RF-09.2 |
| `bondUserSetting.get` | 66 | método | RF-09.7 |
| `bondUserSetting.insert` | 75 | método | **Fusionado**: autocreación interna de RF-09.1 (no se expone) |
| `bondUserSetting.update` | 96 | método | **Descartado**: `$set {}` vacío |
| `bondUserSetting.setSelectedAccount` | 111 | método | RF-09.3 (usuario) y RF-09.9 (interno) |
| `bondUserSetting.setSelectedBondBound` | 125 | método | RF-09.4 |
| `bondUserSetting.setLanguage` | 165 | método | RF-09.5 |
| `bondUserSetting.remove` | 178 | método | RF-09.8 |
| `userSettingExtraInfo` | 191 | helper | RF-09.2 |

## 3. Requisitos funcionales

- **RF-09.1** — Mis preferencias.
  - Busca el setting con `{userId: session.userId, isRemove: false}`.
  - Si no existe, lo **crea de forma atómica** (upsert por `userId`) con:
    - `language: "es"`;
    - `selectedAccountId`: el `_id` de la primera cuenta del usuario (`balanceAccount` con `userId`, `isRemove: false` y la `agencyId` de la sesión, ordenadas por `createdAt` asc), o `null`;
    - `selectedBondBoundId: null`;
    - `userId` y `agencyId` de la sesión.
  - Responde el setting más los datos de RF-09.2.
- **RF-09.2** — Datos agregados del setting:
  - `account`: la cuenta seleccionada con la proyección `{_id, number (enmascarado), available, statusId}`, o `null`.
    - Se lee con `BalancePort.GetAccount`, con control de dueño: es `null` si la cuenta no existe, está borrada o no es del usuario.
    - El legado la leía sin filtros, y por U6 pudo quedar apuntando a la cuenta de otro usuario;
  - `user`: el perfil público (`ProfileReader.GetPublic`, sin `password`);
  - `utcDiff`: el de la sesión;
  - `boundSummary = {investment, matured, profit: 0}`, en **una sola agregación** sobre `bondBound` con `{isRemove: false, agencyId, userId}` agrupada por `statusId`:
    - `investment = Σ amount` de los bonos en estado `'1'`;
    - `matured = Σ amount` de los bonos en estado `'2'`.
- **RF-09.3** — Cambiar la cuenta destino (usuario).
  - Recibe `{selectedAccountId}`. La cuenta debe existir y ser **del usuario de la sesión** (`BalancePort.GetAccount`); si no, `ERR_ACCOUNT_NOT_FOUND`.
  - El `userId` sale **solo** de la sesión. Corrige U6: el legado aceptaba `data.userId`.
  - Si el setting no existe, lo crea (upsert, corrige U1).
  - Responde el setting actualizado.
- **RF-09.4** — Cambiar el bono seleccionado. Recibe `{selectedBondBoundId}` (string o `null`). Hace upsert sobre el setting del usuario y responde el setting.
- **RF-09.5** — Cambiar el idioma. Recibe `{language}`, que debe ser un string no vacío de hasta 5 caracteres (`ERR_INVALID_LANGUAGE`). Hace upsert y responde el setting.
- **RF-09.6** — Listado de administración.
  - Devuelve los settings con `isRemove: false` más `AgencyScope` (corrige U2: el legado no filtraba por agencia).
  - Paginado (RF-00.18) y ordenado por `createdAt` desc.
- **RF-09.7** — Detalle de administración. Devuelve el setting por `_id` con `isRemove: false`; si no existe, `ERR_BOND_USER_SETTING_NOT_FOUND`.
- **RF-09.8** — Baja de administración. Soft delete por `_id`.
- **RF-09.9** — Sincronización interna desde Meteor.
  - `PUT /internal/bond-user-settings/{userId}/selected-account` con `{selectedAccountId}`. Solo requiere `x-secret` (`NoRequiresAuthentication`).
  - Actualiza `selectedAccountId` **solo si el setting de ese `userId` existe**. Es paridad con `balanceAccount.insert`, que solo llama si encuentra el setting.
  - Responde `200 {updated: true|false}`.
  - Reemplaza la llamada `Caller.call('bondUserSetting.setSelectedAccount', …, {userId})` de `balanceAccount.insert`. Meteor la cambia por un HTTP a este endpoint (BLM-0072).
- **RF-09.10** — Lectura interna `GetByUser(ctx, userID) (*BondUserSetting, error)` para la spec 05. **No** autocrea: devuelve `nil` si el setting no existe.

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| GET | `/bond-user-settings/me` | usuario | RF-09.1, RF-09.2 |
| PATCH | `/bond-user-settings/me/selected-account` | usuario | RF-09.3 |
| PATCH | `/bond-user-settings/me/selected-bond-bound` | usuario | RF-09.4 |
| PATCH | `/bond-user-settings/me/language` | usuario | RF-09.5 |
| GET | `/bond-user-settings` | admin | RF-09.6 |
| GET | `/bond-user-settings/{id}` | admin | RF-09.7 |
| DELETE | `/bond-user-settings/{id}` | admin | RF-09.8 |
| PUT | `/internal/bond-user-settings/{userId}/selected-account` | interno (`x-secret`) | RF-09.9 |

**`GET /bond-user-settings/me`** → `200`

```json
{ "_id": "…", "userId": "u1", "agencyId": "A", "language": "es",
  "selectedAccountId": "acc1", "selectedBondBoundId": null,
  "account": { "_id": "acc1", "number": "****************4821", "available": 5000, "statusId": "1" },
  "user": { "_id": "u1", "first_name": "Juan", "last_name": "Pérez", "email": "…" },
  "utcDiff": -5,
  "boundSummary": { "investment": 1500, "matured": 2000, "profit": 0 },
  "createdAt": "…", "updatedAt": "…", "isRemove": false }
```

**Cambio de contrato respecto al legado:**
- Los `set*` respondían `1`; ahora responden el setting.
- `getByUserLogged` sin sesión respondía `-13`; ahora la sesión la valida el middleware (401).
- `user` ya no trae el perfil completo.

**Errores**

| Sentinel | HTTP | Código legacy | Cuándo |
|---|---|---|---|
| `ERR_INVALID_ID` | 400 | — | Un id de path, query o body no es UUID v4 ni Meteor de 17 caracteres (spec 00, RF-00.19) |
| `ERR_INVALID_LANGUAGE` | 400 | — | `language` vacío o de más de 5 caracteres |
| `ERR_ACCOUNT_NOT_FOUND` | 404 | — | La cuenta no existe o no es del usuario de la sesión |
| `ERR_BOND_USER_SETTING_NOT_FOUND` | 404 | -11 | No existe o `isRemove: true` |
| `ERR_FORBIDDEN` | 403 | — | No es admin (spec 00) |
| `UNAUTHORIZED_INVALID_SECRET` | 401 | — | Endpoint interno sin `x-secret` válido (spec 00) |

## 5. Reglas de negocio

- **Unicidad:** un setting vivo por `userId`. Se garantiza con un índice único parcial `{userId: 1}` y `partialFilterExpression: {isRemove: false}`, más upserts con `{userId, isRemove: false}`.
- **Duplicados legacy:** pueden existir, porque el legado no era atómico. Antes de crear el índice, el script de auditoría (BLM-0065) informa los duplicados y, con `-apply`, borra de forma lógica todos menos el más reciente.
  - El más reciente es el último modificado: `updatedAt` desc, luego `createdAt` desc y luego `_id`. Es el que el legado venía actualizando, porque su `findOne({userId})` siempre tomaba el mismo documento.
  - Mientras haya duplicados, la lectura y los upserts eligen ese mismo documento.

## 6. Modelo de datos

```go
type BondUserSetting struct {
    ID                  string    `json:"_id" bson:"_id"`
    UserID              string    `json:"userId" bson:"userId"`
    AgencyID            string    `json:"agencyId" bson:"agencyId"`
    Language            string    `json:"language" bson:"language"`
    SelectedAccountID   *string   `json:"selectedAccountId" bson:"selectedAccountId"`
    SelectedBondBoundID *string   `json:"selectedBondBoundId" bson:"selectedBondBoundId"`
    CreatedAt, UpdatedAt time.Time // createdAt, updatedAt
    IsRemove            bool      `json:"isRemove" bson:"isRemove"`
}
```

El modelo en `@futurapps/models` (`blooming.BondUserSetting`) declara `_id: String`, `createdAt`, `updatedAt` e `isRemove` con `strict: false`, y es compatible.

Hay settings legacy con `userId` guardado como ObjectId. Se leen con su hex como `userId`, para que el listado y el detalle de administración no fallen y se puedan borrar.

Índices: el único parcial de §5 y `{agencyId: 1, isRemove: 1, createdAt: -1}`.

## 7. Ports y dependencias

- `BondUserSettingApplicationPorts`: `GetMine`, `GetByUser` (RF-09.10), `SetSelectedAccount`, `SetSelectedBondBound`, `SetLanguage`, `List`, `GetByID`, `Remove` y `SyncSelectedAccount` (RF-09.9).
- `BondUserSettingMongoccRepositoryPorts`: `FindByUser`, `UpsertDefault`, `UpsertField`, `FindPage`, `FindOneByID`, `SoftDelete` y `EnsureIndexes`.
- Adapters de **solo lectura** en `bondUserSetting/infrastructure/mongocc/`:
  - `BoundSummary(ctx, userID, agencyID)`, una agregación sobre `bondBound`;
  - `FirstAccountID(ctx, userID, agencyID)`, una consulta sobre `balanceAccount`.
- `BalancePort.GetAccount` y `ProfileReader` (spec 00).

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| U1 | `setSelectedAccount`, `setSelectedBondBound` y `setLanguage` lanzan `TypeError` si no existe el setting. | Corregir (upsert) | Llamadas que no responden. | BLM-0068 |
| U2 | `bondUserSetting.list` no filtra por agencia ni por permisos. | Corregir | Seguridad. | BLM-0069 |
| U3 | La autocreación de `getByUserLogged` no es atómica (duplicados) y no filtra `isRemove`. | Corregir | Integridad. | BLM-0065, BLM-0067 |
| U4 | `user` devuelve el perfil completo (sin `password`, pero con el resto de los campos), y `account` el documento entero. | Corregir | Exposición de datos. | BLM-0067 |
| U6 | `setSelectedAccount` acepta `data.userId`: cualquier sesión puede cambiar la cuenta destino de **otro** usuario y desviar a su propia cuenta el dinero de las ventas y canjes de la víctima. | Corregir | Seguridad y dinero (crítico). | BLM-0068 |
| U7 | `setSelectedAccount` no valida que la cuenta sea del usuario. | Corregir | Integridad: crédito en una cuenta ajena. | BLM-0068 |

## 9. Criterios de aceptación

- **CA-09.1** (RF-09.1) — Un usuario sin setting y con dos cuentas en su agencia llama a `GET /me`. Se crea un setting con `language "es"`, `selectedAccountId` = la cuenta más antigua y `selectedBondBoundId null`. Una segunda llamada devuelve el mismo `_id`.
- **CA-09.2** (RF-09.1) — Un usuario sin cuentas recibe `selectedAccountId: null` y `account: null`.
- **CA-09.3** (RF-09.1) — Diez llamadas concurrentes a `GET /me` de un usuario nuevo crean exactamente un documento.
- **CA-09.4** (RF-09.2) — Con bonos propios en estado `'1'` (1000 y 500), `'2'` (2000) y `'5'` (700), `boundSummary` es `{investment 1500, matured 2000, profit 0}`. `account.number` sale enmascarado y `user` no tiene `password`.
- **CA-09.5** (RF-09.3) — Con una cuenta propia, `PATCH /me/selected-account` responde `200`. Con la cuenta de otro usuario responde `404 ERR_ACCOUNT_NOT_FOUND`. Un `userId` en el body se ignora y nunca modifica el setting de otro usuario.
- **CA-09.6** (RF-09.3, RF-09.4, RF-09.5) — Sin setting previo, cualquiera de los tres `PATCH` lo crea y responde `200`, sin `TypeError`.
- **CA-09.7** (RF-09.5) — `language: ""` o `language: "español"` responde `400 ERR_INVALID_LANGUAGE`.
- **CA-09.8** (RF-09.6) — Un admin de una agencia no principal ve solo los settings de su agencia, paginados. Un no admin recibe `403`.
- **CA-09.9** (RF-09.7, RF-09.8) — `GET` y `DELETE /bond-user-settings/{id}` sobre un id inexistente responden `404`. `DELETE` deja `isRemove: true`.
- **CA-09.10** (RF-09.9) — `PUT /internal/bond-user-settings/u1/selected-account` con `x-secret` válido y sin `x-token` actualiza el setting de u1 y responde `{updated: true}`. Si u1 no tiene setting responde `{updated: false}` y no crea nada. Sin `x-secret` responde `401`.
- **CA-09.11** (RF-09.10) — `GetByUser` de un usuario sin setting devuelve `nil` y no crea nada.
- **CA-09.12** (RF-09.1, RF-09.3) — Dado un setting legacy (`_id` de 17 caracteres) para el usuario, `GET /me` lo devuelve sin crear otro, y `PATCH /me/selected-account` lo actualiza conservando su `_id`.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0064 | Dominio: `BondUserSetting`, DTOs (`SetSelectedAccountParams`, `SetLanguageParams`, …) con `Validation` y errores | impl | F4 | RF-09.1, RF-09.3, RF-09.4, RF-09.5 | BLM-0009 | `internal/core/bondUserSetting/domain/*.go` |
| BLM-0065 | Ports y repositorio: `FindByUser`, `UpsertDefault`, `UpsertField`, `FindPage`, `FindOneByID`, `SoftDelete`, `EnsureIndexes` (índice único parcial) y script de auditoría de duplicados | impl | F4 | RF-09.1, RF-09.3, RF-09.4, RF-09.5, RF-09.6, RF-09.7, RF-09.8 | BLM-0064 | `internal/core/bondUserSetting/ports/*.go`, `internal/core/bondUserSetting/infrastructure/mongocc/*.adapter.go`, `cmd/tools/dedupusersettings/main.go` |
| BLM-0066 | Adapters de solo lectura `BoundSummary` y `FirstAccountID` | impl | F4 | RF-09.1, RF-09.2 | BLM-0065 | `internal/core/bondUserSetting/infrastructure/mongocc/bondUserSetting.boundSummary.adapter.go`, `bondUserSetting.firstAccount.adapter.go` |
| BLM-0067 | Casos de uso `GetMine` (autocreación atómica y datos agregados) y `GetByUser` | impl | F4 | RF-09.1, RF-09.2, RF-09.10 | BLM-0066, BLM-0012, BLM-0017 | `internal/core/bondUserSetting/application/bondUserSetting.application.go`, `bondUserSetting.getMine.go`, `bondUserSetting.getByUser.go` |
| BLM-0068 | Casos de uso `SetSelectedAccount` (cuenta del usuario), `SetSelectedBondBound` y `SetLanguage`, con upsert | impl | F4 | RF-09.3, RF-09.4, RF-09.5 | BLM-0065, BLM-0012 | `internal/core/bondUserSetting/application/bondUserSetting.set*.go` |
| BLM-0069 | Casos de uso admin `List`, `GetByID` y `Remove` | impl | F4 | RF-09.6, RF-09.7, RF-09.8 | BLM-0065, BLM-0007 | `internal/core/bondUserSetting/application/bondUserSetting.list.go`, `bondUserSetting.getById.go`, `bondUserSetting.remove.go` |
| BLM-0070 | Caso de uso `SyncSelectedAccount` y endpoint interno con `NoRequiresAuthentication` | impl | F4 | RF-09.9 | BLM-0065, BLM-0004, BLM-0026 | `internal/core/bondUserSetting/application/bondUserSetting.syncSelectedAccount.go`, `cmd/api/handlers/bondUserSettings/bondUserSettings.internalSelectedAccount.put.go` |
| BLM-0071 | Handlers y `server.bondUserSetting.go` | impl | F4 | RF-09.1, RF-09.3, RF-09.4, RF-09.5, RF-09.6, RF-09.7, RF-09.8 | BLM-0067, BLM-0068, BLM-0069, BLM-0026 | `cmd/api/handlers/bondUserSettings/*.go`, `cmd/api/server/server.bondUserSetting.go` |
| BLM-0072 | **(repo backend, Meteor)** `balanceAccount.insert` llama por HTTP al endpoint interno en lugar de `Caller.call('bondUserSetting.setSelectedAccount')` | impl | F4 | RF-09.9 | BLM-0070 | `backend/server/model/balance.account.js` |
| BLM-0073 | Tests unitarios de casos de uso y handlers (validaciones, dueño de la cuenta, U6, permisos) | test | F4 | CA-09.2, CA-09.5, CA-09.6, CA-09.7, CA-09.8, CA-09.9, CA-09.11 | BLM-0071 | `internal/core/bondUserSetting/application/*_test.go`, `cmd/api/handlers/bondUserSettings/*_test.go` |
| BLM-0074 | Tests de integración: autocreación concurrente, `boundSummary` y endpoint interno | test | F4 | CA-09.1, CA-09.3, CA-09.4, CA-09.10, CA-09.12 | BLM-0071, BLM-0070 | `internal/core/bondUserSetting/infrastructure/mongocc/*_integration_test.go` |

## 11. Fuera de alcance y preguntas abiertas

- `boundSummary.profit` es siempre `0` en el legado y se mantiene así.
- **Pregunta:** ¿qué idiomas son válidos? Por ahora se acepta cualquier código de hasta 5 caracteres.
- **Pregunta:** ¿se valida que `selectedBondBoundId` sea un bono del usuario? El legado no lo hace y aquí tampoco, para no acoplar este dominio con `bondBound`.
