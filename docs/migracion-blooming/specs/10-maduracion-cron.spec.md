# Spec 10 — Maduración de bonos (cron)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondBound` (caso de uso y endpoint interno); la programación vive en ms-cronjob-go |
| Colecciones | `bondBound`, `bondJobLock` (nueva) |
| Depende de | Spec 00 (notificaciones, shutdown), spec 04 (repositorio), spec 08 (`BondHistoryPort`), ms-cronjob-go (disparo) |

## 1. Resumen y alcance

Cuando un bono `initiated ('1')` llega a su `maturityDate`, pasa a `finalized ('2')`, es decir, madurado. Desde ese momento el inversionista lo puede canjear (spec 05). Además se le avisa con un aviso y un email por usuario que lista los bonos madurados.

Hoy esto lo hace ms-feed (`PUT /bound/check-maturity`, `ms-feed/src/components/bound/index.js:124-246`). Lo llama cada 30 s el job `BondBoundCheckMaturity` (`SyncedCron`) de Meteor, por la variable `MS_FEED_URL` (constante `MS_FINOBONDS_FEED_URL`), y Meteor manda el aviso y el email con la respuesta. Esta spec **trae la regla a ms-blooming-go** (caso de uso y endpoint interno) y deja la programación en ms-cronjob-go, que llama al endpoint cada 5 minutos. Juntos reemplazan a los dos.

## 2. Origen legacy

| Pieza | Archivo:línea | Destino |
|---|---|---|
| `boundController.checkMaturity` de ms-feed (`PUT /bound/check-maturity`): la regla que corre hoy | `ms-feed/src/components/bound/index.js:124-246` | RF-10.3, RF-10.5 |
| Job `BondBoundCheckMaturity` (cada 30 s, `PUT …/bound/check-maturity` + email + notice) | `backend/server/lib/crons.js:8-64` (`origin/main`) | RF-10.1, RF-10.5 |
| `bondBound.checkMaturity` (se agregó ya comentado en `37fedea`, 2018, y nunca corrió: el job siempre llamó a ms-feed) | `backend/server/model/bond.bound.js:1389-1420` (`origin/main`) | RF-10.3 |
| `Text.BOND_BOUND_MESSAGE_MATURITY` / `_SUCESS` | `backend/server/lib/text.js:100-105` | RF-10.5 |
| Arranque de `SyncedCron`: `Cron.init()` | `backend/server/main.js:10` (`origin/main`) | Reemplazado por RF-10.1 |

Las referencias a Meteor son de `origin/main`: el checkout local del backend puede estar muy atrás.

## 3. Requisitos funcionales

- **RF-10.1** — Programación (en ms-cronjob-go).
  - ms-blooming-go no programa jobs ni usa una librería de cron (constitución §12).
  - ms-cronjob-go registra un `scheduler.Job` (`blooming-bond-maturity`, cada 300 s) que hace `POST {MS_BLOOMING_URL}/internal/jobs/bond-maturity/run` con `x-secret` (RF-10.7), igual que el job de auditoría de Telegram de petty-cash (FA-359). `MS_BLOOMING_URL` incluye el prefijo `/api/v1`, como `MS_PETTY_CASH_URL`.
  - Si falta `MS_BLOOMING_URL`, el adapter queda deshabilitado, lo avisa con un warning al arrancar y el job **no se registra** (el de petty-cash, en cambio, loguea un error en cada tick). La variable queda vacía hasta el corte (BLM-0134), porque esta maduración no debe convivir con la de ms-feed.
  - No reintenta: la corrida es idempotente y el próximo tick la repite. El adapter devuelve un error centinela para el `409 ERR_JOB_LOCKED`, que el job loguea como informativo; cualquier otro status se loguea como error.
  - El cliente espera hasta 10 min (el TTL del lock). Si corta antes por timeout, la corrida sigue en ms-blooming-go hasta terminar. Los ticks que caen durante una corrida larga se pierden (el scheduler no los encola), y no importa porque la corrida es idempotente.
- **RF-10.2** — Exclusión entre réplicas.
  - Antes de procesar, el job toma un lock en `bondJobLock` con `updateOne({_id: "bond-maturity", lockedUntil: {$lt: now}}, {$set: {lockedUntil: now + 10min, owner: hostname}}, upsert)`. Si el upsert choca por clave duplicada, el lock está tomado.
  - Si no lo obtiene, termina sin hacer nada.
  - Al final libera el lock: `lockedUntil = now`, solo si `owner` coincide.
