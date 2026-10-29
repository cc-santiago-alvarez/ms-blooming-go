# Corte de la maduración (BLM-0134)

| | |
|---|---|
| Estado | Procedimiento ensayado en local (§8). Testing y live, pendientes |
| Servicios | backend Meteor (job `BondBoundCheckMaturity`), ms-cronjob-go (job `blooming-bond-maturity`), ms-blooming-go, ms-notification, ms-feed |
| Variables | `MS_FEED_MATURITY` en `/opt/backend-meteor/.env`; `MS_BLOOMING_URL`, `MS_BLOOMING_SECRET` y `MS_BLOOMING_DEBUG` en `/opt/ms-cronjob-go/.env`; en testing, `DEBUG_MODE` y `DEBUG_EMAILS` en `/opt/ms-notification/.env` |
| Antes | [Auditoría de datos](auditoria-datos-corte.md), §8 pasos 1 a 5 |
| Después | Auditoría §8 paso 6; BLM-0137 retira el job y su variable, y BLM-0152 apaga ms-feed |
| Decisiones | Spec 10 (RF-10.1 y §11); [README](README.md) §6 |

## 1. Qué cambia

Hoy madura ms-feed: el job SyncedCron `BondBoundCheckMaturity` de Meteor llama cada 30 s a `PUT /bound/check-maturity` y manda el aviso y el email con la respuesta. Después del corte madura ms-blooming-go: ms-cronjob-go llama cada 300 s a `POST {MS_BLOOMING_URL}/internal/jobs/bond-maturity/run`.

En cada entorno, el corte son dos pasos, siempre en este orden:
1. **Meteor deja de correr el job:** `MS_FEED_MATURITY=false` y un reinicio.
2. **ms-cronjob-go empieza a llamar a ms-blooming-go:** `MS_BLOOMING_URL` y un reinicio, recién cuando ninguna réplica de Meteor corre el job.

- **El hueco entre los dos pasos no hace daño.** La primera corrida de Go madura todo lo que esté vencido, así que el hueco solo atrasa la maduración.
- **El solape sí hace daño.** El `updateMany` de ms-feed no repite la condición de estado. Un bono que Go madura entre la consulta y la actualización de ms-feed se avisa dos veces, y los que madura ms-feed quedan sin historial ni `maturedAt`.
- **Ningún paso necesita un release.** El código de la variable ya está desplegado (prerrequisito 1) y, mientras está vacía, el job sigue como siempre. Los `.env` se montan desde el host y se leen al arrancar. Así, testing y live se cortan cuando DevOps decide, y el rollback es volver las variables atrás.

## 2. Prerrequisitos

En cada entorno, antes de la ventana:

