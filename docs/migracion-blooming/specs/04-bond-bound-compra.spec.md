# Spec 04 — Compra de bonos (`bondBound`)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondBound` |
| Handlers | `cmd/api/handlers/bondBounds` |
| Colecciones | `bondBound`, `bondKeyCounter` (nueva); escritura de saldo vía `BalancePort` |
| Depende de | Spec 00 (`BalancePort`, `TransactionRunner`, notificaciones), spec 02 (`bondObject`), spec 08 (`BondHistoryPort`) |

## 1. Resumen y alcance

El inversionista compra uno o varios bonos (`bondBound`) de un producto (`bondObject`) y paga con una de sus cuentas de saldo.

Cada bono comprado guarda una **copia** de los términos del producto: monto, plazo, rendimiento y `earlyCashTerm`. Nace en estado `initiated ('1')`.

Esta spec también define la entidad `BondBound`, su repositorio base y su estado. Las specs 03, 05, 06, 07 y 10 los reutilizan.

## 2. Origen legacy

Archivo: `backend/server/model/bond.bound.js`.

| Operación legacy | Línea | Destino |
|---|---|---|
| `bondBound.insert` | 431 | Caso de uso interno `createBounds` dentro de RF-04.1 y RF-04.2 (no se expone) |
| `bondBound.purchase` | 674 | `POST /bond-bounds/purchase` (RF-04.1) |
| `bondBound.bulkPurchase` | 1660 | `POST /bond-bounds/bulk-purchase` (RF-04.2) |
| `bulkPurchaseNotification` | 1559 | RF-04.7 |
| `Model.bondBound.status` | 18 | Enum de estados (§6) |

## 3. Requisitos funcionales

- **RF-04.1** — Compra individual.
  - Recibe `{accountId, bondObjectId}`.
  - Valida en este orden (§5.1), y cualquier fallo responde sin escribir nada.
  - En **una transacción** (RF-04.5) crea un `bondBound`, ejecuta un `BalancePort.Withdraw` por el `amount` del bono con `descriptionId '7'` (BOND_MD) y `comment` `Compra de Bono "<key>" (<boundId>)`, y registra el historial.
  - Después del commit envía las notificaciones de RF-04.6.
  - Responde `201 {bondBoundId, key, accountAvailable}`.
- **RF-04.2** — Compra masiva.
  - Recibe `{accountId, bondObjectId, quantity}`, con `1 <= quantity <= BULK_PURCHASE_MAX_QUANTITY`; si no, `ERR_INVALID_QUANTITY`.
  - Hace las mismas validaciones de §5.1, usando `total = amount × quantity` para el chequeo de saldo.
  - En **una transacción** reserva `quantity` claves consecutivas (RF-04.3) y crea `quantity` bonos, cada uno con su `Withdraw` (un `balanceDetail` por bono, como el legado) y su historial.
  - Después del commit envía las notificaciones de RF-04.7.
  - Responde `201 {items: [{_id, key, reference}], total, accountAvailable}`.
- **RF-04.3** — Clave del bono.
  - Formato `key = bondObject.key + "-" + seq`, donde `seq` es un contador **por `bondTypeId`** (paridad: el legado numeraba con `count(bondBound{bondTypeId}) + 1`).
  - El contador vive en `bondKeyCounter {_id: bondTypeId, seq}` y avanza con `findOneAndUpdate($inc: {seq: n}, upsert)` dentro de la transacción. Corrige la carrera de B8.
  - Antes de habilitar la compra en Go, un script de siembra inicializa cada contador con el `count` actual de `bondBound` por tipo, incluidos los borrados (paridad).
  - Cada bono guarda además `keySeq` (int), con índice único parcial `{bondTypeId, keySeq}` donde `keySeq` existe.