- **RF-10.3** — Selección y transición.
  - Procesa en lotes de 500 los bonos con `{isRemove: false, statusId: '1', maturityDate: {$lt: cutoff}, soldDate: null, tookDate: null}`, con `cutoff = StartOfDay(now, 0) + 1 día` (el inicio de mañana en UTC). Madura todo lo que vence hoy en UTC, aunque sea más tarde que `now`, igual que ms-feed (`maturityDate <= hoy 23:59:59`, con el servidor en UTC).
  - El corte usa UTC y no un `utcDiff`: el job no tiene sesión y se eligió la paridad con ms-feed (constitución §8). En América el bono madura antes de que empiece su día de vencimiento en la zona del usuario; en Colombia, a las 19:00 del día anterior. Por eso la cotización trata un bono `'2'` como vencido (spec 05, B37).
  - Cada bono pasa a `finalized` con `UpdateStatusIf({_id, statusId: '1', soldDate: null, tookDate: null}, {statusId: '2', matured: true, maturedAt: now})`.
  - **Solo** se maduran bonos en estado `'1'`. Corrige B22: entre enero de 2018 y mayo de 2021, ms-feed maduraba también los estados vendido `'5'` y desligado `'4'`.
  - Si `UpdateStatusIf` no modifica el bono (otro proceso lo vendió entre medio), el bono se omite.
  - Otras diferencias con ms-feed:
    - filtra `isRemove`, `soldDate` y `tookDate`, que ms-feed no mira;
    - madura el bono aunque falten su `profile`, su `bondType` o su `bondObject` (corrige B36);
    - escribe el historial (RF-10.4), `maturedAt` y `updatedAt`; ms-feed solo guarda `statusId`: su esquema de Mongoose no declara `matured`, y ese `$set` se pierde.
- **RF-10.4** — Historial. Por cada bono madurado: `BondHistory.Add` en modo sistema con `{relatedId, typeId: '2', text: 'BOND_BOUND_STATUS_CHANGED', params: ['1','2'], userId: "", agencyId: bound.agencyId}`.
  - Con `userId ""`, la auditoría del gateway (`B_BondBound_Audit_details`, que agrupa por `(userId, relatedId)`) no mezcla la maduración con las ventas y los canjes del usuario, y no genera falsos positivos de doble cobro.
  - La transición y el historial de cada bono van en la misma transacción: si el historial falla, el bono queda en `'1'`, cuenta como fallido y se reintenta en la próxima corrida.
- **RF-10.5** — Notificaciones por usuario (best-effort, después de la transición).
  - Los bonos madurados en la corrida se agrupan por `userId`.
  - Para cada usuario se envía:
    - un `Notice` tipo `bond`, con la `agencyId` del usuario, título `BOND_BOUND_MESSAGE_MATURITY` y mensaje `BOND_BOUND_MESSAGE_MATURITY_SUCESS[first_name, last_name, tabla]`;
    - el email `notify-blooming-bondMatured`.
  - La tabla tiene una fila por bono: `Bono (key) | Tipo (bondType.name) | Nombre (bondObject.name)`, con el HTML del legado.
  - El email sale sin sesión, por `POST /notify/internal/{group}/{notification}`, la ruta interna de ms-notification que solo valida `x-secret` (BLM-0140). Si ms-notification todavía no tiene la ruta, responde 404 y el fallo solo se loguea; el `notice` se guarda igual.
- **RF-10.6** — Robustez y observabilidad.
  - Si falla un bono, se loguea con su `_id` y el lote sigue.
  - Al terminar se loguea `{matured, users, failed, durationMs}`.
  - Una corrida nunca deja el lock tomado más allá de su TTL.
- **RF-10.7** — Endpoint interno. `POST /internal/jobs/bond-maturity/run` (`x-secret`, `NoRequiresAuthentication`) ejecuta la corrida con el lock y responde `{matured, users, failed}`. Es la entrada de ms-cronjob-go y también sirve para operación y pruebas.
  - El endpoint extiende su deadline de escritura hasta el TTL del lock (10 min), para que la respuesta llegue aunque la corrida pase de los 15 s del `WriteTimeout` del servidor.
  - Si llega SIGTERM durante una corrida, el shutdown espera al request en curso (RF-00.16, 20 s). Si se corta, los bonos ya madurados quedan completos (cada uno va en su transacción) y el lock vence por su TTL.

