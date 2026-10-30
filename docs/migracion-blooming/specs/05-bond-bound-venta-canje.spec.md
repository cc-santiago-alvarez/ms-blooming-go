# Spec 05 — Venta anticipada y canje (`bondBound`)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondBound` |
| Handlers | `cmd/api/handlers/bondBounds` |
| Colecciones | `bondBound`; escritura de saldo vía `BalancePort` |
| Depende de | Spec 00, spec 04 (entidad y repositorio), spec 08 (`BondHistoryPort`), spec 09 (cuenta seleccionada) |

## 1. Resumen y alcance

El inversionista puede salir de un bono:
- **Venta anticipada:** el bono está `initiated ('1')`; cobra la inversión más el interés acumulado con la penalidad de `earlyCashTerm`.
- **Canje:** el bono está `finalized ('2')`, es decir, maduró; cobra la inversión más el interés completo.

Antes de vender, el front pide una **cotización** (`preSale`).

El dinero se acredita en la cuenta seleccionada en `bondUserSetting` (spec 09).

Es la spec con más correcciones de dinero y seguridad.

## 2. Origen legacy

Archivo: `backend/server/model/bond.bound.js`.

| Operación legacy | Línea | Destino |
|---|---|---|
| `bondBound.preSale` | 746 | `GET /bond-bounds/{id}/pre-sale` (RF-05.1, RF-05.2) |
| `bondBound.sale` | 816 | `POST /bond-bounds/{id}/sale` (RF-05.3) |
| `bondBound.setStatus` | 541 | Lógica interna de RF-05.3 a RF-05.7 (no se expone) |

## 3. Requisitos funcionales

- **RF-05.1** — Cotización.
  - Carga el bono; si no existe o está fuera del alcance de agencia de la sesión (`AgencyScope`), `ERR_BOND_BOUND_NOT_FOUND` (corrige B38). Si no es del usuario y el usuario no es admin, `ERR_FORBIDDEN`.
  - Si el bono no se puede vender por su estado (§5.3), devuelve el error correspondiente.
  - Si no, calcula `QuoteSale` (RF-05.2) con el `utcDiff` de la sesión, como el legado: cuando consulta el dueño es su zona, y un admin cotiza en la suya.
  - Responde la proyección.
- **RF-05.2** — `QuoteSale(bound, now, utcDiff) (SaleProjection, error)`, función pura con la fórmula de §5.1.
  - Si hoy es anterior a la fecha mínima de retiro, devuelve `ERR_INVALID_WITHDRAW_DATE`.
  - Usa los valores del snapshot del bono, en `float64`, sin truncar (corrige B14).
  - Un bono en estado `'2'` (madurado) se cotiza siempre como vencido: `daysPassed = M` y el monto completo, sin mirar la fecha (corrige B37).
- **RF-05.3** — Venta o canje.
  - Mientras `BLOOMING_TRADING` no sea `true`, responde `ERR_TRADING_DISABLED` antes de cargar el bono y sin escribir nada (spec 04, RF-04.9). La cotización no se apaga.
  - Carga el bono, que **debe ser del usuario de la sesión**. Ni siquiera un admin vende bonos ajenos (corrige B16). No aplica el alcance de agencia: ya exige ser el dueño.
  - Determina la transición por estado (§5.3) y calcula `QuoteSale`. Si la cotización falla, **aborta** sin escribir nada (corrige B3).
  - Ejecuta la transacción de RF-05.5 y las notificaciones de RF-05.7.
  - Responde `200 {bondBoundId, statusId, amount, profit, accountId, available}`.
- **RF-05.4** — Cuenta destino.
  - Se acredita en `bondUserSetting.selectedAccountId` del usuario (paridad, B6), que se lee con `BondUserSettingApplicationPorts.GetByUser` (spec 09).
  - Si no hay setting o `selectedAccountId` está vacío, `ERR_USER_SETTING_ACCOUNT_REQUIRED`.
  - La cuenta debe existir y ser del usuario (`BalancePort.GetAccount`); si no, `ERR_ACCOUNT_NOT_FOUND`.