| # | Qué | Cómo se verifica |
|---|---|---|
| 1 | Backend con la variable `MS_FEED_MATURITY` (el commit de BLM-0134 en `server/lib/crons.js` y `_constants.js`) | La imagen de `BACKEND_futurapps` es de un release que incluye ese commit |
| 2 | ms-blooming-go con BLM-0164, que lee los bonos uno por uno (y sus tests, BLM-0165), y el índice `{statusId: 1, maturityDate: 1}` en `bondBound` | Release de ms-blooming-go; consulta P |
| 3 | ms-notification con la ruta interna (`POST /api/v1/notify/internal/:group/:notification`) y el handler `notify-blooming-bondMatured` | Chequeos en seco 3 y 4 (§3) |
| 4 | ms-cronjob-go con BLM-0141 y BLM-0142, y `MS_BLOOMING_URL` vacío | Su log de arranque tiene el WARN `Blooming api adapter init failed: base url is required (job blooming-bond-maturity no registrado)` |
| 5 | Mongo con replica set: cada bono madura en una transacción | Consulta P |
| 6 | Auditoría §8 pasos 1 a 4, y el paso 5 (`auditdata -at <fecha del corte>`): 0 bloqueantes entre los `'1'` y 0 que Go no decodifica | Informe de la auditoría |
| 7 | Dueños de la primera corrida y dueños por día de vencimiento: menos de 80 | Consultas B y G2. ms-notification limita `/notify/internal` a 100 requests por minuto por IP, y ms-blooming-go no se salta ese límite. Con más dueños, los emails reciben `429` y se pierden, así que hay que resolverlo antes del corte |
| 8 | Valores para ms-cronjob-go | `MS_BLOOMING_URL`: URL de ms-blooming-go **del mismo entorno**, **con** `/api/v1`, alcanzable desde el contenedor. El cronjob no está en la red `futurapps-microservices`, así que es la IP de un nodo con el puerto publicado, o un proxy con timeout de al menos 600 s. `MS_BLOOMING_SECRET`: el `API_SECRET` efectivo de ms-blooming-go (su `.env.<ENV>` pisa al `.env`). Chequeos en seco 1 y 2 |
| 9 | `stop_grace_period` de ms-blooming-go de al menos 30 s: los avisos salen al final de la corrida, y un apagado brusco deja bonos madurados sin aviso ni email | `service inspect` del servicio de ms-blooming-go con `--format '{{json .Spec.TaskTemplate.ContainerSpec.StopGracePeriod}}'` da `30000000000` o más (`null` es el default de 10 s) |
| 10 | **Solo testing:** el ms-notification al que apunta `MS_NOTIFICATION_URL` de ms-blooming-go corre con `DEBUG_MODE=true` y `DEBUG_EMAILS=<buzón del equipo>` | Ver abajo. La base de testing es un clon de producción con datos personales reales y tiene un atraso de bonos `'1'` vencidos (497 de 12 dueños en el dump): sin esto, la primera corrida y las siguientes les mandan el email a esas personas |

**Prerrequisito 10, cómo se activa y se verifica.**
1. En `/opt/ms-notification/.env` de todos los nodos, antes de la ventana. Por nombre y sin imprimir valores: `grep '^DEBUG_MODE=' /opt/ms-notification/.env | tail -1` da `DEBUG_MODE=true` (si la clave está dos veces, gana la última) y `grep -c '^DEBUG_EMAILS=.' /opt/ms-notification/.env` da `1`.
2. ms-notification lee su `.env` solo al arrancar: `docker -H <servidor>:12375 service update --force NOTIFICATION_ms`, y `service ps NOTIFICATION_ms --filter desired-state=running` muestra tareas creadas después de editar el archivo.
3. El chequeo en seco 4 lo prueba en el proceso: el email de prueba llega al buzón del equipo.
4. Si no se puede activar, `MS_NOTIFICATION_URL` de ms-blooming-go de testing apunta a una dirección sin servicio (`http://127.0.0.1:9/api/v1`), con su reinicio antes de la ventana, y queda así hasta que ms-notification de testing tenga `DEBUG_MODE`: no se restaura después de la primera corrida. Con eso, los emails de ms-blooming-go de testing fallan y se loguean, y los avisos se guardan igual. El email de punta a punta no se valida en testing.

Además:
- **Ventana.** Lejos de las 00:00 UTC (19:00 en Colombia), porque a esa hora madura todo lo que vence en el día. Durante la ventana no se despliegan ms-blooming-go ni ms-notification.
- **Comandos.** `<servidor>` es el `DEPLOY_SERVER` del `docker-stack-deploy.sh` de cada servicio, y todos, backend incluido, exponen la API de Docker en el puerto `12375`: `docker -H <servidor>:12375 …`. Las consultas del apéndice son de solo lectura y usan el reloj del servidor de Mongo.
- **Logs con datos sensibles.**
  - ms-cronjob-go imprime al arrancar la URL de Mongo con sus credenciales (`mongocc` v1.2.13): sus logs se leen con `| grep -v 'connected to Mongo'`.
  - Las líneas `[Mail] Correo enviado a:` de ms-notification traen el email de cada destinatario: en live se cuentan con `grep -c`.
  - Ningún log se pega crudo, y §7 lleva solo números.

## 3. Chequeos en seco

Validan URL, secret, rutas y envío sin correr la maduración. Se hacen el día del corte, antes del paso 2 del procedimiento.

