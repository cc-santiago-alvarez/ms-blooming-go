# Spec 00 — Plataforma base

| | |
|---|---|
| Estado | Borrador |
| Dominios | `shared`, `session`, `cmd/api` |
| Depende de | — |
| La usan | Todas las specs (01–10) |

## 1. Resumen y alcance

Deja `ms-blooming-go` arrancando como servicio y le da a los dominios las piezas compartidas:
- configuración y arranque;
- autenticación contra ms-oauth y autorización;
- CORS y formato de errores;
- IDs, helpers de dinero y fecha, y transacciones;
- los ports hacia otros módulos: saldo, notice, activity, email, exportador, perfil y agencia;
- textos, health y graceful shutdown.

Ningún endpoint de negocio vive aquí.

### Estado actual del esqueleto (a corregir)

| Problema | Archivo | Corrección |
|---|---|---|
| El módulo se llama `ms-blooming-go`, con `go 1.25.0` y todas las dependencias `// indirect`. | `go.mod` | `module github.com/futurapps/ms-blooming-go`, `go 1.27.0`, `go mod tidy`. |
| `cmd/main.go` y `cmd/api/server/server.deps.go` están vacíos. | `cmd/` | Implementar según §4. |
| El healthcheck se declara `package healthcheck` y apunta al puerto `25072`, que es el de petty-cash. | `cmd/healthcheck/main.go` | `package main`; el puerto se lee de `API_PORT`. |
| Importa `github.com/futurapps/ms-petty-cash-api/cmd/api/config` y `lib/pq`, y usa `serverDeps`, que no está definido. | `internal/database/mongo/connection.go` | Se elimina. La conexión pasa a `cmd/api/server/server.config.go`, como en petty-cash. |
| `log.Fatalf` si falta `.env.development`. | `cmd/api/config/config.go` | Se mantiene el comportamiento, pero se documenta y se agrega `.env.development` de ejemplo al README. |
| Restos de petty-cash: GOOSE, Postgres, JWT, `BALANCE_CHECK_CRON`. | `.env.example`, `environments.go` | Se quitan. Lista final en §5. |

## 2. Origen legacy

| Pieza legacy | Archivo | Equivalente Go |
|---|---|---|
| `Method.register` / `getSession` | `backend/server/lib/method.js` | Handlers nexus + `VerifySession` |
| `Session.check` (sesión + perfil + agencia + país, `utcDiff`) | `backend/server/lib/session.js` | `UserSession` de ms-oauth + resolución de `utcDiff` (RF-00.4) |
| `Collection` (`insert`, `update`, `hide`, `exist`) | `backend/server/lib/collection.js` | `New()` de entidades + repos mongocc + soft delete |
| `Util.getOptionsPage`, `parseCurrency`, `randomString` | `backend/server/lib/util.js` | `shared` helpers (RF-00.8, RF-00.12, RF-00.18) |
| `Text.*` y `String.prototype.format` | `backend/server/lib/text.js`, `lib/actividas-utils.js` | Catálogo de textos (RF-00.12) |
| `Result.*` (códigos numéricos) | `backend/server/lib/_variables.js` | Sentinels por dominio |
| `balanceAccount.withdraw` | `backend/server/model/balance.account.js:1461` | `BalancePort.Withdraw` (RF-00.10) |
| `balanceDetail.insert` + `balanceHistory.add` + `balanceOperation.insert` + `setAmounts` | `balance.detail.js:159`, `balance.history.js:190`, `balance.operation.js:283`, `balance.account.js:1899` | Adapter de `BalancePort` (RF-00.10) |
| `Notice.add` / `notice.insert` | `backend/server/lib/notice.js`, `model/notice.js:142` | `NoticePort` (RF-00.11) |
| `activity.insert` | `backend/server/model/activity.js:101` | `ActivityPort` (RF-00.11) |
| `Email.send` (SendGrid directo) | `backend/server/lib/email.js` | `NotificationPort` → ms-notification (RF-00.11) |
| `Api.post` a ms-exporter | `backend/server/lib/api.js` | `ExporterPort` (RF-00.14) |
| `agency.getSymbolCurrencyByAgencyId` | `backend/server/model/agency.js:135` | `AgencyReader` (RF-00.15) |

## 3. Requisitos funcionales

- **RF-00.1** — Arranque y configuración. `cmd/main.go` llama a `server.InitServer().Run()`.
  - `InitConfig()` carga `.env` y después `.env.<ENV>` (`ENV` por defecto `development`), y valida las variables requeridas de §5. Si falta una requerida, el proceso termina con código 1 y un mensaje que nombra la variable, sin mostrar valores.
  - Se conecta a Mongo con `mongocc.Connect`.
  - Construye `nexus.Server` con `PathPrefix: "/api/v1"`.
  - Inicializa los dominios en orden: `bondHistory` → `bondUserSetting` → `bondType` → `bondObject` → `bondBound`. Después registra los handlers y arranca el cron (spec 10).
- **RF-00.2** — `ValidateApiSecret`: compara el header `x-secret` con `API_SECRET` (en tiempo constante, tras quitar caracteres no imprimibles). Si falta o no coincide, responde `401 UNAUTHORIZED_INVALID_SECRET`. Los endpoints `IsPublic` (`/_health`, `/_routes`) se saltan esta validación.
- **RF-00.3** — `VerifySession`: valida la sesión que **creó ms-oauth**. ms-blooming-go no crea ni renueva sesiones.
  - Lee `x-token`. Si falta, responde `401 TOKEN_IS_REQUIRED`.
  - Llama a `GET {SESSION_SERVICE_URL}` (ms-oauth, `GET /api/v1/session`) con los headers `x-token` y `x-secret: SESSION_SERVICE_SECRET`. El context tiene timeout de 3 s y el cliente de 4 s.
  - Si la respuesta no es 200, o es un error de red o timeout, responde `401 TOKEN_INVALID`.
  - Con un 200, decodifica el `UserSession` (§6.1) y lo guarda en el context con `session_domain.SetUserSession`.
  - Los endpoints con `NoRequiresAuthentication` (internos) se saltan este paso, pero siguen exigiendo `x-secret`.