- **RF-05.5** — Atomicidad y concurrencia. En una transacción:
  1. `UpdateStatusIf({_id, statusId: estadoActual, soldDate: null, tookDate: null}, {statusId: nuevo, soldDate|tookDate: now})`. Si no modifica ningún documento, `ERR_BOND_ALREADY_SOLD`.
  2. `BalancePort.Deposit` por `Round2(quote.amount)`, con `descriptionId` y `comment` según §5.3.
  3. `BondHistory.Add` (RF-05.6).

  Corrige el doble cobro: el gateway lo audita con `B_BondBound_Audit_doubleProcess`, que busca bonos con `soldDate` **y** `tookDate`.
- **RF-05.6** — Historial. Registra `{relatedId: bondBoundId, typeId: '2', text: 'BOND_BOUND_STATUS_CHANGED', params: [estadoAnterior, estadoNuevo]}`.
  - El formato no se cambia: `B_BondBound_Audit_details`, en `ms-gateway-gql/src/resolvers/B_BoundBound.js`, filtra `{text: 'BOND_BOUND_STATUS_CHANGED', params: {$size: 2}}`.
- **RF-05.7** — Notificaciones (post-commit y best-effort).
  - Un `Notice` tipo `bond` y el email `notify-blooming-bondSold` o `notify-blooming-bondTook`.
  - Título: `BOND_BOUND_SUBJECT_SOLD[key]` o `BOND_BOUND_SUBJECT_TOOK[key]`.
  - Mensaje: `BOND_BOUND_MESSAGE_SOLD` o `BOND_BOUND_MESSAGE_TOOK`, con los 11 parámetros de §5.4.

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| GET | `/bond-bounds/{id}/pre-sale` | dueño o admin | RF-05.1 |
| POST | `/bond-bounds/{id}/sale` | dueño | RF-05.3 |

**`GET /bond-bounds/{id}/pre-sale`** → `200`

```json
{ "inversion": 1000, "interest": 10, "penaltyPercentage": 50,
  "profit": 25, "atMoment": 1050, "amount": 1025, "totalDaysPassed": 50 }
```

**`POST /bond-bounds/{id}/sale`** (sin body) → `200`

```json
{ "bondBoundId": "…", "statusId": "5", "amount": 1025, "profit": 25, "accountId": "acc1", "available": 6025 }
```

**Cambio de contrato respecto al legado:**
- `preSale` respondía `{success, code, data: projection}`.
- `sale` respondía `{success, code, data: {_id, available}}`, y en error a veces nunca respondía.

**Errores**

| Sentinel | HTTP | Código legacy | Cuándo |
|---|---|---|---|
| `ERR_INVALID_ID` | 400 | — | Un id de path o body no es UUID v4 ni Meteor de 17 caracteres (spec 00, RF-00.19) |
| `ERR_BOND_BOUND_NOT_FOUND` | 404 | -11 | No existe o está borrado; en la cotización, también fuera del alcance de agencia |
| `ERR_FORBIDDEN` | 403 | — | Cotización de un bono ajeno sin ser admin, o venta de un bono ajeno |
| `ERR_BOND_ALREADY_SOLD` | 409 | — | Estado `'5'`, o perdió la carrera de RF-05.5 |
| `ERR_BOND_ALREADY_TOOK` | 409 | — | Estado `'6'` |
| `ERR_BOND_STATUS_NOT_SELLABLE` | 409 | — | Estado `'3'` (bloqueado) o `'4'` (desligado) |
| `ERR_INVALID_WITHDRAW_DATE` | 422 | -2030 | Hoy es anterior a `initialDate + initialDayForWithdraw` |
| `ERR_BOND_INVALID_TERMS` | 422 | — | `maturityTimeInDays - inflectionPoint <= 0` en el snapshot legacy |
| `ERR_USER_SETTING_ACCOUNT_REQUIRED` | 422 | — | Sin cuenta seleccionada |
| `ERR_ACCOUNT_NOT_FOUND` | 404 | -700 | La cuenta seleccionada no existe o no es del usuario |
| `ERR_TRADING_DISABLED` | 503 | -2040 | Venta o canje con `BLOOMING_TRADING` distinto de `true` (spec 04, RF-04.9) |

