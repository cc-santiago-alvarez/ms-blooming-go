# Spec 10 — Maduración de bonos (cron)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondBound` (caso de uso y endpoint interno); la programación vive en ms-cronjob-go |
| Colecciones | `bondBound`, `bondJobLock` (nueva) |
| Depende de | Spec 00 (notificaciones, shutdown), spec 04 (repositorio), spec 08 (`BondHistoryPort`), ms-cronjob-go (disparo) |

## 1. Resumen y alcance

Cuando un bono `initiated ('1')` llega a su `maturityDate`, pasa a `finalized ('2')`, es decir, madurado. Desde ese momento el inversionista lo puede canjear (spec 05). Además se le avisa con un aviso y un email por usuario que lista los bonos madurados.

Hoy esto lo hace un servicio externo (`MS_FINOBONDS_FEED_URL`, `127.0.0.1:8090/bound/check-maturity`) cuyo código no está en el workspace. Lo dispara un `SyncedCron` de Meteor que corre cada 30 s. Esta spec **trae la regla a ms-blooming-go** (caso de uso y endpoint interno) y deja la programación en ms-cronjob-go, que llama al endpoint cada 5 minutos. Juntos reemplazan a los dos.

## 2. Origen legacy

| Pieza | Archivo:línea | Destino |
|---|---|---|
| Job `BondBoundCheckMaturity` (cada 30 s, `PUT …/bound/check-maturity` + email + notice) | `backend/server/lib/crons.js:8-65` | RF-10.1, RF-10.5 |
| `bondBound.checkMaturity` (comentado; referencia de la regla) | `backend/server/model/bond.bound.js:1400-1430` | RF-10.3 |
| `Text.BOND_BOUND_MESSAGE_MATURITY` / `_SUCESS` | `backend/server/lib/text.js:100-105` | RF-10.5 |
| Arranque de `SyncedCron` (solo después de editar un cronjob) | `backend/server/model/cronjob.js:107,140,156`, `server/main.js` | Reemplazado por RF-10.1 |

## 3. Requisitos funcionales

- **RF-10.1** — Programación (en ms-cronjob-go).
  - ms-blooming-go no programa jobs ni usa una librería de cron (constitución §12).
  - ms-cronjob-go registra un `scheduler.Job` cada 300 s que hace `POST {MS_BLOOMING_URL}/internal/jobs/bond-maturity/run` con `x-secret` (RF-10.7), igual que el job de auditoría de Telegram de petty-cash (FA-359). Si falta `MS_BLOOMING_URL`, el adapter queda deshabilitado y lo avisa con un warning al arrancar.
  - No reintenta: la corrida es idempotente y el próximo tick la repite. Un `409 ERR_JOB_LOCKED` se loguea como informativo; cualquier otro status, como error.
  - Si ms-cronjob-go corta la conexión por timeout, la corrida sigue en ms-blooming-go hasta terminar.
- **RF-10.2** — Exclusión entre réplicas.
  - Antes de procesar, el job toma un lock en `bondJobLock` con `updateOne({_id: "bond-maturity", lockedUntil: {$lt: now}}, {$set: {lockedUntil: now + 10min, owner: hostname}}, upsert)`. Si el upsert choca por clave duplicada, el lock está tomado.
  - Si no lo obtiene, termina sin hacer nada.
  - Al final libera el lock: `lockedUntil = now`, solo si `owner` coincide.
- **RF-10.3** — Selección y transición.
  - Procesa en lotes de 500 los bonos con `{isRemove: false, statusId: '1', maturityDate: {$lte: now}, soldDate: null, tookDate: null}`.
  - Cada bono pasa a `finalized` con `UpdateStatusIf({_id, statusId: '1', soldDate: null, tookDate: null}, {statusId: '2', matured: true, maturedAt: now})`.
  - **Solo** se maduran bonos en estado `'1'`. Corrige B22: el legado incluía los estados vendido `'5'` y desligado `'4'`.
  - Si `UpdateStatusIf` no modifica el bono (otro proceso lo vendió entre medio), el bono se omite.
- **RF-10.4** — Historial. Por cada bono madurado: `BondHistory.Add` en modo sistema con `{relatedId, typeId: '2', text: 'BOND_BOUND_STATUS_CHANGED', params: ['1','2'], userId: "", agencyId: bound.agencyId}`.
  - Con `userId ""`, la auditoría del gateway (`B_BondBound_Audit_details`, que agrupa por `(userId, relatedId)`) no mezcla la maduración con las ventas y los canjes del usuario, y no genera falsos positivos de doble cobro.
  - La transición y el historial de cada bono van en la misma transacción: si el historial falla, el bono queda en `'1'`, cuenta como fallido y se reintenta en la próxima corrida.