- **RF-00.4** — Zona horaria de la sesión. ms-oauth no entrega `gmt`, así que `utcDiff` (horas) se resuelve en este orden:
  1. el campo `utcDiff` o `gmt` del `UserSession`, si ms-oauth lo agrega en el futuro;
  2. el mapa por `countryCode` del legado: `CO:-5, VE:-4, MX:-7, US:-6`;
  3. `DEFAULT_UTC_DIFF` (por defecto `-5`).

  Se expone con `session_domain.UtcDiff(ctx) int`.
- **RF-00.5** — Autorización.
  - `RequireBloomingAdmin(handler)` responde `403 ERR_FORBIDDEN` si `session.adminBlooming != true`.
  - `shared.IsOwnerOrAdmin(ctx, ownerUserID) bool` es el helper para los casos de uso que operan sobre recursos de un usuario.
  - `shared.AgencyScope(ctx) *string` devuelve `nil` si `session.agency.isMain` (sin filtro de agencia); si no, devuelve `&session.agencyId`.
- **RF-00.6** — CORS con `rs/cors`.
  - `AllowOriginFunc` acepta los orígenes de `CORS_ORIGINS`, separados por coma, y los patrones `*.dominio.com` compilados a regex.
  - Headers permitidos: `Origin, X-Requested-With, Content-Type, Accept, X-Secret, X-Token`. Métodos: `GET, POST, PUT, PATCH, DELETE, OPTIONS`. `AllowCredentials: true`.
- **RF-00.7** — Formato de respuestas.
  - Éxito: la entidad, el arreglo o el objeto resultado, sin envelope (`nexus.ResponseWithJSON`).
  - Error: `nexus.ResponseWithError(w, status, "ERR_X")`.
  - Los errores no mapeados responden `500 ERR_INTERNAL`; el detalle se loguea y nunca se devuelve.
  - Los listados paginados responden `{items, total, page, limit, totalPages}` (RF-00.18).
- **RF-00.8** — IDs y helpers de datos.
  - `shared.NewID()` devuelve `uuid.New().String()`. Es el único generador de `_id` para documentos nuevos.
  - `shared_domain.IDFormat(id)` devuelve `uuid`, `meteor` o `invalid`; `shared_domain.IsValidID(id)` es `IDFormat(id) != invalid` (RF-00.19).
  - `shared.Now()` devuelve `time.Now().UTC()`.
  - `shared.Round2(float64)` redondea a 2 decimales (half away from zero).
  - `shared.StartOfDay(t, utcDiff)` devuelve el inicio del día en la zona `utcDiff`, expresado en UTC.
  - `shared.DaysBetween(a, b, utcDiff)` es la diferencia en días calendario.
  - `shared.MaskAccountNumber(n)` devuelve `"****************" + últimos 4`.
  - `shared.RandomAlnum(n)` sirve para `token` y `reference`; no se usa para `_id`.
- **RF-00.9** — `TransactionRunnerPort.RunInTransaction(ctx, fn func(txCtx context.Context) error) error`.
  - Arranca la sesión de Mongo con `WithTransaction`.
  - Los repositorios que reciben `txCtx` participan de la transacción.
  - Si el callback devuelve error, hace rollback. Requiere replica set.
- **RF-00.10** — `BalancePort`, que escribe directo en las colecciones de saldo y replica el legado.
  - `GetAccount(ctx, accountID, userID) (*BalanceAccount, error)` lee `balanceAccount` con `{_id, userId, isRemove: false}` y devuelve `{_id, userId, agencyId, statusId, available, number}`. Devuelve `nil, nil` si la cuenta no existe **o no es del usuario**.
  - `Withdraw(ctx, WithdrawParams{AccountID, UserID, AgencyID, DescriptionID, Amount, Comment}) (*BalanceMovement, error)` hace las validaciones en el orden legacy:
    1. la cuenta existe para `UserID` con `isRemove:false`; si no, `ERR_ACCOUNT_NOT_FOUND`;
    2. `statusId == '1'`; si no: `'0'` → `ERR_ACCOUNT_INACTIVE`, `'2'` → `ERR_ACCOUNT_SUSPENDED`, `'3'` → `ERR_ACCOUNT_CANCELLED`, otro → `ERR_ACCOUNT_STATUS`;
    3. `Amount > 0`; si no, `ERR_ACCOUNT_WITHDRAW_AMOUNT_INVALID`;
    4. `Amount <= available`; si no, `ERR_ACCOUNT_WITHDRAW_AMOUNT_MORE_THAN_AVAILABLE`.

    Después escribe:
    - `balanceDetail` con `operationTypeId '1'` (MD) y monto negativo;
    - `balanceHistory` (`typeId '3'`, `text 'BALANCE_ACCOUNT_WITHDRAW'`, `params [detailId]`);
    - `balanceOperation` (`typeId '1'`, `categoryId '3'`, `details [accountId, detailId]`, `concept: comment`);
    - la actualización del saldo, que es **atómica y condicional**: filtro `{_id, available: {$gte: amount}}` con `$inc {available: -amount, global: -amount}`. Si no modifica nada, `ERR_ACCOUNT_WITHDRAW_AMOUNT_MORE_THAN_AVAILABLE`.

    `effectiveDate` según §7.1.
  - `Deposit(ctx, DepositParams{AccountID, UserID, AgencyID, DescriptionID, Amount, Comment}) (*BalanceMovement, error)` escribe:
    - `balanceDetail` con `operationTypeId '2'` (MC), monto positivo y `effectiveDate: null`;
    - `balanceHistory` (`typeId '2'`, `text 'BALANCE_ACCOUNT_DEPOSIT'`);
    - `balanceOperation` (`categoryId '2'`);
    - `$inc {available: +amount, global: +amount}`.
  - Cada movimiento devuelve `{detailId, reference, amount, balance, available}`.
  - Todas las escrituras usan el `ctx` recibido, para participar de la transacción del llamador.
  - Las cuentas con `descriptionId` `BOND_MD '7'` **no** suman a `deferred`. Es paridad con `setAmounts`, y como este port solo maneja descripciones de bonos, `deferred` nunca cambia.