El mensaje de `ERR_INVALID_WITHDRAW_DATE` se arma con `BOUND_INVALID_WITHDRAW_DATE[hoy DD/MM/YYYY, en el utcDiff de la sesión]` y va en `errors.detail` (`handlers_shared.ResponseWithErrorDetail`), así el front puede mostrarlo.

## 5. Reglas de negocio

### 5.1 `QuoteSale` (paridad con `bond.bound.js:746-813`)

```
u          = utcDiff
today      = StartOfDay(now, u)
start      = StartOfDay(bound.initialDate, u)
withdraw   = start + earlyCashTerm.initialDayForWithdraw días
inflection = start + earlyCashTerm.inflectionPoint días
maturity   = StartOfDay(bound.maturityDate, u)
M = maturityTimeInDays;  P = inflectionPoint;  R = maturityRewardsPercentage;  Pf = inflectionPenaltyPercentage
effDays    = M - P                                    // si <= 0 → ERR_BOND_INVALID_TERMS
perDay     = amount / effDays
madurado   = statusId == '2'                          // se cotiza como vencido sin mirar la fecha (B37)
daysPassed = madurado ? M : min(DaysBetween(start, today, u), M)
interest   = perDay * (daysPassed - P) * (R / 100)    // negativo antes de la inflexión
penalized  = interest * (1 - Pf / 100)

si today < withdraw → ERR_INVALID_WITHDRAW_DATE

proj = {inversion: amount, interest: R, penaltyPercentage: Pf, profit: max(interest, 0), totalDaysPassed: daysPassed}
si madurado o today >= maturity:
                              proj.atMoment = interest + amount
                              proj.amount   = interest + amount
                              proj.profit   = max(interest, 0)
si no, si today < inflection: proj.amount = amount;            proj.atMoment = amount
si no (inflection <= today < maturity):
                              proj.atMoment = interest + amount
                              proj.amount   = amount + penalized
                              proj.profit   = max(penalized, 0)
redondear amount, atMoment y profit con Round2 al responder
```

**Diferencia con el legado (B1):** el legado evaluaba `totalDaysPassed <= maturityDate`, que compara días contra un timestamp en milisegundos y siempre es verdadero. El resultado:
- `amount` y `atMoment` coinciden con esta versión;
- pero en el vencimiento `profit` quedaba **penalizado**, aunque el monto acreditado era el completo.

Aquí, en el vencimiento, `profit` es el interés completo. Solo cambia el valor de "Ganancia" en el email y en la respuesta; el monto acreditado no cambia.

**Diferencia con el legado (B37):** el legado cotiza un bono `'2'` por la fecha, como a cualquier otro. Como la maduración corta al fin del día UTC (spec 10, RF-10.3), un bono puede estar en `'2'` cuando en la zona del usuario todavía es el día anterior al vencimiento (en Colombia, de 19:00 a 24:00), y entonces el canje se pagaba con penalidad y `M - 1` días. Aquí un `'2'` se cotiza siempre como vencido. La venta de un `'1'` no cambia.

### 5.2 Ejemplo verificable

`amount 1000`, `M 100`, `R 10`, `P 0`, `Pf 50`, `initialDayForWithdraw 0`:

| Día | `atMoment` | `amount` | `profit` (Go) | `profit` (legacy) |
|---|---|---|---|---|
| 50 | 1050 | 1025 | 25 | 25 |
| 100 | 1100 | 1100 | 100 | 50 |
| 150 (tope M) | 1100 | 1100 | 100 | 50 |

### 5.3 Transiciones