## 4. Contrato API

| Método | Ruta | Permiso | RF |
|---|---|---|---|
| POST | `/internal/jobs/bond-maturity/run` | interno (`x-secret`) | RF-10.7 |

→ `200 {"matured": 12, "users": 7, "failed": 0}`. Si el lock está ocupado: `409 ERR_JOB_LOCKED`.

## 5. Reglas de negocio

```
si !Lock.Acquire("bond-maturity", 10m) → return ERR_JOB_LOCKED
defer Lock.Release
porUsuario = {}
cutoff = StartOfDay(now, 0) + 1 día              // inicio de mañana en UTC
para cada lote de Repo.FindMaturable(cutoff, 500):
  para cada b en lote:
    ok, err = Tx {                              // transición e historial juntos
      ok = Repo.UpdateStatusIf(b._id, from '1', to '2', set {matured: true, maturedAt: now})
      si ok → BondHistory.Add(WithSystemAgency(txCtx, b.agencyId), {relatedId: b._id, typeId: '2', text: 'BOND_BOUND_STATUS_CHANGED', params: ['1','2']})
    }
    si err → log, failed++ ; continue         // el bono queda en '1'
    si !ok → continue                          // ya no estaba en '1'
    porUsuario[b.userId].append(b)
para cada (userId, bonds) en porUsuario:
  Notify(userId, bonds)                         // best-effort
```

Una corrida que se repite es idempotente: la condición `statusId: '1'` impide madurar dos veces y, por lo tanto, avisar dos veces.

## 6. Modelo de datos

- `bondBound`: escribe `statusId`, `matured`, `maturedAt` (campo nuevo, para auditoría) y `updatedAt`. Usa el índice `{statusId: 1, maturityDate: 1}` de la spec 04.
- `bondJobLock`: `{_id: string, lockedUntil: Date, owner: string, updatedAt: Date}`.

## 7. Ports y dependencias

- `BondBoundApplicationPorts.RunMaturity(ctx, now) (MaturityResult, error)`.
- `BondBoundMongoccRepositoryPorts`: `FindMaturable(ctx, cutoff, limit, afterID)` (`maturityDate < cutoff`) y `UpdateStatusIf` (BLM-0077).
- `JobLockPort`: `Acquire(ctx, name, ttl) (bool, error)` y `Release(ctx, name)`, en `shared_ports`.
- `BondHistoryPort.Add` en modo sistema (spec 08).
- `NoticePort`, `NotificationPort`, `ProfileReader` (spec 00) y lectura de los nombres de `bondType` y `bondObject` (specs 01 y 02).
- ms-cronjob-go: adapter HTTP `blooming_api` y job `blooming-bond-maturity` (300 s), con `MS_BLOOMING_URL` (con `/api/v1`), `MS_BLOOMING_SECRET` y `MS_BLOOMING_DEBUG`.

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| B22 | ms-feed maduraba también bonos **vendidos** (`'5'`) y desligados (`'4'`) desde `87b4593` (enero de 2018) hasta `a33478e` (mayo de 2021); queda el comentario en `bound/index.js:133`. La regla de Meteor (`bondBound.checkMaturity`) se agregó ya comentada y nunca corrió. Un bono vendido volvía a `'2'` y se podía canjear de nuevo: es el doble cobro que audita `B_BondBound_Audit_doubleProcess`. La versión actual de ms-feed solo toma `'1'`, pero no mira `soldDate`/`tookDate`, y su `updateMany` no repite la condición de estado: un bono vendido entre la consulta y la actualización también pasa a `'2'`. | Corregir (solo `'1'` y sin `soldDate`/`tookDate`); los `'2'` con `soldDate` pasan a `'5'` (§11) | Dinero (crítico). | BLM-0118, BLM-0133 |
| B32 | En una rama vieja del backend, `Cron.init()` está comentado en `main.js` y la maduración solo arranca después de editar un cronjob. En `origin/main` está activo desde `029e73c` (2022), así que en producción la maduración corre cada 30 s vía ms-feed. | No aplica en `main` | Se revisó contra `origin/main`. | — |
| B36 | `checkMaturity` de ms-feed nunca madura un bono cuyo `profile`, `bondType` o `bondObject` no existe: los `$unwind` (`bound/index.js:149,158,167`) lo sacan de la agregación, y el `updateMany` solo usa los ids que salen de ella. | Corregir | Bonos que nunca se pueden canjear. | BLM-0118 |
| B33 | Con 3 réplicas sin lock, la maduración y los avisos se ejecutarían tres veces. | Corregir (prevención) | Avisos duplicados. | BLM-0119 |