- **RF-00.11** — Notificaciones.
  - `NoticePort.Add(ctx, Notice{Type, UserID, AgencyID, Title, Message})` inserta en `notice`: `typeId` (`bond '3'`, `balance '1'`), `categoryId '1'` (info), `unread: null`, `_id` UUID y timestamps.
  - `ActivityPort.Add(ctx, Activity{Description, RelatedID, AgencyID, UserID, Params})` inserta en `activity` con `moduleId '1'` y `name 'Blooming'`.
  - `NotificationPort.Send(ctx, group, notification, params)` hace `POST {MS_NOTIFICATION_URL}/notify/{group}/{notification}` con body `{"data":{"params":{...}}}`, headers `x-secret: MS_NOTIFICATION_SECRET` y el `x-token` de la sesión cuando existe, y timeout de 15 s.
  - Los handlers de ms-notification que hacen falta están en §7.3.
  - Ningún fallo de estos tres ports se propaga al llamador: se loguea (constitución §9).
- **RF-00.12** — Textos y formato.
  - `shared_domain.Text` es el catálogo de las plantillas legacy que se usan, en §7.2.
  - `shared.Format(tpl string, args ...any)` reemplaza `{0}`, `{1}`, …
  - `shared.FormatCurrency(v float64) string` da 2 decimales y separador de miles con coma (`1234567.5` → `"1,234,567.50"`). Corrige el truncado de `Util.parseCurrency` (B14).
- **RF-00.13** — `RequestLog` registra `[API_NAME] METHOD path status duración`, omite `/_health` y **nunca** incluye headers ni body.
- **RF-00.14** — `ExporterPort.Excel(ctx, report string, payload any) ([]byte, error)` hace `POST {MS_EXPORTER_URL}/export/excel/report/{report}`.
  - Envía `{"data": payload}` (contrato de ms-exporter) y los headers `x-secret` y `x-token`.
  - Una respuesta `201` trae `buffer.toJSON()` (`{"type":"Buffer","data":[...]}`), que se convierte a `[]byte`.
  - Cualquier otro status da `ERR_EXPORT_FAILED`.
- **RF-00.15** — Lectores compartidos (solo lectura).
  - `ProfileReader.GetPublic(ctx, ids...)` lee de `profile` con la proyección `{_id, id, first_name, last_name, email, avatar_url, agencyId}`.
  - `AgencyReader.SymbolCurrency(ctx, agencyId)` sigue agency → country y devuelve `symbolCurrency`, `'$'` por defecto.
- **RF-00.16** — Health y apagado.
  - `GET /_health` es el de nexus.
  - `cmd/healthcheck` es un binario `package main` que hace `GET /_health` por TCP a `127.0.0.1:${API_PORT}` y termina en 1 si la respuesta no es 200.
  - Graceful shutdown: `signal.NotifyContext(SIGINT, SIGTERM)` → `http.Server.Shutdown` con timeout de 20 s → detener el cron (esperando el job en curso) → `mongo.Client.Disconnect`.
- **RF-00.17** — `internal/core/architecture_test.go` falla si un paquete `*_domain` importa otro `*_domain` distinto de `shared_domain`.
- **RF-00.18** — Paginación estándar.
  - `shared_domain.Pagination{Page, Limit}` se normaliza a `Page >= 1` (por defecto 1) y `Limit` entre 1 y 100 (por defecto 10, el `Pagination.default` del legado).
  - `skip = (page-1)*limit`.
  - El resultado es `{items, total, page, limit, totalPages}`.
- **RF-00.19** — Compatibilidad de IDs. El servicio trabaja con dos formatos de `_id` en las mismas colecciones:
  - **UUID v4**, en todos los documentos que crea `ms-blooming-go`;
  - **Meteor**, 17 caracteres alfanuméricos (`Random.id()`), en los documentos creados por el backend legacy. Por ejemplo, la agencia por defecto `tDr6NwSsDe4Lnccp8`.

  Reglas:
  - Toda consulta, actualización y soft delete funciona igual con los dos formatos. Los documentos legacy **no se migran** ni se les reescribe el `_id`.
  - Las referencias pueden ser mixtas: un documento nuevo apunta a uno legacy y viceversa (`bondObjectId`, `bondTypeId`, `userId`, `agencyId`, `selectedAccountId`, `relatedId`, …).
  - Los IDs de entrada (path, query y body) se validan **en el handler** con `IsValidID` antes de llamar al caso de uso. Si no coinciden con ninguno de los dos formatos, se responde `400 ERR_INVALID_ID` sin consultar Mongo.
  - Para path params, el helper común es `handlers_shared.PathID(w, r, "id") (string, bool)`, que escribe el 400 y devuelve `false`.
  - Formatos:
    - UUID v4: `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`;
    - Meteor: `^[A-Za-z0-9]{17}$`.