| Estado actual | Resultado | Nuevo estado | Fecha | `descriptionId` | `comment` |
|---|---|---|---|---|---|
| `'1'` initiated | venta | `'5'` sold | `soldDate` | `'8'` BOND_SALE_MC | `Bono "<key>" vendido.` |
| `'2'` finalized | canje | `'6'` took | `tookDate` | `'6'` BOND_MC | `Bono "<key>" canjeado.` |
| `'3'`, `'4'` | — | — | — | — | `ERR_BOND_STATUS_NOT_SELLABLE` |
| `'5'` | — | — | — | — | `ERR_BOND_ALREADY_SOLD` |
| `'6'` | — | — | — | — | `ERR_BOND_ALREADY_TOOK` |

Un bono con `soldDate` o `tookDate` ya seteado se trata como vendido o canjeado, sin importar su `statusId`.
Un estado desconocido también es `ERR_BOND_STATUS_NOT_SELLABLE`.

### 5.4 Parámetros del mensaje (11, en orden)

1. `bondObject.name`, leído con `BondObjectApplicationPorts.GetByIDIncludingRemoved`, aunque esté borrado.
2. `bound.key`.
3. `FormatCurrency(bound.amount)`.
4. `bound.maturityRewardsPercentage`, del **snapshot**. El legado usaba el valor actual del producto.
5. `bound.maturityTimeInDays`, del snapshot.
6. `bound.initialDate` con formato `DD/MM/YYYY h:m:s a` en `utcDiff`. Corrige B2: el legado formateaba "ahora" por un typo.
7. `bound.maturityDate`, mismo formato.
8. La cuenta enmascarada.
9. `FormatCurrency(profit)`.
10. `FormatCurrency(amount acreditado)`.
11. `FormatCurrency(available después del depósito)`, tomado del `BalanceMovement` devuelto. Corrige B4b: el legado sumaba el monto otra vez.

## 6. Modelo de datos

Usa `BondBound` de la spec 04. Campos que escribe: `statusId`, `soldDate`, `tookDate` y `updatedAt`.

```go
type SaleProjection struct {
    Inversion         float64 `json:"inversion"`
    Interest          float64 `json:"interest"`
    PenaltyPercentage float64 `json:"penaltyPercentage"`
    Profit            float64 `json:"profit"`
    AtMoment          float64 `json:"atMoment"`
    Amount            float64 `json:"amount"`
    TotalDaysPassed   int     `json:"totalDaysPassed"`
}
```

## 7. Ports y dependencias

- `BondBoundApplicationPorts`: `PreSale` y `Sale`.
- `BondBoundMongoccRepositoryPorts`: `FindOneByID` y `UpdateStatusIf` (BLM-0077).
- `BondUserSettingApplicationPorts.GetByUser` (spec 09).
- `BondObjectApplicationPorts.GetByIDIncludingRemoved` (spec 02), solo para el nombre en el mensaje.
- `BalancePort.GetAccount` y `Deposit`, `TransactionRunnerPort`, `NoticePort`, `NotificationPort`, `ProfileReader` (spec 00).
- `BondHistoryPort.Add` (spec 08).

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| B1 | `preSale` compara `totalDaysPassed` (días) con `maturityDate` (ms): la rama penalizada siempre corre y en el vencimiento `profit` sale penalizado. | Corregir | El monto acreditado no cambia; se corrige el `profit` informado. | BLM-0090 |
| B2 | `setStatus` usa `bondBound.initalDate` (typo): la fecha de compra del email es "ahora". | Corregir | Dato incorrecto al cliente. | BLM-0094 |
| B3 | `sale` no verifica el error de `preSale`: con la fecha de retiro inválida acredita `NaN` y cambia el estado igual. | Corregir | Dinero. | BLM-0093 |
| B4 | `sale` traga las excepciones en `catch {}` y nunca responde. | Corregir | Llamada que no responde. | BLM-0093 |
| B4b | "Disponible actual" del email suma el monto dos veces. | Corregir | Dato incorrecto al cliente. | BLM-0094 |
| B5 | `sale` sobre un estado `'3'` o `'4'` envía `status: ''` y lanza `TypeError`. | Corregir | Llamada que no responde. | BLM-0091 |
| B6 | Se acredita en `selectedAccountId` y no en la cuenta de compra; crashea si no hay setting. | Mantener la cuenta destino y corregir el crash | Paridad funcional, con error explícito. | BLM-0093 |
| B14 | `parseInt` en `maturityTimeInDays`, `inflectionPoint` y `maturityRewardsPercentage`. | Corregir | Tasas con decimales. | BLM-0090 |
| B16 | `preSale` y `sale` no validan que el bono sea del usuario. `sale` acredita el valor de un bono ajeno en la cuenta de quien llama. | Corregir | Seguridad y dinero (crítico). | BLM-0093 |
| B21 | El email usa el rendimiento y el plazo actuales del producto, no los del bono comprado. | Corregir | Se usa el snapshot del bono. | BLM-0094 |
| B37 | `preSale` cotiza un bono `'2'` por la fecha, en la zona del usuario, pero la maduración (ms-feed) lo pasa a `'2'` al inicio del día UTC. Un canje en esas horas se paga con penalidad y `M - 1` días: un bono de 1000 a 365 días al 15 % (`P 30`, `Pf 75`) acredita 1037,39 en vez de 1150. | Corregir | Dinero. | BLM-0150 |