## 9. Criterios de aceptación

- **CA-10.1** (RF-10.1) — En ms-cronjob-go, con `MS_BLOOMING_URL` vacío el job no se registra y hay un warning al arrancar. Con la URL (que incluye `/api/v1`), cada tick hace `POST …/internal/jobs/bond-maturity/run` con `x-secret` y loguea `{matured, users, failed}`; un `409` se loguea como informativo y un `500`, como error.
- **CA-10.2** (RF-10.2) — Dos ejecuciones concurrentes (que simulan dos réplicas) dan este resultado: una procesa y la otra termina con `ERR_JOB_LOCKED` sin modificar nada. Un lock vencido (`lockedUntil` en el pasado) se puede volver a tomar.
- **CA-10.3** (RF-10.3) — Dados estos bonos:
  - A: `'1'`, vence ayer;
  - B: `'1'`, vence mañana (en UTC);
  - C: `'5'`, vence ayer;
  - D: `'4'`, vence ayer;
  - E: `'1'`, vence ayer, `isRemove: true`;
  - F: `'1'`, vence ayer, con `soldDate`.

  Solo A pasa a `'2'`, con `matured: true` y `maturedAt`.
- **CA-10.4** (RF-10.3, RF-10.5) — Si se ejecuta dos veces seguidas, la segunda corrida da `matured: 0`, sin nuevos historiales ni avisos.
- **CA-10.5** (RF-10.4) — El historial de A es `{typeId '2', text 'BOND_BOUND_STATUS_CHANGED', params ['1','2'], userId ''}`. Agrupado por `(userId, relatedId)` como hace `B_BondBound_Audit_details`, queda en un grupo distinto del de una venta del usuario sobre el mismo bono.
- **CA-10.6** (RF-10.5) — Un usuario con 3 bonos madurados en la misma corrida recibe 1 `notice` y 1 email, con una tabla de 3 filas (key, tipo y nombre). Si ms-notification falla, los bonos quedan madurados igual.
- **CA-10.7** (RF-10.6) — Si `UpdateStatusIf` falla para 1 de 5 bonos (fake), los otros 4 maduran y el log final informa `failed: 1`.
- **CA-10.8** (RF-10.7) — `POST /internal/jobs/bond-maturity/run` sin `x-secret` responde `401`. Con `x-secret` responde `200 {matured, users, failed}`.
- **CA-10.9** (RF-10.3, RF-10.4) — Un `bondBound` legacy (17 caracteres) en estado `'1'` y vencido madura igual que uno nuevo, y su historial de sistema tiene `relatedId` igual al `_id` legacy.
- **CA-10.10** (RF-10.3) — El corte es el fin del día en UTC: con `now 2026-10-13T00:03Z`, un bono `'1'` con `maturityDate 2026-10-13T23:59:59Z` madura, con `maturedAt` igual a `now`, y uno con `2026-10-14T00:00:00Z` no.

## 10. Tareas