## 4. Contrato API

Esta spec no define endpoints de negocio. Expone:

| Método | Ruta | Auth | Respuesta |
|---|---|---|---|
| GET | `/_health` | Pública | `200 text/plain "<API_NAME> is running"` |
| GET | `/_routes` | Pública (solo con `API_DEBUG=true`) | Lista de rutas registradas |

**Errores transversales**

| Sentinel | HTTP | Cuándo |
|---|---|---|
| `UNAUTHORIZED_INVALID_SECRET` | 401 | `x-secret` falta o no coincide |
| `TOKEN_IS_REQUIRED` | 401 | Falta `x-token` |
| `TOKEN_INVALID` | 401 | ms-oauth rechaza el token o no responde |
| `ERR_FORBIDDEN` | 403 | Endpoint admin sin `adminBlooming`, o recurso ajeno |
| `BODY_REQUIRED` / `INVALID_REQUEST_BODY` | 400 | El body está vacío o no es JSON válido |
| `ERR_INVALID_ID` | 400 | Un id de path, query o body no es UUID v4 ni Meteor de 17 caracteres (RF-00.19) |
| `ERR_INTERNAL` | 500 | Cualquier error no mapeado |

**Errores del `BalancePort`** (los reexponen las specs 04 y 05)

| Sentinel | HTTP | Código legacy |
|---|---|---|
| `ERR_ACCOUNT_NOT_FOUND` | 404 | -700 |
| `ERR_ACCOUNT_INACTIVE` | 422 | -750 |
| `ERR_ACCOUNT_SUSPENDED` | 422 | -751 |
| `ERR_ACCOUNT_CANCELLED` | 422 | -752 |
| `ERR_ACCOUNT_STATUS` | 422 | -730 |
| `ERR_ACCOUNT_WITHDRAW_AMOUNT_INVALID` | 400 | -710 |
| `ERR_ACCOUNT_WITHDRAW_AMOUNT_MORE_THAN_AVAILABLE` | 422 | -720 |
| `ERR_EXPORT_FAILED` | 502 | -1000 |

## 5. Configuración (`.env.example`)

| Variable | Requerida | Default | Uso |
|---|---|---|---|
| `ENV` | no | `development` | Selecciona `.env.<ENV>` |
| `API_NAME` | sí | — | Nombre en logs y en `/_health` |
| `API_PORT` | sí | — | Puerto HTTP (también lo usa el binario de healthcheck) |
| `API_SECRET` | sí | — | Valor esperado de `x-secret` |
| `API_DEBUG` | no | `false` | Activa `/_routes` y el log de nexus |
| `CORS_ORIGINS` | sí | — | Orígenes permitidos, separados por coma; acepta `*.dominio.com` |
| `FA_MONGO_URL` | sí | — | URI de Mongo (replica set, por las transacciones) |
| `FA_MONGO_DBNAME` | sí | — | Base `futurapps` |
| `FA_MONGO_DEBUG` | no | `false` | Log de queries de mongocc |
| `SESSION_SERVICE_URL` | sí | — | `http://<ms-oauth>/api/v1/session` |
| `SESSION_SERVICE_SECRET` | sí | — | `x-secret` de ms-oauth |
| `MS_NOTIFICATION_URL` | sí | — | Base de ms-notification |
| `MS_NOTIFICATION_SECRET` | sí | — | `x-secret` de ms-notification |
| `MS_EXPORTER_URL` | sí | — | Base de ms-exporter |
| `MS_EXPORTER_SECRET` | sí | — | `x-secret` de ms-exporter |
| `BLOOMING_ADMIN_EMAILS` | no | vacío | Destinatarios del aviso de retiro al admin (legacy `Constant.ADMIN_BALANCE_EMAILS`) |
| `DEFAULT_UTC_DIFF` | no | `-5` | Fallback de zona horaria (RF-00.4) |
| `BULK_PURCHASE_MAX_QUANTITY` | no | `100` | Tope de la compra masiva (spec 04) |
| `BOND_MATURITY_CRON` | no | vacío = desactivado | Expresión del cron de maduración (spec 10) |

## 6. Modelo de datos

### 6.1 `UserSession` (lo devuelve ms-oauth, `src/modules/session/session.service.js#readSession`)

```go
type UserSession struct {
    UserID        string        `json:"userId"`
    Token         string        `json:"token"`
    AgencyID      string        `json:"agencyId"`
    Agency        SessionAgency `json:"agency"`        // {_id, name, isMain}
    CountryCode   string        `json:"countryCode"`
    FirstName     string        `json:"first_name"`
    LastName      string        `json:"last_name"`
    Email         string        `json:"email"`
    AdminBlooming bool          `json:"adminBlooming"`
    SymbolCurrency string       `json:"symbolCurrency"`
    AccessUser    AccessUser    `json:"accessUser"`    // adminAccess []string, isAdmin map
}
```

Solo se decodifican estos campos; el resto se ignora.

### 6.2 Colecciones externas donde escribe el ms

