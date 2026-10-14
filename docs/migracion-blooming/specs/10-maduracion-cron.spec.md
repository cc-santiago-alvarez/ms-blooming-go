# Spec 10 — Maduración de bonos (cron)

| | |
|---|---|
| Estado | Borrador |
| Dominio | `internal/core/bondBound` (caso de uso), `internal/cron` (scheduler) |
| Colecciones | `bondBound`, `bondJobLock` (nueva) |
| Depende de | Spec 00 (notificaciones, shutdown), spec 04 (repositorio), spec 08 (`BondHistoryPort`) |

## 1. Resumen y alcance

Cuando un bono `initiated ('1')` llega a su `maturityDate`, pasa a `finalized ('2')`, es decir, madurado. Desde ese momento el inversionista lo puede canjear (spec 05). Además se le avisa con un aviso y un email por usuario que lista los bonos madurados.

Hoy esto lo hace un servicio externo (`MS_FINOBONDS_FEED_URL`, `127.0.0.1:8090/bound/check-maturity`) cuyo código no está en el workspace. Lo dispara un `SyncedCron` de Meteor que corre cada 30 s. Esta spec lo **trae a ms-blooming-go** y reemplaza los dos.

## 2. Origen legacy

| Pieza | Archivo:línea | Destino |
|---|---|---|
| Job `BondBoundCheckMaturity` (cada 30 s, `PUT …/bound/check-maturity` + email + notice) | `backend/server/lib/crons.js:8-65` | RF-10.1, RF-10.5 |
| `bondBound.checkMaturity` (comentado; referencia de la regla) | `backend/server/model/bond.bound.js:1400-1430` | RF-10.3 |
| `Text.BOND_BOUND_MESSAGE_MATURITY` / `_SUCESS` | `backend/server/lib/text.js:100-105` | RF-10.5 |
| Arranque de `SyncedCron` (solo después de editar un cronjob) | `backend/server/model/cronjob.js:107,140,156`, `server/main.js` | Reemplazado por RF-10.1 |

## 3. Requisitos funcionales

- **RF-10.1** — Programación.
  - Al arrancar se registra el job `bond-maturity` en el scheduler `robfig/cron/v3`, con la expresión de `BOND_MATURITY_CRON` (recomendada: `*/5 * * * *`). Si la variable está vacía, el job no se registra y queda un log informativo.
  - El scheduler se detiene en el graceful shutdown y espera al job en curso (RF-00.16).
- **RF-10.2** — Exclusión entre réplicas.
  - Antes de procesar, el job toma un lock en `bondJobLock` con `findOneAndUpdate({_id: "bond-maturity", lockedUntil: {$lt: now}}, {$set: {lockedUntil: now + 10min, owner: hostname}}, upsert)`. Si el upsert choca por clave duplicada, el lock está tomado.
  - Si no lo obtiene, termina sin hacer nada.
  - Al final libera el lock: `lockedUntil = now`, solo si `owner` coincide.
- **RF-10.3** — Selección y transición.
  - Procesa en lotes de 500 los bonos con `{isRemove: false, statusId: '1', maturityDate: {$lte: now}, soldDate: null, tookDate: null}`.
  - Cada bono pasa a `finalized` con `UpdateStatusIf({_id, statusId: '1', soldDate: null, tookDate: null}, {statusId: '2', matured: true, maturedAt: now})`.
  - **Solo** se maduran bonos en estado `'1'`. Corrige B22: el legado incluía los estados vendido `'5'` y desligado `'4'`.
  - Si `UpdateStatusIf` no modifica el bono (otro proceso lo vendió entre medio), el bono se omite.
- **RF-10.4** — Historial. Por cada bono madurado: `BondHistory.Add` en modo sistema con `{relatedId, typeId: '2', text: 'BOND_BOUND_STATUS_CHANGED', params: ['1','2'], userId: "", agencyId: bound.agencyId}`.
  - Con `userId ""`, la auditoría del gateway (`B_BondBound_Audit_details`, que agrupa por `(userId, relatedId)`) no mezcla la maduración con las ventas y los canjes del usuario, y no genera falsos positivos de doble cobro.
- **RF-10.5** — Notificaciones por usuario (best-effort, después de la transición).
  - Los bonos madurados en la corrida se agrupan por `userId`.
  - Para cada usuario se envía:
    - un `Notice` tipo `bond`, con la `agencyId` del usuario, título `BOND_BOUND_MESSAGE_MATURITY` y mensaje `BOND_BOUND_MESSAGE_MATURITY_SUCESS[first_name, last_name, tabla]`;
    - el email `blooming-bondMatured`.
  - La tabla tiene una fila por bono: `Bono (key) | Tipo (bondType.name) | Nombre (bondObject.name)`, con el HTML del legado.
- **RF-10.6** — Robustez y observabilidad.
  - Si falla un bono, se loguea con su `_id` y el lote sigue.
  - Al terminar se loguea `{matured, users, failed, durationMs}`.
  - Una corrida nunca deja el lock tomado más allá de su TTL.