## 9. Criterios de aceptación

- **CA-05.1** (RF-05.2) — Con los datos de §5.2, el día 50: `amount 1025`, `atMoment 1050`, `profit 25` y `totalDaysPassed 50`.
- **CA-05.2** (RF-05.2) — Mismos datos, día 100 y día 150: `amount 1100`, `atMoment 1100`, `profit 100` y `totalDaysPassed 100`.
- **CA-05.3** (RF-05.2) — Con `M 100`, `P 30` y el día 20: `amount 1000`, `atMoment 1000` y `profit 0`.
- **CA-05.4** (RF-05.2) — Con `R 7.5` y `amount 1000.50`, el resultado coincide con la fórmula en float, sin truncar.
- **CA-05.5** (RF-05.1, RF-05.2) — Con `initialDayForWithdraw 30` y el día 10, `pre-sale` responde `422 ERR_INVALID_WITHDRAW_DATE` con la fecha de hoy en `DD/MM/YYYY`.
- **CA-05.6** (RF-05.1) — `pre-sale` de un bono ajeno responde `403` para un no admin y `200` para un admin.
- **CA-05.7** (RF-05.3, RF-05.4, RF-05.5) — Un bono propio en estado `'1'`, vendido el día 50, queda en `statusId '5'` con `soldDate`. Se crea un `balanceDetail` `+1025` con `descriptionId '8'` en la cuenta seleccionada, y la respuesta trae `available` actualizado.
- **CA-05.8** (RF-05.3, RF-05.5) — Un bono propio en estado `'2'` queda en `'6'` con `tookDate`, y se acredita el monto completo con `descriptionId '6'`.
- **CA-05.9** (RF-05.3) — Estados `'3'`/`'4'` → `409 ERR_BOND_STATUS_NOT_SELLABLE`; `'5'` → `ERR_BOND_ALREADY_SOLD`; `'6'` → `ERR_BOND_ALREADY_TOOK`. En ninguno se escribe nada.
- **CA-05.10** (RF-05.3) — `POST /sale` sobre un bono ajeno responde `403` incluso para un admin, y no se acredita nada en ninguna cuenta.
- **CA-05.11** (RF-05.3) — `POST /sale` antes de la fecha mínima de retiro responde `422 ERR_INVALID_WITHDRAW_DATE`; el estado y el saldo no cambian.
- **CA-05.12** (RF-05.4) — Sin `bondUserSetting`, o con `selectedAccountId: null`, responde `422 ERR_USER_SETTING_ACCOUNT_REQUIRED`.
- **CA-05.13** (RF-05.5) — Con dos `POST /sale` concurrentes sobre el mismo bono, uno responde `200` y el otro `409`, y existe un solo `balanceDetail` de crédito.
- **CA-05.14** (RF-05.6) — Después de una venta, `bondHistory` tiene `{typeId '2', text 'BOND_BOUND_STATUS_CHANGED', params ['1','5']}`, y la consulta del gateway `{text: 'BOND_BOUND_STATUS_CHANGED', params: {$size: 2}}` lo encuentra.
- **CA-05.15** (RF-05.7) — El mensaje del email usa `initialDate` del bono (no ahora) y el `available` posterior al depósito sin sumarlo dos veces. Si ms-notification falla, la venta responde `200`.
- **CA-05.16** (RF-05.3, RF-05.5, RF-05.6) — Venta de un `bondBound` legacy (`_id` de 17 caracteres, comprado en Meteor): el estado cambia, el `_id` se conserva y el `bondHistory` nuevo (con `_id` UUID) tiene `relatedId` igual al `_id` legacy.
- **CA-05.17** (RF-05.2) — Un bono de `amount 1000`, `M 365`, `R 15`, `P 30` y `Pf 75`, comprado el `2026-10-13T20:30Z` (`maturityDate 2027-10-13T20:30Z`), se cotiza el `2027-10-13T01:00Z` con `utcDiff -5`, que en la zona del usuario todavía es el 12 de octubre:
  - en `'2'`: `amount 1150`, `atMoment 1150`, `profit 150` y `totalDaysPassed 365`;
  - en `'1'`: `amount 1037.39`, `atMoment 1149.55`, `profit 37.39` y `totalDaysPassed 364`, igual que antes.