| ID | Tarea | Tipo | Fase | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|---|
| BLM-0117 | Dominio: `MaturityResult` y función pura `BuildMaturityTable(rows)` (HTML del legado) | impl | F7 | RF-10.3, RF-10.5 | BLM-0075 | `internal/core/bondBound/domain/bondBound.maturity.go` |
| BLM-0118 | Adapter `FindMaturable` (lotes con cursor por `_id`), con transición condicional solo desde `'1'` | impl | F7 | RF-10.3 | BLM-0077 | `internal/core/bondBound/infrastructure/mongocc/bondBound.findMaturable.adapter.go`, `internal/core/bondBound/ports/bondBound.ports.go` (método `FindMaturable`) |
| BLM-0119 | `JobLockPort` y su adapter mongocc (`bondJobLock`) | impl | F7 | RF-10.2 | BLM-0010 | `internal/core/shared/ports/shared.ports.go` (`JobLockPort`), `internal/core/shared/infrastructure/mongocc/jobLock.mongocc.repository.go`, `jobLock.acquire.adapter.go`, `jobLock.release.adapter.go` |
| BLM-0120 | Caso de uso `RunMaturity`: lock, lotes, transición, historial de sistema, agrupación y notificaciones | impl | F7 | RF-10.3, RF-10.4, RF-10.5, RF-10.6 | BLM-0117, BLM-0118, BLM-0119, BLM-0035, BLM-0013, BLM-0014, BLM-0015 | `internal/core/bondBound/application/bondBound.runMaturity.go`, `bondBound.runMaturity.notify.go`, `bondBound.application.go` (`BondTypes` y `JobLock`), `bondBound.sale.notify.go` (`bondName` compartido), `internal/core/bondBound/ports/bondBound.ports.go` (`RunMaturity`), `internal/core/bondBound/domain/bondBound.errors.go` y `bondBound.httpStatus.go` (`ERR_JOB_LOCKED`), `cmd/api/server/server.bondBound.go`, `server.config.go` y `server.deps.go` (`JobLockPort` con el hostname) |
| BLM-0122 | Endpoint interno de ejecución de la maduración (lo llama ms-cronjob-go) | impl | F7 | RF-10.7 | BLM-0120, BLM-0004 | `cmd/api/handlers/jobs/jobs.bondMaturity.post.go`, `jobs.handler.go`, `cmd/api/server/server.jobs.go` y `server.go` (registro de las rutas) |
| BLM-0123 | Tests unitarios: tabla HTML, agrupación por usuario, estados excluidos, fallo parcial | test | F7 | CA-10.3, CA-10.6, CA-10.7 | BLM-0120 | `internal/core/bondBound/application/bondBound.runMaturity_test.go`, `bondBound.fakes_test.go` (`FindMaturable`, fallo por bono de `UpdateStatusIf`, `BondTypes`, `JobLock` y `captureLog`), `internal/core/bondBound/domain/bondBound.maturity_test.go` |
| BLM-0124 | Tests de integración: idempotencia, lock concurrente, historial de sistema y endpoint | test | F7 | CA-10.2, CA-10.4, CA-10.5, CA-10.8, CA-10.9 | BLM-0120, BLM-0122 | `internal/core/bondBound/application/bondBound.runMaturity_integration_test.go` |
| BLM-0140 | **(repo ms-notification)** Ruta interna de notificaciones que solo valida `x-secret`; `NotificationClient` la usa cuando el ctx no tiene sesión | impl | F7 | RF-10.5 | BLM-0014, BLM-0015 | `ms-notification/src/api/modules/notify/notify.routes.js`, `ms-notification/CLAUDE.md`, `internal/core/shared/infrastructure/notification/notification.client.go` |
| BLM-0141 | **(repo ms-cronjob-go)** Adapter `blooming_api` y job `blooming-bond-maturity` cada 300 s que llama al endpoint interno de maduración: no se registra sin URL, el 409 se loguea como informativo y el timeout es de 10 min. Corrige además lo desactualizado de `CLAUDE.md` y `README.md` (antes, `git pull`: la rama va detrás de su remoto) | impl | F7 | RF-10.1 | BLM-0122 | `ms-cronjob-go/internal/core/infrastructure/adapter/blooming_api/`, `internal/core/application/bond_maturity/`, `internal/core/port/`, `internal/core/domain/`, `cmd/api/main.go`, `cmd/api/config/enviroments.config.go`, `.env.example`, `README.md`, `CLAUDE.md` |
| BLM-0142 | **(repo ms-cronjob-go)** Tests del adapter (`httptest`) y del job: URL vacía sin registrar el job, POST con `x-secret` a la URL con `/api/v1`, 409 como error centinela informativo y error | test | F7 | CA-10.1 | BLM-0141 | `ms-cronjob-go/internal/core/infrastructure/adapter/blooming_api/blooming_api_test.go`, `internal/core/application/bond_maturity/*_test.go`, `CLAUDE.md` (paquetes con tests) |
| BLM-0148 | Corte de maduración al fin del día en UTC: `RunMaturity` calcula `cutoff` y `FindMaturable` filtra `maturityDate < cutoff` | impl | F7 | RF-10.3 | BLM-0118, BLM-0120 | `internal/core/bondBound/application/bondBound.runMaturity.go`, `internal/core/bondBound/infrastructure/mongocc/bondBound.findMaturable.adapter.go`, `internal/core/bondBound/ports/bondBound.ports.go` (parámetro `cutoff`), `cmd/api/handlers/jobs/jobs.bondMaturity.post.go` (comentario) |
| BLM-0149 | Tests del corte (lo que vence hoy en UTC madura aunque sea más tarde; lo de mañana no) y ajuste de los tests de maduración que asumían `maturityDate <= now` | test | F7 | CA-10.10 | BLM-0148 | `internal/core/bondBound/application/bondBound.runMaturity_test.go`, `bondBound.fakes_test.go` (`FindMaturable` con corte), `bondBound.runMaturity_integration_test.go` |
| BLM-0153 | Tests del `NotificationClient`: sin sesión, o con una sesión sin token, va a `/notify/internal/{group}/{notification}` sin `x-token`; con sesión, a la ruta normal; el email de maduración va a la ruta interna | test | F7 | CA-00.40, CA-10.6 | BLM-0140 | `internal/core/shared/infrastructure/notification/notification.client_test.go`, `internal/core/bondBound/application/bondBound.runMaturity_test.go`, `bondBound.fakes_test.go` y `bondBound.purchase.notify_test.go` (token en las sesiones de prueba) |