- **RF-04.4** — Snapshot y fechas. El bono se construye con la función pura `NewBoundFromObject(obj, buyer, seq, token, now)`.
  - `obj` es un `ObjectSnapshot` del producto y `buyer` el usuario y la agencia de la sesión. Los arma el caso de uso, porque `bondBound_domain` no importa los dominios de `bondObject` ni de la sesión.
  - Copia del comprador `userId` y `agencyId`, y del producto `bondObjectId`, `bondTypeId`, `amount`, `maturityTimeInDays`, `maturityRewardsPercentage` y `earlyCashTerm` (los cinco campos).

  Además asigna:
  - `key = obj.key + "-" + seq` y `keySeq = seq` (RF-04.3);
  - `statusId '1'` y el `token` que recibe, que el caso de uso genera con `RandomAlnum(6)`;
  - `initialDate = now` (UTC) y `maturityDate = now + maturityTimeInDays × 24h`, en UTC y con la hora exacta. Corrige B13 (`hh` de 12 horas).
- **RF-04.5** — Atomicidad. Dentro de `RunInTransaction` se ejecutan: la reserva de claves, el `InsertMany` de bonos, cada `Withdraw` y los `BondHistory.Add`.
  - Si cualquiera falla (por ejemplo, el saldo cambió entre la validación y el retiro), se revierte todo y se responde el error del paso que falló.
  - Corrige B7 (bono creado sin cobro) y la compra masiva parcial.
- **RF-04.6** — Notificaciones de la compra individual (post-commit y best-effort). Paridad con `bondBound.insert` más las de `balanceAccount.withdraw`:
  1. `Notice` tipo `bond` para el comprador: título `BOND_BOUND_SUBJECT_PURCHASED[key, amount]` y mensaje `BOND_BOUND_MESSAGE_PURCHASED[first_name, last_name, key, amount, cuentaEnmascarada]`.
  2. Email `notify-blooming-bondPurchased` al comprador.
  3. `Notice` tipo `balance` para el comprador: título `NEW_WITHDRAWAL` y mensaje `BALANCE_ACCOUNT_WITHDRAW_CONTENT[nombre, cuentaEnmascarada, amount]`.
  4. Email `notify-blooming-withdrawUser` al comprador, como una compra de 1 bono.
  5. Email `notify-blooming-withdrawAdmin` a `BLOOMING_ADMIN_EMAILS`, como una compra de 1 bono.
- **RF-04.7** — Notificaciones de la compra masiva (post-commit y best-effort). Paridad con `bulkPurchaseNotification`:
  1. `Notice` tipo `bond` con `BOND_BOUND_SUBJECT_MASSIVE_PURCHASED[quantity, total]` y `BOND_BOUND_MESSAGE_MASSIVE_PURCHASED[nombre, quantity, total, amount, cuentaEnmascarada, tabla]`. La tabla tiene una fila por bono: `# | key | reference | amount`.
  2. Email `notify-blooming-bondBulkPurchased`.
  3. `Notice` tipo `balance` con `NEW_WITHDRAWAL` y `BALANCE_ACCOUNT_WITHDRAW_FOR_BONUS[nombre, cuentaEnmascarada, total, keys, amount]`.
  4. Email `notify-blooming-withdrawUser` al comprador.
  5. Email `notify-blooming-withdrawAdmin` con las referencias, la fecha, las claves y el precio unitario.
- **RF-04.8** — Persistencia de `bondBound`.
  - Entidad, enum de estados (§6) y repositorio base: `InsertMany`, `FindOneByID`, `UpdateOne` condicional por estado y `EnsureIndexes`. Las specs 03, 05, 06, 07 y 10 lo reutilizan.
  - El decoder acepta documentos legacy: `_id` Meteor, numéricos como string (`FlexNumber`) y sin `keySeq`.

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| POST | `/bond-bounds/purchase` | usuario | RF-04.1 |
| POST | `/bond-bounds/bulk-purchase` | usuario | RF-04.2 |

**`POST /bond-bounds/purchase`**: `{"accountId": "acc1", "bondObjectId": "obj1"}` → `201`

```json
{ "bondBoundId": "6f1c…", "key": "BP12-58", "accountAvailable": 4000 }
```

**`POST /bond-bounds/bulk-purchase`**: `{"accountId": "acc1", "bondObjectId": "obj1", "quantity": 3}` → `201`