| Colección | Quién escribe | Campos |
|---|---|---|
| `balanceDetail` | `BalancePort` | `_id, balanceAccountId, operationTypeId, reference (16 alfanum), descriptionId, amount (con signo, 2 dec), balance, agencyId, effectiveDate, userId, comment, createdAt, updatedAt, isRemove` |
| `balanceHistory` | `BalancePort` | `_id, balanceAccountId, userId, typeId, text, params, agencyId, timestamps, isRemove` |
| `balanceOperation` | `BalancePort` | `_id, agencyId, userId, typeId '1', categoryId, details[], extraInfo null, concept, timestamps, isRemove` |
| `balanceAccount` | `BalancePort` (solo `$inc available/global` y `updatedAt`) | — |
| `notice` | `NoticePort` | `_id, typeId, userId, agencyId, message, title, categoryId, unread: null, timestamps, isRemove` |
| `activity` | `ActivityPort` | `_id, description, moduleId '1', name 'Blooming', relatedId, agencyId, userId, params, timestamps, isRemove` |

Constantes de balance (`backend/server/model/balance.constant.js`):
- `operationType`: `MD '1'` (resta), `MC '2'` (suma).
- `description`: `BOND_MC '6'` (cobro de bonos), `BOND_MD '7'` (compra de bonos), `BOND_SALE_MC '8'` (venta de bonos).

## 7. Reglas de negocio

### 7.1 `effectiveDate` del retiro (paridad con `balance.account.js:1523-1545`)

```
d = día del mes de hoy (zona del servidor legacy → usar utcDiff de la sesión)
si 11 <= d <= 25            → effectiveDate = "{año}-{mes}-{últimoDíaDelMes}"
si d >= 26                   → effectiveDate = "{año}-{mes+1}-15"   (dic → enero del año siguiente)
si 1 <= d <= 10              → effectiveDate = "{año}-{mes}-15"
```

Se guarda como string con el mismo formato que el legado: `YYYY-M-D`, sin ceros a la izquierda.

**Corrección:** el legado no incrementa el año al pasar de diciembre a enero, por la comparación `moment().month() + 1 === 13`, que nunca es verdadera. Aquí se incrementa. Ver bug P1.

### 7.2 Catálogo de textos (`backend/server/lib/text.js`)

Se portan literalmente, con sus placeholders: `BOND_ADDED`, `BOND_UPDATED`, `BOND_UPDATED_KEY`, `BOND_REMOVE`, `BOND_BOUND_SOLD`, `BOND_BOUND_PURCHASED`, `BOND_CREATED`, `BOND_BOUND_SUBJECT_PURCHASED`, `BOND_BOUND_MESSAGE_PURCHASED`, `BOND_BOUND_SUBJECT_SOLD`, `BOND_BOUND_MESSAGE_SOLD`, `BOND_BOUND_SUBJECT_TOOK`, `BOND_BOUND_MESSAGE_TOOK`, `BOND_BOUND_MESSAGE_MATURITY`, `BOND_BOUND_MESSAGE_MATURITY_SUCESS`, `BOND_BOUND_SUBJECT_MASSIVE_PURCHASED`, `BOND_BOUND_MESSAGE_MASSIVE_PURCHASED`, `BALANCE_ACCOUNT_WITHDRAW_FOR_BONUS`, `BALANCE_ACCOUNT_WITHDRAW_ADMIN_NOTIFICATION_SUBJECT`, `BALANCE_ACCOUNT_WITHDRAW_ADMIN_NOTIFICATION_FOR_BONUS`, `BALANCE_ACCOUNT_WITHDRAW_CONTENT`, `NEW_WITHDRAWAL`, `WITHDRAW`, `NOT_EXIST`, `FILENAME_REPORT`.

Se **agregan** `BOND_SOLD` y `BOND_BOUND_STATUS_CHANGED`, que el legado referencia pero nunca definió (H2, spec 08).

### 7.3 Handlers nuevos en ms-notification

ms-notification enruta `POST /notify/:group/:notification` a un handler por notificación (`src/api/modules/notify/handlers/notify-<modulo>-<evento>.js`). Hay que crear:

| `notification` | `group` | Params | Origen legacy |
|---|---|---|---|
| `blooming-bondPurchased` | `user-ids` | `userIds, key, amount, accountNumberMasked` | `bondBound.insert` |
| `blooming-bondBulkPurchased` | `user-ids` | `userIds, quantity, total, rows[{key, reference, amount}]` | `bulkPurchaseNotification` |
| `blooming-bondSold` | `user-ids` | los 11 parámetros de `BOND_BOUND_MESSAGE_SOLD` | `bondBound.setStatus` |
| `blooming-bondTook` | `user-ids` | los 11 parámetros de `BOND_BOUND_MESSAGE_TOOK` | `bondBound.setStatus` |
| `blooming-bondMatured` | `user-ids` | `userIds, bonds[{key, type, name}]` | cron `BondBoundCheckMaturity` |
| `blooming-withdrawAdmin` | `to-emails` | `emails, userName, amount, references, date, keys, unitPrice` | admin de `balanceAccount.withdraw` y bulk |

## 8. Ports y dependencias

| Port | Paquete | Implementación |
|---|---|---|
| `TransactionRunnerPort` | `shared_ports` | `shared/infrastructure/mongocc/shared.transaction.adapter.go` (mismo patrón que petty-cash) |
| `BalancePort` | `shared_ports` | `shared/infrastructure/mongocc/balance.*.adapter.go` |
| `NoticePort`, `ActivityPort` | `shared_ports` | `shared/infrastructure/mongocc/notice.add.adapter.go`, `activity.add.adapter.go` |
| `NotificationPort` | `shared_ports` | `shared/infrastructure/notification/notification.client.go` |
| `ExporterPort` | `shared_ports` | `shared/infrastructure/exporter/exporter.client.go` |
| `ProfileReader`, `AgencyReader` | `shared_ports` | `shared/infrastructure/mongocc/profile.*.adapter.go`, `agency.*.adapter.go` |
| `SessionVerifier` | `shared_ports` | `shared/adapters/oauth/session.client.go` |