## 11. Fuera de alcance y preguntas abiertas

- **Resuelta:** el servicio `:8090` es ms-feed (`bound/index.js:124-246`). Hoy solo madura `'1'`, con corte al fin del día del servidor (UTC), sin historial ni `maturedAt`, y deja sin madurar los bonos sin `profile`, `bondType` o `bondObject` (B36). Una versión anterior maduraba también `'5'`/`'4'`, así que la auditoría de datos (BLM-0133) sigue buscando bonos vendidos que pasaron a `'2'`.
- **Decisión (corte):** paridad con ms-feed: madura todo lo que vence en el día UTC (RF-10.3). La hora exacta (`maturityDate <= now`), que se implementó primero (BLM-0118 y BLM-0120), movía hasta ~24 h el momento de la maduración respecto a producción. El corte en UTC exige que la cotización trate un `'2'` como vencido (spec 05, B37).
- **Decisión (datos de B22):** con `-apply`, la auditoría de datos (BLM-0133) pasa a `'5'` cada bono `'2'` con `soldDate` y sin `tookDate`, en una transacción por bono y con esa misma condición en el update. Solo lo hace si el historial del bono registra la venta (`BOND_BOUND_STATUS_CHANGED` hacia `'5'`); sin esa evidencia el bono se informa para revisión manual, porque pudo quedar sin cobrar. Registra un historial de sistema `{typeId '2', text 'BOND_BOUND_STATUS_CHANGED', params ['2','5'], userId ""}` con la `agencyId` del bono, y antes guarda un respaldo que permite revertirlo. No mueve saldo: el bono ya se cobró con la venta.
  - Así Meteor deja de pagarlo como canje. Go ya lo rechaza (`409 ERR_BOND_ALREADY_SOLD`, spec 05).
  - Con `userId ""`, `B_BondBound_Audit_details` no lo cuenta (RF-10.4).
  - Los `'6'` con `soldDate` y `tookDate` ya se cobraron dos veces: solo se informan, para contabilidad y como línea base de BLM-0138.
  - Se corre en live cuanto antes, y otra vez después de BLM-0134, por la carrera del `updateMany` de ms-feed.
- **Corte (primera corrida):** la primera corrida de Go madura todos los `'1'` vencidos, también los que ms-feed nunca maduró (B36), y avisa una vez a cada dueño con todos sus bonos. Antes de BLM-0134, la auditoría de datos informa cuántos son y de cuántos dueños, cuáles no tienen `profile`, `bondType` o `bondObject`, los que Go no decodifica, que cortan la corrida en su lote, y los `'1'` sin `agencyId`, que fallan solos en cada corrida y nunca maduran.
- **Decisión (Pregunta 4 de la spec 00):** el email de maduración sale sin sesión, y `POST /notify/:group/:notification` de ms-notification exige `x-token`. Se agrega a ms-notification una ruta interna que solo valida `x-secret`, y `NotificationClient` la usa cuando el ctx no tiene sesión (BLM-0140). El corte (BLM-0134) depende de esa tarea.
- **Decisión (programación):** la maduración la programa ms-cronjob-go y no el ms, para no sumar una librería de cron (constitución §1 y §12). El scheduler interno con `robfig/cron/v3` (BLM-0121) se retiró: su código nunca se versionó, y el commit con ese ID solo quitó su configuración y su documentación. El ID no se reutiliza.
- **Pregunta:** ¿la frecuencia de 5 minutos es suficiente? El legado corría cada 30 s, pero los avisos se pueden atrasar unos minutos sin impacto.