```json
{ "items": [ { "_id": "…", "key": "BP12-59", "reference": "A7k2…" }, { "…": "…" }, { "…": "…" } ],
  "total": 3000, "accountAvailable": 1000 }
```

**Cambio de contrato respecto al legado:**
- `purchase` respondía `{success, code, message, data: {data: {accountAvailable, bondBoundId}}}` (doble `data`).
- `bulkPurchase` respondía `{success, code, data: [ids]}`.
- Los errores eran `{success: false, code: -20xx, message}` con status 200; ahora son status HTTP con sentinel.

**Errores**

| Sentinel | HTTP | Código legacy | Cuándo |
|---|---|---|---|
| `ERR_INVALID_ID` | 400 | — | Un id de path o body no es UUID v4 ni Meteor de 17 caracteres (spec 00, RF-00.19) |
| `ERR_ACCOUNT_ID_REQUIRED` / `ERR_BOND_OBJECT_ID_REQUIRED` | 400 | — | Falta un id |
| `ERR_INVALID_QUANTITY` | 400 | — | `quantity` fuera de `1..BULK_PURCHASE_MAX_QUANTITY` |
| `ERR_ACCOUNT_NOT_FOUND` | 404 | -700 | La cuenta no existe, no es del usuario o es de otra agencia |
| `ERR_BOND_OBJECT_NOT_FOUND` | 404 | -11 | El producto no existe o está borrado |
| `ERR_BOND_PRIVATE_NOT_AUTHORIZED` | 403 | — | Bono privado y el usuario no está en `authorizedUserIds` |
| `ERR_ACCOUNT_WITHOUT_FUNDS` | 422 | -2010 | `available <= 0` |
| `ERR_ACCOUNT_NOT_ACTIVE` | 422 | -2011 | `statusId != '1'` |
| `ERR_BOND_INACTIVE` | 422 | -2020 | `active == false` |
| `ERR_INSUFFICIENT_AMOUNT` | 422 | -2005 | `available < amount × quantity` |
| `ERR_BOND_NOT_AVAILABLE_BY_DATE` | 422 | -2021 | `saleInitDate > now` |
| `ERR_BOND_SALE_ENDED` | 422 | — | `saleEndDate != null && saleEndDate < now` |
| Errores de `BalancePort.Withdraw` | 422 | -7xx | El saldo cambió entre la validación y el retiro (spec 00) |

## 5. Reglas de negocio

### 5.1 Validaciones, en orden

1. `accountId` y `bondObjectId` no vacíos; `quantity` válido (solo en la masiva).
2. `account = BalancePort.GetAccount(accountId, session.userId)`. Si es `nil` o `account.agencyId != session.agencyId`, `ERR_ACCOUNT_NOT_FOUND`. Corrige B18: el legado no validaba el dueño en la compra masiva.
3. `obj = BondObjectApplicationPorts.FindVisibleForPurchase(bondObjectId)`, con `isRemove: false` y **sin** filtro de agencia (paridad: el legado lo comentó a propósito para permitir comprar bonos de otra agencia). Si es `nil`, `ERR_BOND_OBJECT_NOT_FOUND`.
4. Si `obj.isPrivateBond` y `session.userId ∉ obj.authorizedUserIds`, `ERR_BOND_PRIVATE_NOT_AUTHORIZED` (B9).
5. Si `account.available <= 0`, `ERR_ACCOUNT_WITHOUT_FUNDS`.
6. Si `account.statusId != '1'`, `ERR_ACCOUNT_NOT_ACTIVE`.
7. Si `!obj.active`, `ERR_BOND_INACTIVE`.
8. Si `account.available < obj.amount × quantity`, `ERR_INSUFFICIENT_AMOUNT`.
9. Si `obj.saleInitDate > now`, `ERR_BOND_NOT_AVAILABLE_BY_DATE`.
10. Si `obj.saleEndDate < now`, `ERR_BOND_SALE_ENDED` (B9).

Los pasos 5–9 siguen el orden de `bondBound.purchase`. La compra masiva del legado validaba en otro orden (6, 5, 7, 9, 8). Se unifica: solo cambia cuál error se ve cuando fallan varios a la vez.

### 5.2 Transacción