- **RF-10.7** — Ejecución manual. `POST /internal/jobs/bond-maturity/run` (`x-secret`, `NoRequiresAuthentication`) ejecuta la misma lógica con el lock y responde `{matured, users, failed}`. Sirve para operación y pruebas.

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
    ok, err = Repo.UpdateStatusIf(b._id, from '1', to '2', set {matured: true, maturedAt: now})
    si err → log, failed++ ; continue
    si !ok → continue                          // ya no estaba en '1'
    BondHistory.Add(systemCtx(b.agencyId), {relatedId: b._id, typeId: '2', text: 'BOND_BOUND_STATUS_CHANGED', params: ['1','2']})
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
- `BondBoundMongoccRepositoryPorts`: `FindMaturable(ctx, now, limit, afterID)` y `UpdateStatusIf` (T-04.3).
- `JobLockPort`: `Acquire(ctx, name, ttl) (bool, error)` y `Release(ctx, name)`, en `shared_ports`.
- `BondHistoryPort.Add` en modo sistema (spec 08).
- `NoticePort`, `NotificationPort`, `ProfileReader` (spec 00) y lectura de los nombres de `bondType` y `bondObject` (specs 01 y 02).
- `internal/cron`: scheduler, registro del job y parada en el shutdown.

## 8. Bugs legacy

| Bug | Descripción | Decisión | Motivo | Tarea |
|---|---|---|---|---|
| B22 | La regla legacy (`checkMaturity`) maduraba también bonos **vendidos** (`'5'`) y desligados (`'4'`). Un bono vendido volvía a `'2'` y se podía canjear de nuevo: es el doble cobro que audita `B_BondBound_Audit_doubleProcess`. | Corregir (solo `'1'` y sin `soldDate`/`tookDate`) | Dinero (crítico). | T-10.2 |
| B32 | `SyncedCron.start()` solo corre en `Cron.init()`, que está comentado en `main.js`. La maduración solo arranca después de editar un cronjob, y depende de un servicio externo que no está versionado en el workspace. | Corregir | Los bonos podían no madurar nunca. | T-10.5 |
| B33 | Con 3 réplicas sin lock, la maduración y los avisos se ejecutarían tres veces. | Corregir (prevención) | Avisos duplicados. | T-10.3 |

## 9. Criterios de aceptación

- **CA-10.1** (RF-10.1) — Con `BOND_MATURITY_CRON` vacío, el job no se registra y hay un log informativo. Con `*/5 * * * *` se registra. Si llega SIGTERM durante una corrida, el proceso espera a que termine.
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

| ID | Tarea | Tipo | Cubre | Depende de | Archivos |
|---|---|---|---|---|---|
| T-10.1 | Dominio: `MaturityResult` y función pura `BuildMaturityTable(rows)` (HTML del legado) | impl | RF-10.3, RF-10.5 | T-04.1 | `internal/core/bondBound/domain/bondBound.maturity.go` |
| T-10.2 | Adapter `FindMaturable` (lotes con cursor por `_id`), con transición condicional solo desde `'1'` | impl | RF-10.3 | T-04.3 | `internal/core/bondBound/infrastructure/mongocc/bondBound.findMaturable.adapter.go` |
| T-10.3 | `JobLockPort` y su adapter mongocc (`bondJobLock`) | impl | RF-10.2 | T-00.10 | `internal/core/shared/ports/shared.jobLock.ports.go`, `internal/core/shared/infrastructure/mongocc/shared.jobLock.adapter.go` |
| T-10.4 | Caso de uso `RunMaturity`: lock, lotes, transición, historial de sistema, agrupación y notificaciones | impl | RF-10.3, RF-10.4, RF-10.5, RF-10.6 | T-10.1, T-10.2, T-10.3, T-08.4, T-00.13, T-00.14, T-00.15 | `internal/core/bondBound/application/bondBound.runMaturity.go` |
| T-10.5 | Scheduler `internal/cron` (robfig, `BOND_MATURITY_CRON`, parada en el shutdown) y registro del job | impl | RF-10.1 | T-10.4, T-00.19 | `internal/cron/infra/scheduler.go`, `internal/cron/jobs/bondMaturity.job.go`, `cmd/api/server/server.go` |
| T-10.6 | Endpoint interno de ejecución manual | impl | RF-10.7 | T-10.4, T-00.4 | `cmd/api/handlers/jobs/jobs.bondMaturity.post.go` |
| T-10.7 | Tests unitarios: tabla HTML, agrupación por usuario, estados excluidos, fallo parcial | test | CA-10.3, CA-10.6, CA-10.7 | T-10.4 | `internal/core/bondBound/application/bondBound.runMaturity_test.go`, `internal/core/bondBound/domain/bondBound.maturity_test.go` |
| T-10.8 | Tests de integración: idempotencia, lock concurrente, historial de sistema, registro del cron y endpoint | test | CA-10.1, CA-10.2, CA-10.4, CA-10.5, CA-10.8, CA-10.9 | T-10.5, T-10.6 | `internal/core/bondBound/application/bondBound.runMaturity_integration_test.go`, `internal/cron/infra/scheduler_test.go` |

## 11. Fuera de alcance y preguntas abiertas

- **Pregunta:** ¿qué hace hoy el servicio `:8090` (`MS_FINOBONDS_FEED_URL`)? ¿Escribe historial, o madura también estados `'5'`/`'4'`? Hay que revisarlo antes del corte, porque con los datos actuales puede haber bonos vendidos que pasaron a `'2'`. Se agrega una tarea de auditoría (T-CORTE.2).
- **Pregunta:** ¿la frecuencia de 5 minutos es suficiente? El legado corría cada 30 s, pero los avisos se pueden atrasar unos minutos sin impacto.