1. **URL.** Desde el nodo del cronjob (su imagen es `scratch` y no trae `curl`):
   ```bash
   curl -s -o /dev/null -w '%{http_code}\n' -X POST "<MS_BLOOMING_URL>/internal/jobs/bond-maturity/run"   # sin x-secret: 401
   ```
   El `401` solo no alcanza: con `API_DEBUG=false`, ms-blooming-go responde `401` sin `x-secret` a cualquier ruta, también a una que no existe (con `API_DEBUG=true`, una que no existe da `404 Endpoint not found`). En el log del ms-blooming-go **de ese mismo entorno** tiene que aparecer exactamente `[ValidateApiSecret] x-secret ausente en POST /api/v1/internal/jobs/bond-maturity/run`. Si la línea trae otra ruta, o no aparece, la URL está mal.
2. **Secret.** La huella de `MS_BLOOMING_SECRET` tiene que coincidir con la del `API_SECRET` efectivo de ms-blooming-go, sin imprimir ninguno:
   ```bash
   grep '^API_SECRET=' <.env efectivo> | cut -d= -f2- | tr -d '[:space:]' | sha256sum
   ```
3. **ms-notification, en seco.** Con `MS_NOTIFICATION_URL` y `MS_NOTIFICATION_SECRET` del `.env` efectivo de ms-blooming-go, sin imprimir el secret:
   ```bash
   printf 'header = "x-secret: %s"\n' "$(grep '^MS_NOTIFICATION_SECRET=' <.env efectivo> | cut -d= -f2-)" \
     | curl -s -K - -X POST -H 'Content-Type: application/json' -d '{"data":{"params":{}}}' -w '\n%{http_code}\n' \
       "<MS_NOTIFICATION_URL>/notify/internal/user-ids/notify-blooming-bondMatured"
   ```
   - `400` con `\"userIds\" required in @params` en el cuerpo: ruta, secret y handler están bien. No manda nada ni lee la base.
   - `404`: falta la ruta. `400` con `do not exists`: falta el handler. `401`: el secret no coincide. `500`: faltó el `Content-Type`.
4. **ms-notification, con un envío real a un perfil del equipo.** El mismo comando, con body `{"data":{"params":{"userIds":["<_id de un perfil del equipo>"],"bonds":[]}}}`. Responde `202`, y en el log del ms-notification **de ese entorno** (`docker -H <servidor>:12375 service logs --since <hora> NOTIFICATION_ms`) aparece una línea `[Mail] Correo enviado a:`, sin `Notification errors`. En testing tiene que mostrar el buzón del equipo, y en live, el email de ese perfil. Prueba que el proceso tiene `DEBUG_MODE` en testing, que la URL es la de ese entorno y que la plantilla de SendGrid funciona.

## 4. Procedimiento

Primero en testing. En live, recién con testing cerrado (§7).

1. **B0, de referencia.** Consulta B, para tener el orden de magnitud.
2. **Apagar el job de Meteor.**
   1. `MS_FEED_MATURITY=false` en `/opt/backend-meteor/.env` de **todos** los nodos: una sola vez, en su propia línea, sin espacios alrededor del `=`, sin comentario en la misma línea y sin comillas tipográficas. El contenedor exporta el archivo con `export $(grep -v '^#' .env | xargs)`: un `#` o un `=` sueltos hacen fallar el `export` y Meteor no arranca, y `MS_FEED_MATURITY= false` deja la variable vacía.
   2. En cada nodo, `sh -c 'export $(grep -v "^#" /opt/backend-meteor/.env | xargs) && printenv MS_FEED_MATURITY'` imprime `false`, y `sha256sum /opt/backend-meteor/.env` da lo mismo en todos.
   3. `docker -H <servidor>:12375 service update --force BACKEND_futurapps`, sin `-d`. Reinicia las réplicas de a una: 6 en live (unos 7 min) y 2 en testing. Hasta que cae la última réplica vieja, el job sigue corriendo.