```
RunInTransaction(ctx, func(tx) {
  seqStart = KeyCounter.Reserve(tx, obj.bondTypeId, quantity)        // devuelve el primer seq reservado
  bounds   = [NewBoundFromObject(snapshot(obj), buyer(session), seqStart+i, RandomAlnum(6), now) para i en 0..quantity-1]
  Repo.InsertMany(tx, bounds)
  para cada b en bounds:
     mov = BalancePort.Withdraw(tx, {accountId, userId: session.userId, agencyId: session.agencyId,
                                     descriptionId: '7', amount: b.amount,
                                     comment: `Compra de Bono "${b.key}" (${b._id})`})
     BondHistory.Add(tx, {relatedId: b._id, typeId: '2', text: 'BOND_BOUND_SOLD'})
     BondHistory.Add(tx, {relatedId: obj._id, typeId: '1', text: 'BOND_SOLD', params: [b._id]})
})
```

El historial va **dentro** de la transacción, porque es dato de auditoría que consume el gateway (`B_BondBound_Audit_*`). No es una notificación.

## 6. Modelo de datos

```go
type BondBoundStatus string
const (
    StatusInitiated BondBoundStatus = "1" // Iniciado
    StatusFinalized BondBoundStatus = "2" // Madurado
    StatusBlocked   BondBoundStatus = "3" // Bloqueado
    StatusUnbound   BondBoundStatus = "4" // Desligado
    StatusSold      BondBoundStatus = "5" // Vendido
    StatusTook      BondBoundStatus = "6" // Canjeado
)

type BondBound struct {
    ID                        string                      `json:"_id" bson:"_id"`
    BondObjectID              string                      `json:"bondObjectId" bson:"bondObjectId"`
    BondTypeID                string                      `json:"bondTypeId" bson:"bondTypeId"`
    Key                       string                      `json:"key" bson:"key"`
    KeySeq                    *int                        `json:"-" bson:"keySeq,omitempty"`
    Token                     string                      `json:"token" bson:"token"`
    UserID                    string                      `json:"userId" bson:"userId"`
    AgencyID                  string                      `json:"agencyId" bson:"agencyId"`
    StatusID                  BondBoundStatus             `json:"statusId" bson:"statusId"`
    Amount                    float64                     `json:"amount" bson:"amount"`
    MaturityTimeInDays        int                         `json:"maturityTimeInDays" bson:"maturityTimeInDays"`
    MaturityRewardsPercentage float64                     `json:"maturityRewardsPercentage" bson:"maturityRewardsPercentage"`
    EarlyCashTerm             shared_domain.EarlyCashTerm `json:"earlyCashTerm" bson:"earlyCashTerm"`
    InitialDate               time.Time                   `json:"initialDate" bson:"initialDate"`
    MaturityDate              time.Time                   `json:"maturityDate" bson:"maturityDate"`
    SoldDate                  *time.Time                  `json:"soldDate,omitempty" bson:"soldDate,omitempty"`
    TookDate                  *time.Time                  `json:"tookDate,omitempty" bson:"tookDate,omitempty"`
    Matured                   bool                        `json:"matured,omitempty" bson:"matured,omitempty"`
    CreatedAt, UpdatedAt      time.Time                   // createdAt, updatedAt
    IsRemove                  bool                        `json:"isRemove" bson:"isRemove"`
}

type BondKeyCounter struct { // colección bondKeyCounter
    ID  string `bson:"_id"` // bondTypeId
    Seq int    `bson:"seq"`
}
```

**Índices de `bondBound`:**
- `{userId: 1, isRemove: 1, statusId: 1, createdAt: -1}`
- `{agencyId: 1, isRemove: 1, statusId: 1, createdAt: -1}`
- `{bondObjectId: 1, isRemove: 1, statusId: 1}`
- `{statusId: 1, maturityDate: 1}` (spec 10)
- único parcial `{bondTypeId: 1, keySeq: 1}` con `partialFilterExpression: {keySeq: {$exists: true}}`

`key` **no** lleva índice único global: el legado ya pudo generar duplicados por B8. Ver la tarea de auditoría BLM-0079.