- **CA-05.18** (RF-05.1) — `pre-sale` de un bono de la agencia B responde `404 ERR_BOND_BOUND_NOT_FOUND` a un admin de A que no es la principal, y `200` a un admin de la agencia principal.
- **CA-05.19** (RF-05.3) — Con `BLOOMING_TRADING` distinto de `true`, `POST /sale` sobre un bono propio vendible responde `503 ERR_TRADING_DISABLED`. El estado, el saldo y el historial no cambian, y `pre-sale` del mismo bono sigue respondiendo `200`.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0090 | Función pura `QuoteSale` y `SaleProjection`, con la corrección de B1 y sin truncados | impl | F5 | RF-05.2 | BLM-0075, BLM-0009 | `internal/core/bondBound/domain/bondBound.quoteSale.go`, `bondBound.errors.go` y `bondBound.httpStatus.go` (`ERR_INVALID_WITHDRAW_DATE`, `ERR_BOND_INVALID_TERMS`) |
| BLM-0091 | Función pura `SaleTransitionFor(bound)` (tabla de §5.3) y sus errores | impl | F5 | RF-05.3 | BLM-0075 | `internal/core/bondBound/domain/bondBound.saleTransition.go`, `bondBound.errors.go` y `bondBound.httpStatus.go` (`ERR_BOND_ALREADY_SOLD`, `ERR_BOND_ALREADY_TOOK`, `ERR_BOND_STATUS_NOT_SELLABLE`) |
| BLM-0092 | Caso de uso `PreSale`, con control de dueño o admin | impl | F5 | RF-05.1 | BLM-0090, BLM-0091, BLM-0077, BLM-0006, BLM-0007 | `internal/core/bondBound/application/bondBound.preSale.go`, `internal/core/bondBound/ports/bondBound.ports.go` (`PreSale`) |
| BLM-0093 | Caso de uso `Sale`: dueño, transición, cotización, cuenta destino y transacción con `UpdateStatusIf`, `Deposit` y el historial | impl | F5 | RF-05.3, RF-05.4, RF-05.5, RF-05.6 | BLM-0092, BLM-0067, BLM-0010, BLM-0012, BLM-0035 | `internal/core/bondBound/application/bondBound.sale.go`, `bondBound.application.go` (`UserSettings`), `internal/core/bondBound/ports/bondBound.ports.go` (`Sale`), `internal/core/bondBound/domain/bondBound.dto.domain.go` (`SaleResult`), `bondBound.errors.go` y `bondBound.httpStatus.go` (`ERR_USER_SETTING_ACCOUNT_REQUIRED`), `cmd/api/server/server.bondBound.go` (`UserSettings`) |
| BLM-0094 | Notificaciones post-commit de venta y canje (11 parámetros, snapshot y fechas) | impl | F5 | RF-05.7 | BLM-0093, BLM-0013, BLM-0014, BLM-0015, BLM-0018 | `internal/core/bondBound/application/bondBound.sale.notify.go`, `bondBound.sale.go` (llamada después del commit) |
| BLM-0095 | Handlers `pre-sale` (GET) y `sale` (POST) | impl | F5 | RF-05.1, RF-05.3 | BLM-0092, BLM-0093, BLM-0084, BLM-0026 | `cmd/api/handlers/bondBounds/bondBounds.preSale.get.go`, `bondBounds.sale.post.go`, `bondBounds.handler.go` (rutas y detalle de `ERR_INVALID_WITHDRAW_DATE`), `cmd/api/handlers/shared/handlers.response.go` (`ResponseWithErrorDetail`), `internal/core/shared/domain/shared.text.go` (`BOUND_INVALID_WITHDRAW_DATE`) |
| BLM-0096 | Tests de tabla de `QuoteSale` contra casos calculados con el código legacy | test | F5 | CA-05.1, CA-05.2, CA-05.3, CA-05.4 | BLM-0090 | `internal/core/bondBound/domain/bondBound.quoteSale_test.go` |
| BLM-0097 | Tests unitarios de `PreSale` y `Sale` con fakes (dueño, estados, cotización inválida, cuenta destino) y de sus handlers | test | F5 | CA-05.5, CA-05.6, CA-05.9, CA-05.10, CA-05.11, CA-05.12 | BLM-0095 | `internal/core/bondBound/application/bondBound.preSale_test.go`, `bondBound.sale_test.go`, `bondBound.fakes_test.go` (`UpdateStatusIf`, `Deposit`, `GetByIDIncludingRemoved` y `UserSettings` en memoria), `cmd/api/handlers/bondBounds/bondBounds.preSale.get_test.go`, `bondBounds.sale.post_test.go`, `bondBounds.testServer_test.go` (`PreSale` y `Sale` en `fakeApp`) |
| BLM-0098 | Tests de integración: venta, canje, concurrencia, formato del historial para el gateway y notificaciones | test | F5 | CA-05.7, CA-05.8, CA-05.13, CA-05.14, CA-05.15, CA-05.16 | BLM-0095, BLM-0094 | `internal/core/bondBound/application/bondBound.sale_integration_test.go`, `bondBound.testdb_integration_test.go` (`bondUserSetting` y `UserSettings` con el adapter real) |
| BLM-0150 | `QuoteSale` cotiza un bono `'2'` como vencido (`daysPassed = M` y monto completo), sin mirar la fecha | impl | F5 | RF-05.2 | BLM-0090 | `internal/core/bondBound/domain/bondBound.quoteSale.go` |
| BLM-0151 | Tests de la cotización de un `'2'` antes de su día de vencimiento en la zona del usuario, y de que un `'1'` no cambia | test | F5 | CA-05.17 | BLM-0150 | `internal/core/bondBound/domain/bondBound.quoteSale_test.go` |

El alcance de agencia de RF-05.1 (B38) va en BLM-0162 y BLM-0163 (spec 06). El apagado de la venta hasta el corte (RF-05.3) va en BLM-0166 y BLM-0167 (spec 04).

## 11. Fuera de alcance y preguntas abiertas

- `earlyCashTerm.blockWithdrawals` no lo usa ninguna regla del legado y tampoco se usa aquí.
- **Pregunta:** si un bono sigue en `'1'` pero ya pasó su `maturityDate` (el cron todavía no corrió), el legado lo **vende** (`descriptionId '8'`) con el monto completo. ¿Debería ser un canje? Por ahora se mantiene la paridad. Con el corte al fin del día UTC (spec 10), esa ventana dura como mucho un tick del job (5 min) después de las 00:00 UTC.
- **Pregunta:** ¿se debe validar que la cuenta destino esté activa? El legado acredita aunque esté suspendida o cancelada.
- **Pregunta:** ¿el admin necesita vender en nombre de un usuario (acreditando la cuenta del dueño)? Hoy no está contemplado.