3. **Portón: Meteor ya no corre el job.** Tiene que cumplirse todo antes de seguir.
   - `docker -H <servidor>:12375 service inspect BACKEND_futurapps --format '{{.UpdateStatus.State}}'` da `completed`. Con `rollback_completed` o `paused`, alguna réplica sigue con el job: no se sigue.
   - `docker -H <servidor>:12375 service ps BACKEND_futurapps --filter desired-state=running` muestra solo tareas creadas después del paso 2. Si alguna tarea nueva no arranca (`Failed` o `Rejected`), el `.env` está mal formado: se corrige en todos los nodos, se valida como en 2.2 y se repite el paso 2. Mientras tanto, las réplicas viejas siguen con el job.
   - `docker -H <servidor>:12375 service logs --since <inicio del paso 2> BACKEND_futurapps 2>&1 | grep -c 'BondBoundCheckMaturity: deshabilitado con MS_FEED_MATURITY=false'` da al menos el número de réplicas.
   - La consulta A da 0 durante 2 minutos seguidos. Si ya daba 0 antes del paso 2, Meteor no corría el job en ese entorno (el dump de testing tiene `cronHistory` vacío) y el portón descansa en los tres puntos anteriores.
4. **Bloquear ms-feed y tomar B1.**
   1. DevOps bloquea `PUT /bound/check-maturity` de ms-feed en el proxy, y en el puerto `8090` publicado si se alcanza sin pasar por el proxy. No tiene autenticación, devuelve el nombre y el email de cada dueño, y desde ahora cualquier llamada maduraría bonos sin historial ni avisos con Go activo. Meteor ya no lo llama. Los `GET` de las gráficas siguen hasta BLM-0136 y BLM-0152.
   2. Consulta B justo antes del paso 5: bonos, dueños, dueños sin perfil y la lista `base`. La sesión de mongosh queda abierta hasta el paso 7.
5. **Activar el job de ms-cronjob-go.**
   1. En `/opt/ms-cronjob-go/.env` de **todos** los nodos (la tarea no está fijada a un nodo): `MS_BLOOMING_URL`, `MS_BLOOMING_SECRET` y, durante el corte, `MS_BLOOMING_DEBUG=true`, que loguea la URL y el status de cada tick. `sha256sum` da lo mismo en todos.
   2. `docker -H <servidor>:12375 service update --force CRONJOB_ms`.
   3. En `docker -H <servidor>:12375 service logs --since <hora del --force> CRONJOB_ms 2>&1 | grep -v 'connected to Mongo'` ya no aparece el WARN de blooming. Registrar el job no escribe nada.
6. **Primer tick, 300 s después del reinicio.**
   - ms-cronjob-go: `[Job: blooming bond maturity] Running`, `[BloomingApi] [Response] Status: 200` y `[BondMaturity.RunMaturity] N bono(s) madurado(s) de M usuario(s), F fallido(s)`.
   - ms-blooming-go: `[bondBound] maduración: matured=N users=M failed=F durationMs=…` y `POST /api/v1/internal/jobs/bond-maturity/run 200`. Tantas líneas `ms-notification respondió 400` como dueños sin perfil en B1, que reciben `400 Users not found`, y ninguna otra de `[NotificationPort]`. Con la alternativa del prerrequisito 10, en testing cada dueño da `[NotificationPort] … connection refused`, y es lo esperado.
   - ms-notification, que solo loguea los requests con status ≥ 400: tantas líneas `[Mail] Correo enviado a:` como M menos los dueños sin perfil, y ninguna `Notification errors` ni `Rate limit exceeded`. Un `Notification errors` es un email que SendGrid rechazó: ms-notification igual responde `202` y ms-blooming-go no lo ve. En testing, esas líneas muestran solo el buzón del equipo.
   - Se espera `N + F + los vendidos o canjeados entre B1 y el tick = bonos de B1`.
7. **Verificación en Mongo.** Consultas C, D, E y F:
   - todos los bonos de `base` en `'2'` con `maturedAt`, o todavía en `'1'` si fallaron;
   - ninguno en `'2'` sin `maturedAt`, que sería un bono madurado por ms-feed;
   - un historial de sistema por bono, un aviso por dueño y el lock libre.