## 7. Ports y dependencias

- `BondBoundApplicationPorts`: `Purchase`, `BulkPurchase`. El resto se define en las specs 03, 05, 06, 07 y 10.
- `BondBoundMongoccRepositoryPorts`: `InsertMany`, `FindOneByID`, `UpdateStatusIf`, `EnsureIndexes`, más los de las otras specs.
- `KeyCounterPort.Reserve(ctx, bondTypeID, n) (start int, err error)`.
- `BondObjectApplicationPorts.FindVisibleForPurchase` (spec 02).
- `BalancePort.GetAccount` y `Withdraw`, `TransactionRunnerPort`, `NoticePort`, `NotificationPort`, `ProfileReader` (spec 00).
- `BondHistoryPort.Add` (spec 08).

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| B7 | `insert` no verifica el resultado del retiro: se puede crear un bono sin cobro. Sin transacción ni idempotencia. | Corregir | Dinero. | BLM-0081 |
| B8 | La clave se arma con `count(bondTypeId) + 1`: hay carrera bajo compras concurrentes, sin índice único, y cuenta los borrados. | Corregir (contador atómico); se mantiene contar los borrados en la siembra | Claves duplicadas. | BLM-0079 |
| B9 | No se valida `isPrivateBond`/`authorizedUserIds` ni `saleEndDate`, ni se hace null-check de la cuenta y el producto (crash). | Corregir | Venta de bonos privados o vencidos, y llamadas que no responden. | BLM-0081 |
| B13 | `maturityDate` usa `format('YYYY-MM-DD hh:mm:ss')` (12 h, sin am/pm): las compras de la tarde vencen 12 h antes. | Corregir | Fecha de vencimiento incorrecta. | BLM-0080 |
| B18 | `bulkPurchase` busca la cuenta solo por `agencyId` y retira con `userId: account.userId`: se puede comprar debitando la cuenta de otro usuario de la misma agencia. | Corregir | Seguridad y dinero. | BLM-0082 |
| B19 | La compra masiva crea los bonos y retiros uno por uno, y si uno falla quedan compras parciales; además crashea si un retiro devuelve error (`withdrawResponse.data` es undefined). | Corregir | Dinero; transacción única. | BLM-0082 |
| B20 | El retiro de la compra individual no tiene `comment`. | Corregir | Trazabilidad en el estado de cuenta: se usa el mismo `comment` que en la masiva. | BLM-0081 |

## 9. Criterios de aceptación

- **CA-04.1** (RF-04.1, RF-04.4) — Dada una cuenta propia activa con `available 5000` y un producto activo de `amount 1000`, `POST /bond-bounds/purchase` responde `201` con `accountAvailable 4000`. Además:
  - hay un `bondBound` con `statusId '1'`, `userId` de la sesión y los términos copiados;
  - hay un `balanceDetail` con `amount -1000` y `descriptionId '7'`;
  - hay dos `bondHistory`: `BOND_BOUND_SOLD` (`typeId '2'`) y `BOND_SOLD` (`typeId '1'`, `params [boundId]`).
- **CA-04.2** (RF-04.1) — Tabla de validaciones; en todos los casos no se escribe nada:
  - `available 0` → `422 ERR_ACCOUNT_WITHOUT_FUNDS`;
  - cuenta `statusId '2'` → `ERR_ACCOUNT_NOT_ACTIVE`;
  - producto inactivo → `ERR_BOND_INACTIVE`;
  - `available 500` con `amount 1000` → `ERR_INSUFFICIENT_AMOUNT`;
  - `saleInitDate` mañana → `ERR_BOND_NOT_AVAILABLE_BY_DATE`;
  - `saleEndDate` ayer → `ERR_BOND_SALE_ENDED`.