## 9. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| P1 | `effectiveDate` en diciembre (día ≥ 26) genera `año-1-15` sin incrementar el año. | Corregir | Fecha efectiva en el pasado. | T-00.12 |
| P2 | `getSession` deja `session = false` si el token no existe, y los métodos fallan con TypeError. | Corregir | `VerifySession` responde 401 antes de llegar al caso de uso. | T-00.5 |
| P3 | `withdraw` lee el saldo y después escribe; dos compras simultáneas pueden dejar el saldo negativo. | Corregir | Dinero. Se usa `$inc` condicional. | T-00.11 |
| P4 | `balanceDetail.insert` hace `future.return` antes de `setAmounts`; el saldo se actualiza después de responder. | Corregir | Todo ocurre dentro de la transacción. | T-00.11 |

## 10. Criterios de aceptación

- **CA-00.1** (RF-00.1) — Dado que falta `API_SECRET`, cuando arranca el proceso, entonces termina con código 1 y un log que nombra `API_SECRET` sin mostrar ningún valor.
- **CA-00.2** (RF-00.2) — Dado un request sin `x-secret` o con un valor distinto, cuando llega a cualquier endpoint de negocio, entonces responde `401 UNAUTHORIZED_INVALID_SECRET`. `/_health` responde 200 sin `x-secret`.
- **CA-00.3** (RF-00.3) — Dado un request con `x-secret` válido y sin `x-token`, entonces responde `401 TOKEN_IS_REQUIRED` y no llama a ms-oauth.
- **CA-00.4** (RF-00.3) — Dado un `x-token` que ms-oauth responde con 403, o un ms-oauth que no responde en 3 s, entonces el ms responde `401 TOKEN_INVALID`.
- **CA-00.5** (RF-00.3) — Dado un token válido, cuando el handler lee `GetUserSession(ctx)`, entonces obtiene `userId`, `agencyId`, `agency.isMain`, `countryCode` y `adminBlooming` iguales a los que devolvió ms-oauth.
- **CA-00.6** (RF-00.4) — Dada una sesión con `countryCode: "MX"`, `UtcDiff(ctx)` es `-7`. Con `countryCode: "AR"` (fuera del mapa) es `DEFAULT_UTC_DIFF`.
- **CA-00.7** (RF-00.5) — Dada una sesión con `adminBlooming: false`, cuando llama a un endpoint envuelto en `RequireBloomingAdmin`, entonces responde `403 ERR_FORBIDDEN`.
- **CA-00.8** (RF-00.5) — Dada una sesión con `agency.isMain: true`, `AgencyScope(ctx)` devuelve `nil`. Con `isMain: false` devuelve `agencyId`.
- **CA-00.9** (RF-00.6) — Con `CORS_ORIGINS=https://app.futurapps.com,*.finocompany.com`, un preflight desde `https://x.finocompany.com` recibe `Access-Control-Allow-Origin` y uno desde `https://evil.com` no.
- **CA-00.10** (RF-00.7) — Dado un caso de uso que devuelve un error de Mongo no mapeado, la respuesta es `500` con `ERR_INTERNAL`, y el texto del error solo aparece en el log.
- **CA-00.11** (RF-00.8) — `New()` de cualquier entidad asigna `_id` UUID v4, `createdAt == updatedAt` en UTC e `isRemove == false`.
- **CA-00.12** (RF-00.8) — `StartOfDay(2026-03-10T03:00Z, -5)` devuelve `2026-03-09T05:00Z`, y `DaysBetween` cuenta días calendario en esa zona.
- **CA-00.13** (RF-00.9) — Dado un callback que inserta un documento y después devuelve error, al terminar `RunInTransaction` el documento no existe.
- **CA-00.14** (RF-00.10) — Dada una cuenta activa con `available 1000`, `Withdraw(300, BOND_MD)` deja `available 700`, `global` reducido en 300 y `deferred` sin cambios, y crea un `balanceDetail` con `amount -300` y `balance 700`, un `balanceHistory` con `typeId '3'` y un `balanceOperation` con `details [accountId, detailId]`.
- **CA-00.15** (RF-00.10) — Dada una cuenta con `statusId '2'`, `Withdraw` devuelve `ERR_ACCOUNT_SUSPENDED` y no escribe nada. Con `amount > available` devuelve `ERR_ACCOUNT_WITHDRAW_AMOUNT_MORE_THAN_AVAILABLE`.
- **CA-00.16** (RF-00.10) — Dados dos `Withdraw` concurrentes de 600 sobre `available 1000`, exactamente uno termina bien y el saldo final es 400.
- **CA-00.17** (RF-00.10) — Tabla de `effectiveDate`: 5-mar → `2026-3-15`; 11-mar → `2026-3-31`; 25-feb-2026 → `2026-2-28`; 26-mar → `2026-4-15`; 28-dic-2026 → `2027-1-15`.
- **CA-00.18** (RF-00.10) — `Deposit(500, BOND_SALE_MC)` sobre `available 700` deja `available 1200` y un `balanceDetail` con `operationTypeId '2'`, `amount 500` y `effectiveDate null`.
- **CA-00.19** (RF-00.11) — Dado un ms-notification que responde 500, el llamador de `NotificationPort.Send` no recibe error y queda un log de warning.
- **CA-00.20** (RF-00.11) — `NoticePort.Add` con `Type: bond` inserta `typeId '3'`, `categoryId '1'` y `unread: null`.
- **CA-00.21** (RF-00.12) — `Format("Bono {0} vendido", "AB-3")` da `"Bono AB-3 vendido"`, y `FormatCurrency(1234567.5)` da `"1,234,567.50"`.
- **CA-00.22** (RF-00.13) — Con un request que lleva `x-token: abc`, ninguna línea del log contiene `abc`.
- **CA-00.23** (RF-00.14) — Si ms-exporter responde 201 con `{"type":"Buffer","data":[80,75]}`, `Excel` devuelve `[]byte{80,75}`. Con un 500 devuelve `ERR_EXPORT_FAILED`.
- **CA-00.24** (RF-00.15) — `ProfileReader.GetPublic` nunca devuelve el campo `password`, aunque el documento lo tenga.
- **CA-00.25** (RF-00.16) — Si llega SIGTERM con un request de 2 s en curso, el request se completa con 200, el proceso termina con código 0 y el cron queda detenido.
- **CA-00.26** (RF-00.16) — El binario `healthcheck` termina en 0 si `/_health` responde 200, y en 1 si no hay proceso escuchando en `API_PORT`.
- **CA-00.27** (RF-00.17) — Si `bondBound_domain` importa `bondObject_domain`, `go test ./internal/core/...` falla y nombra los dos paquetes.
- **CA-00.28** (RF-00.18) — `Pagination{Page: 0, Limit: 500}` se normaliza a `{1, 100}`, y `{3, 10}` da `skip 20`.
- **CA-00.29** (RF-00.10) — `GetAccount(accountId, otroUserId)` sobre una cuenta que existe pero es de otro usuario devuelve `nil, nil`.
- **CA-00.30** (RF-00.8, RF-00.19) — `IDFormat` devuelve `uuid` para `3f2b8c1e-9d4a-4f6b-8c2d-1a2b3c4d5e6f` y `meteor` para `tDr6NwSsDe4Lnccp8`. Devuelve `invalid` para `""`, `abc`, un ObjectId de 24 caracteres hex (`64b7f0c2a1e4d3b2c1a0f9e8`), un id de 17 caracteres con `-` y un UUID v1.
- **CA-00.31** (RF-00.19) — Un `GET` a un endpoint con `{id}` = `abc` responde `400 ERR_INVALID_ID`, y el fake del repositorio registra cero llamadas.
- **CA-00.32** (RF-00.19) — Con un fixture de dos documentos en la misma colección, uno con `_id` UUID y otro con `_id` Meteor de 17 caracteres, `FindOneByID` encuentra a los dos, y un `UpdateOne` sobre el legacy conserva su `_id` original.