8. **Segundo tick.** `0 bono(s) madurado(s) de 0 usuario(s)` y los mismos `F` fallidos del primer tick (0 si no hubo), sin avisos nuevos. La consulta A sigue en 0.
9. **Cierre.**
   1. Opcional: `MS_BLOOMING_DEBUG=false` en todos los nodos y otro `--force` del cronjob, que reinicia la cuenta de 300 s.
   2. `auditdata -apply -confirm <ENV>/<base> -only vendidos` (auditoría §8 paso 6): el `updateMany` de ms-feed pudo dejar un `'2'` con `soldDate` antes de apagarse.
   3. Confirmar en la configuración del proxy que el bloqueo del paso 4 sigue, sin hacer un `PUT` de prueba.
   4. Al día siguiente, el primer tick después de las 00:00 UTC madura los bonos que vencen ese día, sin `429` ni `Notification errors`.
   5. Se completa §7.

## 5. Seguimiento

Hasta el cierre de BLM-0138, con alertas o con una revisión diaria:
- `RunMaturity -> Error` en ms-cronjob-go, que es cualquier respuesta distinta de `200` y `409`;
- `failed` mayor que 0, que ms-cronjob-go loguea como información y no como error;
- `no se pudo madurar el bono` y `no se pudo leer` en ms-blooming-go;
- `bondJobLock.updatedAt` quieto por más de 10 min (consulta F): el cronjob no está llegando;
- `respondió 429` en ms-blooming-go, y `Rate limit exceeded` o `Notification errors` en ms-notification: emails perdidos, que ms-blooming-go da por enviados o solo loguea;
- la consulta A mayor que 0: el job de Meteor volvió y conviven las dos maduraciones. Se corrige el `.env` de ese nodo y se hace `--force` del backend.

## 6. Rollback

Primero se apaga Go y después se reactiva Meteor, nunca al revés:
1. Vaciar `MS_BLOOMING_URL` en **todos** los nodos, con `sha256sum` igual en todos (si la tarea se reprograma en un nodo que conserva la URL, Go vuelve con Meteor activo), y hacer `--force` de `CRONJOB_ms`. El WARN vuelve a aparecer en `service logs --since <hora del --force>`.
2. Esperar el lock libre (consulta F): una corrida en curso puede seguir hasta 10 min.
3. Desbloquear `PUT /bound/check-maturity` en el proxy.
4. Vaciar `MS_FEED_MATURITY` en todos los nodos y hacer `--force` de `BACKEND_futurapps`. Con el update en `completed`, ninguna réplica loguea `BondBoundCheckMaturity: deshabilitado` desde el rollback, y la consulta A vuelve a dar más de 0 en 2 minutos, salvo donde ya daba 0 antes del corte.

No hay datos que revertir: los bonos que maduró Go tienen historial y `maturedAt`, y ms-feed no toca bonos en `'2'`.

Después del corte, y hasta que BLM-0137 retire el job, ninguna réplica del backend puede arrancar sin `MS_FEED_MATURITY=false`: ni con una imagen anterior a BLM-0134, ni con un `/opt/backend-meteor/.env` restaurado de un respaldo previo al corte, ni en un nodo nuevo con un `.env` copiado sin esa línea. Vacía o ausente, la variable reactiva el job con Go activo.

| Resultado | Qué significa | Qué hacer |
|---|---|---|
| `failed` > 0 | Esos bonos siguen en `'1'` y se reintentan en cada tick. El log de ms-blooming-go da el `_id` y el motivo (por ejemplo, un `'1'` sin `agencyId`) | Corregir el dato. No se revierte |
| `401 TOKEN_IS_REQUIRED`, o `404 Endpoint not found` si ms-blooming-go tiene `API_DEBUG=true` | La URL no lleva `/api/v1` | Corregir la URL |
| `401 UNAUTHORIZED_INVALID_SECRET` | El secret no coincide | Corregir el secret |
| `409 ERR_JOB_LOCKED` | Otra corrida tiene el lock | Nada: el job lo loguea como información |
| `500 ERR_INTERNAL` | Falló el lock o la lectura de los bonos | Ver `[handler] error interno` en ms-blooming-go |
| Timeout o `504` | Lo cortó el cliente o el proxy; la corrida sigue en ms-blooming-go | Ver `[bondBound] maduración:` y subir el timeout del proxy |
| `404`, `400` (salvo dueños sin perfil) o `429` de ms-notification, o `Notification errors` | Los emails de esa corrida se pierden, porque el bono ya no está en `'1'` | Rollback inmediato, hasta corregirlo |
| Consulta C con `{'2', go: false}` > 0 | ms-feed maduró bonos con Go activo: un Meteor que el portón no vio, o una llamada directa | Bloquear ya `PUT /bound/check-maturity` (o hacer el rollback completo, nunca reactivar solo Meteor), buscar quién llama y listar esos bonos: no tienen historial ni aviso |