- **RF-10.5** — Notificaciones por usuario (best-effort, después de la transición).
  - Los bonos madurados en la corrida se agrupan por `userId`.
  - Para cada usuario se envía:
    - un `Notice` tipo `bond`, con la `agencyId` del usuario, título `BOND_BOUND_MESSAGE_MATURITY` y mensaje `BOND_BOUND_MESSAGE_MATURITY_SUCESS[first_name, last_name, tabla]`;
    - el email `notify-blooming-bondMatured`.
  - La tabla tiene una fila por bono: `Bono (key) | Tipo (bondType.name) | Nombre (bondObject.name)`, con el HTML del legado.
  - El email sale sin sesión, por la ruta interna de ms-notification que solo valida `x-secret` (BLM-0140). Mientras no exista, ms-notification responde 401 y el fallo solo se loguea; el `notice` se guarda igual.
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
para cada lote de Repo.FindMaturable(now, 500):
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
- `BondBoundMongoccRepositoryPorts`: `FindMaturable(ctx, now, limit, afterID)` y `UpdateStatusIf` (BLM-0077).
- `JobLockPort`: `Acquire(ctx, name, ttl) (bool, error)` y `Release(ctx, name)`, en `shared_ports`.
- `BondHistoryPort.Add` en modo sistema (spec 08).
- `NoticePort`, `NotificationPort`, `ProfileReader` (spec 00) y lectura de los nombres de `bondType` y `bondObject` (specs 01 y 02).
- ms-cronjob-go: adapter HTTP `blooming_api` y job `blooming-bond-maturity` (300 s), con `MS_BLOOMING_URL`, `MS_BLOOMING_SECRET` y `MS_BLOOMING_DEBUG`.

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| B22 | La regla legacy (`checkMaturity`) maduraba también bonos **vendidos** (`'5'`) y desligados (`'4'`). Un bono vendido volvía a `'2'` y se podía canjear de nuevo: es el doble cobro que audita `B_BondBound_Audit_doubleProcess`. | Corregir (solo `'1'` y sin `soldDate`/`tookDate`) | Dinero (crítico). | BLM-0118 |
| B32 | `SyncedCron.start()` solo corre en `Cron.init()`, que está comentado en `main.js`. La maduración solo arranca después de editar un cronjob, y depende de un servicio externo que no está versionado en el workspace. | Corregir | Los bonos podían no madurar nunca. | BLM-0141 |
| B33 | Con 3 réplicas sin lock, la maduración y los avisos se ejecutarían tres veces. | Corregir (prevención) | Avisos duplicados. | BLM-0119 |

## 9. Criterios de aceptación

- **CA-10.1** (RF-10.1) — En ms-cronjob-go, con `MS_BLOOMING_URL` vacío el job no llama a nada y hay un warning al arrancar. Con la URL, cada tick hace `POST …/internal/jobs/bond-maturity/run` con `x-secret` y loguea `{matured, users, failed}`; un `409` se loguea como informativo y un `500`, como error.
- **CA-10.2** (RF-10.2) — Dos ejecuciones concurrentes (que simulan dos réplicas) dan este resultado: una procesa y la otra termina con `ERR_JOB_LOCKED` sin modificar nada. Un lock vencido (`lockedUntil` en el pasado) se puede volver a tomar.
- **CA-10.3** (RF-10.3) — Dados estos bonos:
  - A: `'1'`, vence ayer;
  - B: `'1'`, vence mañana;
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
| BLM-0140 | **(repo ms-notification)** Ruta interna de notificaciones que solo valida `x-secret`; `NotificationClient` la usa cuando el ctx no tiene sesión | impl | F7 | RF-10.5 | BLM-0014, BLM-0015 | `ms-notification/src/api/modules/notify/notify.routes.js`, `internal/core/shared/infrastructure/notification/notification.client.go` |
| BLM-0141 | **(repo ms-cronjob-go)** Adapter `blooming_api` y job `blooming-bond-maturity` cada 300 s que llama al endpoint interno de maduración | impl | F7 | RF-10.1 | BLM-0122 | `ms-cronjob-go/internal/core/infrastructure/adapter/blooming_api/`, `internal/core/application/bond_maturity/`, `internal/core/port/`, `internal/core/domain/`, `cmd/api/main.go`, `cmd/api/config/enviroments.config.go`, `.env.example`, `README.md`, `CLAUDE.md` |
| BLM-0142 | **(repo ms-cronjob-go)** Tests del adapter (`httptest`) y del job: URL vacía, POST con `x-secret`, 409 informativo y error | test | F7 | CA-10.1 | BLM-0141 | `ms-cronjob-go/internal/core/infrastructure/adapter/blooming_api/blooming_api_test.go`, `internal/core/application/bond_maturity/*_test.go` |

## 11. Fuera de alcance y preguntas abiertas

- **Pregunta:** ¿qué hace hoy el servicio `:8090` (`MS_FINOBONDS_FEED_URL`)? ¿Escribe historial, o madura también estados `'5'`/`'4'`? Hay que revisarlo antes del corte, porque con los datos actuales puede haber bonos vendidos que pasaron a `'2'`. Se agrega una tarea de auditoría (BLM-0134).
- **Decisión (Pregunta 4 de la spec 00):** el email de maduración sale sin sesión, y `POST /notify/:group/:notification` de ms-notification exige `x-token`. Se agrega a ms-notification una ruta interna que solo valida `x-secret`, y `NotificationClient` la usa cuando el ctx no tiene sesión (BLM-0140). El corte (BLM-0134) depende de esa tarea.
- **Decisión (programación):** la maduración la programa ms-cronjob-go y no el ms, para no sumar una librería de cron (constitución §1 y §12). El scheduler interno con `robfig/cron/v3` (BLM-0121) se retiró sin llegar a commitearse; su ID no se reutiliza.
- **Pregunta:** ¿la frecuencia de 5 minutos es suficiente? El legado corría cada 30 s, pero los avisos se pueden atrasar unos minutos sin impacto.