## 11. Tareas

| ID | Tarea | Tipo | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|
| T-00.1 | Config: `ApiEnv` completo, carga de `.env` y `.env.<ENV>`, validación de requeridas, `.env.example` depurado | impl | RF-00.1 | T-PREP.3 | `cmd/api/config/config.go`, `cmd/api/config/environments.go`, `.env.example` |
| T-00.2 | Composition root: `serverDeps`, `InitServer`, conexión mongocc, `nexus.Server` con `/api/v1`, `cmd/main.go` | impl | RF-00.1, RF-00.7 | T-00.1 | `cmd/main.go`, `cmd/api/server/server.go`, `server.config.go`, `server.deps.go` |
| T-00.3 | Middleware `RequestLog` sin headers ni body | impl | RF-00.13 | T-00.2 | `internal/core/shared/middlewares/requestLog.middleware.go` |
| T-00.4 | Middleware `ValidateApiSecret` con comparación en tiempo constante | impl | RF-00.2 | T-00.2 | `internal/core/shared/middlewares/validateApiSecret.middleware.go` |
| T-00.5 | `session_domain` (`UserSession`, claves de contexto), cliente ms-oauth y middleware `VerifySession` | impl | RF-00.3 | T-00.2 | `internal/core/session/domain/`, `internal/core/shared/adapters/oauth/session.client.go`, `shared/middlewares/verifySession.middleware.go` |
| T-00.6 | Resolución de `utcDiff` (sesión → país → `DEFAULT_UTC_DIFF`) | impl | RF-00.4 | T-00.5 | `internal/core/session/domain/session.utcDiff.go` |
| T-00.7 | `RequireBloomingAdmin`, `IsOwnerOrAdmin` y `AgencyScope` | impl | RF-00.5 | T-00.5 | `internal/core/shared/middlewares/requireBloomingAdmin.middleware.go`, `internal/core/shared/application/shared.authorization.go` |
| T-00.8 | CORS configurable con soporte de wildcard | impl | RF-00.6 | T-00.2 | `cmd/api/server/server.config.go`, `internal/core/shared/middlewares/cors.go` |
| T-00.9 | Helpers compartidos: `NewID`, `Now`, `Round2`, `StartOfDay`, `DaysBetween`, `MaskAccountNumber`, `RandomAlnum`, `Pagination` | impl | RF-00.8, RF-00.18 | T-PREP.3 | `internal/core/shared/domain/`, `internal/core/shared/adapters/utils/` |
| T-00.10 | `TransactionRunnerPort` y su adapter mongocc | impl | RF-00.9 | T-00.2 | `internal/core/shared/ports/shared.ports.go`, `shared/infrastructure/mongocc/shared.transaction.adapter.go` |
| T-00.11 | `BalancePort`: `GetAccount` (con control de dueño), `Withdraw` y `Deposit` con escrituras en detail, history, operation y `$inc` condicional | impl | RF-00.10 | T-00.9, T-00.10, T-00.12 | `internal/core/shared/infrastructure/mongocc/balance.withdraw.adapter.go`, `balance.deposit.adapter.go` |
| T-00.12 | Función pura `EffectiveDate(now, utcDiff)` con el cambio de año corregido | impl | RF-00.10 | T-00.9 | `internal/core/shared/domain/balance.effectiveDate.go` |
| T-00.13 | `NoticePort` y `ActivityPort` (inserts en `notice` y `activity`) | impl | RF-00.11 | T-00.9 | `internal/core/shared/infrastructure/mongocc/notice.add.adapter.go`, `activity.add.adapter.go` |
| T-00.14 | `NotificationPort`: cliente HTTP de ms-notification, best-effort | impl | RF-00.11 | T-00.5 | `internal/core/shared/infrastructure/notification/notification.client.go` |
| T-00.15 | **(repo ms-notification)** Handlers `blooming-*` de §7.3 | impl | RF-00.11 | — | `ms-notification/src/api/modules/notify/handlers/notify-blooming-*.js` |
| T-00.16 | `ExporterPort`: cliente HTTP de ms-exporter con conversión del Buffer | impl | RF-00.14 | T-00.5 | `internal/core/shared/infrastructure/exporter/exporter.client.go` |
| T-00.17 | `ProfileReader` y `AgencyReader` con proyecciones seguras | impl | RF-00.15 | T-00.2 | `internal/core/shared/infrastructure/mongocc/profile.getPublic.adapter.go`, `agency.symbolCurrency.adapter.go` |
| T-00.18 | Catálogo `Text`, `Format` y `FormatCurrency` | impl | RF-00.12 | T-PREP.3 | `internal/core/shared/domain/shared.text.go`, `shared.format.go` |
| T-00.19 | Graceful shutdown y binario `cmd/healthcheck` (`package main`, puerto de `API_PORT`) | impl | RF-00.16 | T-00.2 | `cmd/api/server/server.go`, `cmd/healthcheck/main.go` |
| T-00.20 | Regla de dependencias entre dominios en `architecture_test.go` (el test es la implementación de la regla) | impl | RF-00.17 | T-PREP.3 | `internal/core/architecture_test.go` |
| T-00.21 | Tests de middlewares: secret, sesión, admin, scope de agencia, log, utcDiff | test | CA-00.2, CA-00.3, CA-00.4, CA-00.5, CA-00.6, CA-00.7, CA-00.8, CA-00.22 | T-00.3, T-00.4, T-00.5, T-00.6, T-00.7 | `internal/core/shared/middlewares/*_test.go`, `internal/core/session/domain/*_test.go` |
| T-00.22 | Tests de integración: transacción, `BalancePort` (incluida la concurrencia), notice, activity y readers | test | CA-00.13, CA-00.14, CA-00.15, CA-00.16, CA-00.18, CA-00.20, CA-00.24, CA-00.29 | T-00.10, T-00.11, T-00.13, T-00.17 | `internal/core/shared/infrastructure/mongocc/*_integration_test.go` |
| T-00.23 | Tests unitarios de helpers, `EffectiveDate`, textos y paginación | test | CA-00.11, CA-00.12, CA-00.17, CA-00.21, CA-00.28 | T-00.9, T-00.12, T-00.18 | `internal/core/shared/domain/*_test.go` |
| T-00.24 | Tests de los clientes HTTP (ms-notification, ms-exporter) con `httptest` | test | CA-00.19, CA-00.23 | T-00.14, T-00.16 | `internal/core/shared/infrastructure/notification/*_test.go`, `exporter/*_test.go` |
| T-00.25 | Tests de arranque, CORS, errores, shutdown y healthcheck, más el caso negativo de la regla de dependencias (paquete de fixture con un import prohibido) | test | CA-00.1, CA-00.9, CA-00.10, CA-00.25, CA-00.26, CA-00.27 | T-00.1, T-00.2, T-00.8, T-00.19, T-00.20 | `cmd/api/server/*_test.go`, `cmd/healthcheck/main_test.go` |
| T-00.26 | `IDFormat`/`IsValidID` en `shared_domain` y helper de handlers `PathID` (valida y responde `400 ERR_INVALID_ID`) | impl | RF-00.19, RF-00.8 | T-00.9, T-00.2 | `internal/core/shared/domain/shared.id.go`, `cmd/api/handlers/shared/handlers.pathId.go` |
| T-00.27 | Tests de formatos de ID (tabla), del rechazo en el handler sin llegar al repositorio y de la lectura y actualización de documentos con los dos formatos | test | CA-00.30, CA-00.31, CA-00.32 | T-00.26, T-00.10 | `internal/core/shared/domain/shared.id_test.go`, `cmd/api/handlers/shared/handlers.pathId_test.go`, `internal/core/shared/infrastructure/mongocc/shared.ids_integration_test.go` |

## 12. Fuera de alcance y preguntas abiertas

- **Fuera de alcance:** login y creación de sesión (ms-oauth), y la lógica de saldo que no sea de bonos (depósitos de nómina, anulaciones, retiros manuales).
- **Pregunta 1:** ¿ms-oauth puede agregar `gmt`/`utcDiff` a la sesión? Mientras no lo haga, se usa el mapa por país (RF-00.4).
- **Pregunta 2:** ¿qué puerto usará el servicio? El esqueleto copió `25072` de petty-cash.
- **Pregunta 3:** `blooming-withdrawAdmin` usa el grupo `to-emails`, que en ms-notification busca usuarios por email. Si `bonds@finocompany.com` no es un usuario, hace falta un grupo que envíe a direcciones crudas.