## 7. Resultados

| Entorno | Fecha | B1 (bonos / dueños) | Primera corrida | Segunda corrida | Observaciones |
|---|---|---|---|---|---|
| Local (ensayo, §8) | 2026-10-29 | 497 / 12 | `matured=497 users=12 failed=0`, 1012 ms | `matured=0 users=0 failed=0` | Emails desviados a un destino sin servicio |
| Testing | | | | | |
| Live | | | | | |

## 8. Ensayo local

El 2026-10-29 se ensayó el paso 5 en adelante con los binarios de ms-blooming-go (`3a6887b`) y de ms-cronjob-go (BLM-0141 y BLM-0142), contra `corte_ensayo`: una copia con `$out` de las 7 colecciones que lee la auditoría (las de bonos, `profile` y `balanceAccount`), tomadas del dump local. ms-blooming-go corrió con un `API_SECRET` aleatorio y con ms-notification, OAuth y ms-exporter apuntando a `127.0.0.1:9`, así que ningún email salió.

| Paso | Resultado |
|---|---|
| `auditdata -db corte_ensayo -at 2026-10-29` | 497 bonos `'1'` vencidos de 12 dueños; 0 que ms-feed no madura; 0 que Go no decodifica |
| Chequeo en seco 1 | `401` sin `x-secret` y con uno inválido. Una URL sin `/api/v1` también da `401`: lo que distingue es la ruta en el log de `ValidateApiSecret` |
| Arranque del cronjob con la URL | Sin el WARN de blooming y sin ningún log del job |
| Primer tick (300 s) | `497 bono(s) madurado(s) de 12 usuario(s), 0 fallido(s)` en 1012 ms. Consultas C a F: 497 en `'2'` con `maturedAt`, 497 historiales de sistema, 12 avisos sin repetidos y el lock libre. Los 12 emails fallaron con `connection refused`, como se buscaba |
| Segundo tick | `0 bono(s) madurado(s) de 0 usuario(s), 0 fallido(s)`, sin historiales ni avisos nuevos |
| Lock tomado a mano | El `POST` con secret responde `409 ERR_JOB_LOCKED`, y el tick lo loguea como `otra corrida tiene el lock, se omite este tick`, sin error |
| `MS_BLOOMING_URL` vacía | WARN `Blooming api adapter init failed: base url is required (job blooming-bond-maturity no registrado)` |

Al terminar se borró `corte_ensayo`, y `futurapps` y `parity` mantuvieron su huella (`dbHash`).

## Apéndice: consultas

Son de solo lectura y van en una sola sesión de mongosh sobre la base del entorno: B define `corte`, `filtro`, `base` y `T0`, que usan las siguientes.