- **CA-04.3** (RF-04.1, RF-04.2) — Una compra individual o masiva con el `accountId` de otro usuario de la misma agencia responde `404 ERR_ACCOUNT_NOT_FOUND`, y el saldo de esa cuenta no cambia.
- **CA-04.4** (RF-04.1) — Un producto privado sin el usuario en `authorizedUserIds` responde `403 ERR_BOND_PRIVATE_NOT_AUTHORIZED`. Un producto de otra agencia, activo y público, se puede comprar.
- **CA-04.5** (RF-04.2) — Con `quantity 3`, `amount 1000` y el contador del tipo en 58, se crean los bonos `BP12-59`, `BP12-60` y `BP12-61`, con tres `balanceDetail` de `-1000`. `available` baja 3000 y la respuesta trae 3 `items` con su `reference`.
- **CA-04.6** (RF-04.2) — `quantity 0` o `quantity > BULK_PURCHASE_MAX_QUANTITY` responde `400 ERR_INVALID_QUANTITY`. Con `available 2500` y `quantity 3` de 1000 responde `422 ERR_INSUFFICIENT_AMOUNT`.
- **CA-04.7** (RF-04.3) — Si la siembra deja el contador de un tipo en 7, la siguiente compra genera `<prefijo>-8`. Dos compras concurrentes del mismo tipo obtienen `seq` distintos.
- **CA-04.8** (RF-04.4) — Una compra a `2026-10-13T20:30:00Z` de un producto de 365 días guarda `maturityDate 2027-10-13T20:30:00Z` (sin el corrimiento de 12 h).
- **CA-04.9** (RF-04.5) — Si en una compra masiva de 3 el tercer `Withdraw` falla (fake que devuelve error), no queda ningún `bondBound`, `balanceDetail` ni `bondHistory`, el contador no avanza y el saldo queda igual.
- **CA-04.10** (RF-04.5) — Con `available 1500`, dos compras concurrentes de 1000: exactamente una responde `201`, la otra un `422`, y el saldo final es 500.
- **CA-04.11** (RF-04.6) — Una compra individual exitosa crea un `notice` `typeId '3'` y uno `typeId '1'`, y envía `notify-blooming-bondPurchased`, `notify-blooming-withdrawUser` y `notify-blooming-withdrawAdmin`. Si ms-notification responde 500, la compra igual responde `201`.
- **CA-04.12** (RF-04.7) — Una compra masiva de 3 genera un solo `notice` de bono con una tabla de 3 filas que contienen las `reference` de los `balanceDetail`, un `notice` de balance con las 3 claves y un email de admin.
- **CA-04.13** (RF-04.8) — `EnsureIndexes` crea los índices de §6 y es idempotente. Un documento legacy con `_id` de 17 caracteres y `amount: "1000"` (string) se decodifica sin error.
- **CA-04.14** (RF-04.1, RF-04.3, RF-04.4) — Compra de un bondObject legacy (`_id` y `bondTypeId` de 17 caracteres) con una cuenta legacy: el `bondBound` creado tiene `_id` UUID y guarda `bondObjectId` y `bondTypeId` de 17 caracteres tal cual, y el contador `bondKeyCounter` usa el `bondTypeId` legacy como `_id`.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0075 | Dominio: `BondBound`, `BondBoundStatus` (con nombres e `IsValid`), `BondKeyCounter`, DTOs `PurchaseParams`/`BulkPurchaseParams`, errores y `httpStatus` | impl | F5 | RF-04.1, RF-04.2, RF-04.8 | BLM-0028, BLM-0009 | `internal/core/bondBound/domain/*.go` |
| BLM-0076 | Ports: `BondBoundApplicationPorts`, `BondBoundMongoccRepositoryPorts`, `KeyCounterPort` | impl | F5 | RF-04.1, RF-04.2, RF-04.3, RF-04.8 | BLM-0075 | `internal/core/bondBound/ports/bondBound.ports.go` |
| BLM-0077 | Repositorio base: `InsertMany`, `FindOneByID`, `UpdateStatusIf` y `EnsureIndexes` (§6) | impl | F5 | RF-04.8 | BLM-0076 | `internal/core/bondBound/infrastructure/mongocc/bondBound.mongocc.repository.go`, `bondBound.insertMany.adapter.go`, `bondBound.findOneById.adapter.go`, `bondBound.updateStatusIf.adapter.go`, `bondBound.ensureIndexes.adapter.go` |
| BLM-0079 | `KeyCounterPort.Reserve`, script de siembra de contadores y auditoría de claves duplicadas existentes | impl | F5 | RF-04.3 | BLM-0077 | `internal/core/bondBound/infrastructure/mongocc/bondBound.keyCounter.adapter.go`, `cmd/tools/seedkeycounters/main.go` |
| BLM-0080 | Función pura `NewBoundFromObject` (snapshot, `maturityDate` exacta, `token`) | impl | F5 | RF-04.4 | BLM-0075 | `internal/core/bondBound/domain/bondBound.new.go` |
| BLM-0081 | Caso de uso `Purchase`: validaciones de §5.1 y transacción de §5.2 | impl | F5 | RF-04.1, RF-04.5 | BLM-0077, BLM-0079, BLM-0080, BLM-0010, BLM-0012, BLM-0055, BLM-0035 | `internal/core/bondBound/application/bondBound.purchase.go`, `bondBound.validatePurchase.go`, `bondBound.application.go` (dependencias y sesión) |
| BLM-0082 | Caso de uso `BulkPurchase`: cantidad, validaciones compartidas y transacción única | impl | F5 | RF-04.2, RF-04.5 | BLM-0081 | `internal/core/bondBound/application/bondBound.bulkPurchase.go`, `bondBound.application.go` (tope de la compra masiva y aserción del port) |
| BLM-0083 | Notificaciones post-commit de la compra individual y la masiva (armado de textos y tabla) | impl | F5 | RF-04.6, RF-04.7 | BLM-0081, BLM-0082, BLM-0013, BLM-0014, BLM-0018, BLM-0015 | `internal/core/bondBound/application/bondBound.purchase.notify.go`, `bondBound.application.go` (dependencias de notificación), `bondBound.purchase.go` y `bondBound.bulkPurchase.go` (llamada después del commit) |
| BLM-0084 | Handlers `purchase` y `bulk-purchase`, y `server.bondBound.go` (inicialización del dominio y registro) | impl | F5 | RF-04.1, RF-04.2 | BLM-0081, BLM-0082, BLM-0026 | `cmd/api/handlers/bondBounds/bondBounds.handler.go`, `bondBounds.purchase.post.go`, `bondBounds.bulkPurchase.post.go`, `cmd/api/server/server.bondBound.go`, `server.go` (inicialización y registro), `server.deps.go` y `server.config.go` (aviso, notificación y utilidades compartidas) |
| BLM-0087 | Tests unitarios de validaciones (tabla y orden), de `NewBoundFromObject` y de los handlers | test | F5 | CA-04.2, CA-04.3, CA-04.4, CA-04.6, CA-04.8 | BLM-0084 | `internal/core/bondBound/application/bondBound.purchase_test.go`, `internal/core/bondBound/domain/bondBound.new_test.go` |
| BLM-0088 | Tests de integración: compra completa, rollback, concurrencia, contador e índices | test | F5 | CA-04.1, CA-04.5, CA-04.7, CA-04.9, CA-04.10, CA-04.13, CA-04.14 | BLM-0084 | `internal/core/bondBound/application/bondBound.purchase_integration_test.go`, `internal/core/bondBound/infrastructure/mongocc/*_integration_test.go` |
| BLM-0089 | Tests de las notificaciones (textos, tabla y tolerancia a fallos) | test | F5 | CA-04.11, CA-04.12 | BLM-0083 | `internal/core/bondBound/application/bondBound.purchase.notify_test.go` |

## 11. Fuera de alcance y preguntas abiertas

- **Convivencia:** mientras Meteor siga vendiendo, generará claves con `count + 1` y puede chocar con el contador de Go. La compra se habilita en Go y se deshabilita en Meteor **en el mismo corte** (ver `tasks.md`, BLM-0135).
- **Pregunta:** ¿se quiere un header `Idempotency-Key` para evitar compras dobles por doble clic? El legado no lo tiene.
- **Pregunta:** ¿el tope por defecto de la compra masiva (100) es razonable?
- **Pregunta:** ¿se mantiene la compra de bonos de otra agencia, como dice el comentario del legado?