```js
var ahora = () => db.hello().localTime;   // reloj del servidor de Mongo

// P. Prerrequisitos: replica set e índice de la maduración
db.hello().setName;   // no vacío
db.bondBound.getIndexes().some(i => JSON.stringify(i.key) === '{"statusId":1,"maturityDate":1}');   // true

// A. Meteor ya no corre el job: 0 durante 2 minutos, y otra vez después de cada tick
db.cronHistory.find({name: 'BondBoundCheckMaturity'}, {_id: 0, intendedAt: 1, startedAt: 1, finishedAt: 1}).sort({intendedAt: -1}).limit(3);
db.cronHistory.countDocuments({name: 'BondBoundCheckMaturity', startedAt: {$gte: new Date(ahora().getTime() - 120000)}});

// B. Lo que madura la próxima corrida, con el filtro y el corte de FindMaturable
var t = ahora();
var corte = new Date(Date.UTC(t.getUTCFullYear(), t.getUTCMonth(), t.getUTCDate() + 1));
var filtro = {isRemove: false, statusId: '1', maturityDate: {$lt: corte}, soldDate: null, tookDate: null};
db.bondBound.countDocuments(filtro);                                                          // bonos
db.bondBound.aggregate([{$match: filtro}, {$group: {_id: '$userId'}}, {$count: 'duenos'}]);   // dueños (< 80)
db.bondBound.countDocuments({...filtro, $or: [{agencyId: {$not: {$type: 'string'}}}, {agencyId: ''}]});   // 0: sin agencyId fallan
db.bondBound.aggregate([{$match: filtro}, {$group: {_id: '$userId'}},
  {$lookup: {from: 'profile', localField: '_id', foreignField: '_id', as: 'p'}},
  {$match: {p: {$size: 0}}}, {$count: 'sinPerfil'}]);   // dueños sin perfil: 400 esperados en ms-notification
var base = db.bondBound.distinct('_id', filtro);
var T0 = ahora();

// C. Bonos de B después del primer tick
db.bondBound.aggregate([{$match: {_id: {$in: base}}},
  {$group: {_id: {statusId: '$statusId', go: {$eq: [{$type: '$maturedAt'}, 'date']}}, n: {$sum: 1}}}]);
// {'2', go: true} = matured; {'2', go: false} tiene que ser 0 (si no, lo maduró ms-feed); los '1' = failed
db.bondBound.countDocuments(filtro);   // = failed, si no se cruzaron las 00:00 UTC

// D. Historial de sistema: uno por bono madurado
var ids = db.bondBound.distinct('_id', {maturedAt: {$gte: T0}});
var hq = {createdAt: {$gte: T0}, typeId: '2', text: 'BOND_BOUND_STATUS_CHANGED', params: ['1', '2'], userId: ''};
db.bondHistory.countDocuments(hq);   // = ids.length
var conHist = new Set(db.bondHistory.distinct('relatedId', hq));
ids.filter(id => !conHist.has(id)).length;   // 0

// E. Avisos: uno por dueño
var duenos = db.bondBound.distinct('userId', {maturedAt: {$gte: T0}});
var avisos = db.notice.aggregate([{$match: {userId: {$in: duenos}, createdAt: {$gte: T0}, typeId: '3', title: 'Bono Madurado'}},
  {$group: {_id: '$userId', n: {$sum: 1}}}]).toArray();
avisos.length;                        // = duenos.length; menos: la corrida se cortó antes de avisar
avisos.filter(a => a.n > 1).length;   // 0; más: solape o corridas partidas

// F. Lock: libre si lockedUntil ya pasó; updatedAt avanza en cada tick
db.bondJobLock.findOne({_id: 'bond-maturity'}, {_id: 0, lockedUntil: 1, owner: 1, updatedAt: 1});
db.bondJobLock.countDocuments({_id: 'bond-maturity', lockedUntil: {$gt: ahora()}});   // 0

// G1. Zona horaria de ms-feed (informativa): hora UTC de los avisos de maduración (0: UTC; 4 o 5: América)
db.notice.aggregate([{$match: {typeId: '3', title: 'Bono Madurado', createdAt: {$gte: ISODate('2025-01-01')}}},
  {$group: {_id: {$hour: '$createdAt'}, n: {$sum: 1}}}, {$sort: {n: -1}}, {$limit: 3}]);

// G2. Dueños por día de vencimiento desde el corte (el máximo, < 80)
db.bondBound.aggregate([{$match: {isRemove: false, statusId: '1', soldDate: null, tookDate: null, maturityDate: {$gte: corte}}},
  {$group: {_id: {d: {$dateToString: {format: '%Y-%m-%d', date: '$maturityDate'}}, u: '$userId'}}},
  {$group: {_id: '$_id.d', duenos: {$sum: 1}}}, {$sort: {duenos: -1}}, {$limit: 5}]);
```
