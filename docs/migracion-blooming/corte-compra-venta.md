# Corte de compra y venta (BLM-0135)

| | |
|---|---|
| Estado | Procedimiento ensayado en local (§8). Testing y live, pendientes |
| Servicios | backend Meteor (`bondBound.*`), ms-blooming-go, front, ms-notification |
| Variables | `BOND_BOUND_TRADING` en `/opt/backend-meteor/.env`; `BLOOMING_TRADING` en el `.env.<ENV>` de ms-blooming-go; en testing, `DEBUG_MODE` y `DEBUG_EMAILS` en `/opt/ms-notification/.env` |
| Herramientas | `seedkeycounters` (BLM-0079) y `auditdata` (BLM-0133), compiladas como en la [auditoría](auditoria-datos-corte.md) §2 |
| Antes | [Auditoría de datos](auditoria-datos-corte.md), §8 pasos 1 a 4 (y 6 si ya se cortó la maduración) |
| Después | Auditoría §8 paso 8; BLM-0137 retira los métodos y la variable de Meteor; BLM-0138 monitorea dos semanas |
| Decisiones | Spec 04 (RF-04.3, RF-04.9 y §11), spec 05 (RF-05.3); [README](README.md) §6 |

## 1. Qué cambia

Hoy compran y venden los métodos DDP de Meteor: `bondBound.purchase`, `bulkPurchase`, `preSale` y `sale`. Por dentro usan `bondBound.insert` y `bondBound.setStatus`, que también se pueden llamar por DDP. Después del corte, el front llama a `POST /bond-bounds/purchase`, `/bulk-purchase` y `/{id}/sale` de ms-blooming-go, y cotiza con `GET /bond-bounds/{id}/pre-sale`.

En cada entorno, el corte son cuatro pasos, siempre en este orden:
1. **Meteor deja de comprar y vender:** `BOND_BOUND_TRADING=false` y un reinicio. Los cuatro métodos públicos responden el código `-2040`, con el mensaje "La compra y la venta de bonos no están disponibles en este momento…". `insert` y `setStatus` lanzan ese mismo error.
2. **Siembra de los contadores de clave,** cuando ninguna réplica de Meteor compra.
3. **ms-blooming-go empieza a comprar y vender:** `BLOOMING_TRADING=true` y un reinicio. Hasta entonces, esas tres rutas responden `503 ERR_TRADING_DISABLED`.
4. **Deploy del front** que compra y vende por REST.

- **Las dos compras nunca conviven.** Go numera con su contador y Meteor con `count + 1`. Si compra Go y después Meteor, la siguiente clave de Go repite la de Meteor. Si compra primero Meteor, Go repite la de Meteor.
- **Las dos ventas tampoco.** La venta de Meteor no tiene transacción ni repite la condición de estado: un bono que Go acaba de vender se puede cobrar otra vez en Meteor.
- **La siembra va después del paso 1, no antes.** Una compra de Meteor entre la siembra y su apagado deja el contador un número atrás, y la primera clave de Go repite la suya. La siembra no baja nunca un contador (`$max`), así que se puede correr en seco antes y repetir sin riesgo.
- **El hueco entre los pasos 1 y 4 deja la compra y la venta sin servicio unos minutos.** Mientras tanto los usuarios ven el mensaje de Meteor. La cotización de Go sigue respondiendo durante el hueco.
- **Todo es configuración, salvo el front.** El código de las dos variables ya está desplegado (prerrequisitos 1 y 2) y los `.env` se leen al arrancar. El rollback es volver las variables atrás y el front a su release anterior.

## 2. Prerrequisitos

En cada entorno, antes de la ventana:

| # | Qué | Cómo se verifica |
|---|---|---|
| 1 | Backend con la variable `BOND_BOUND_TRADING` (el commit de BLM-0135 en `server/model/bond.bound.js` y `server/lib/`), sin valor | La imagen de `BACKEND_futurapps` es de un release que incluye ese commit; chequeo en seco 1 |
| 2 | ms-blooming-go con BLM-0166, y `BLOOMING_TRADING` ausente o `false` | Su log de arranque tiene `[bondBound] compra y venta deshabilitadas: BLOOMING_TRADING no es true`; chequeo en seco 3 |
| 3 | Release del front que compra, compra en masa, cotiza y vende por REST (la parte "compra y venta" de BLM-0136), probada contra el ms-blooming-go de ese entorno | Release identificada, con su deploy y su vuelta atrás. En el orden de BLM-0136, el catálogo, las preferencias y el historial ya van a Go, así que la URL, el `x-secret` y `CORS_ORIGINS` ya están probados |
| 4 | Mongo con replica set: la compra y la venta usan transacciones | `db.hello().setName` no vacío |
| 5 | Auditoría §8, pasos 1 a 4, y el 6 si la maduración ya se cortó | Informe de la auditoría |
| 6 | Consulta K2 vacía: ninguna `key` de producto está en dos tipos, porque los dos tipos numerarían las mismas claves | Consulta K. Si no sale vacía, se cambia la `key` de uno de los productos antes de la ventana. K1 se corrige en el paso 4 |
| 7 | `stop_grace_period` de ms-blooming-go de al menos 30 s: los avisos y los emails de la compra y la venta salen después de responder | Como el prerrequisito 9 de [corte de la maduración](corte-maduracion.md) |
| 8 | **Solo testing:** el ms-notification al que apunta `MS_NOTIFICATION_URL` de ms-blooming-go corre con `DEBUG_MODE=true` y `DEBUG_EMAILS=<buzón del equipo>` | Como el prerrequisito 10 de [corte de la maduración](corte-maduracion.md). La base de testing es un clon de producción con datos personales reales: sin esto, la compra y la venta de prueba les mandan el email a esas personas |
| 9 | Perfil del equipo con una cuenta activa con saldo y un producto activo para comprar. En testing, además, un bono suyo que se pueda vender | Para el humo del paso 7. En live, la compra solo se hace si el negocio lo aprueba: el monto se descuenta de la nómina |

Además:
- **Ventana.** Con poco tráfico, y avisada a soporte: la compra y la venta quedan sin servicio entre los pasos 2 y 6. Durante la ventana no se despliegan ni ms-blooming-go ni el backend.
- **Comandos.** `<servidor>` es el `DEPLOY_SERVER` del `docker-stack-deploy.sh` de cada servicio, que expone la API de Docker en el puerto `12375`: `docker -H <servidor>:12375 …`. Las consultas del apéndice son de solo lectura y usan el reloj del servidor de Mongo.
- **Datos sensibles.** `seedkeycounters` y `auditdata` imprimen ids opacos de bonos. Ningún log se pega crudo, y §7 lleva solo números y claves.

## 3. Chequeos en seco

Se hacen el día del corte, antes del paso 2.

1. **Meteor lee la variable vacía.** En cada nodo, `sh -c 'export $(grep -v "^#" /opt/backend-meteor/.env | xargs) && printenv BOND_BOUND_TRADING'` no imprime nada, o imprime algo distinto de `false`.
2. **`seedkeycounters` apunta a la base del entorno.** Desde un directorio con el `.env` y el `.env.<ENV>` efectivos de ms-blooming-go, `./seedkeycounters` en seco. La herramienta no imprime el nombre de la base: la cantidad de bonos de cada tipo tiene que ser la misma que da la tercera consulta de K.
3. **Go apagado responde 503 sin escribir nada.** Con el token del perfil del equipo y el `x-secret` del front, `POST <URL de ms-blooming-go>/bond-bounds/purchase` con body `{}` responde `503 ERR_TRADING_DISABLED`.

## 4. Procedimiento

Primero en testing. En live, recién con testing cerrado (§7).

1. **Línea base.**
   - `auditdata` sin `-apply`: se anota lo que cuenta el resolver de doble cobro, que es la línea base de BLM-0138, y se confirma que no aparecieron bloqueantes nuevos.
   - Consulta K y `seedkeycounters` en seco: se anota el contador que le corresponde a cada tipo.
2. **Apagar la compra y la venta de Meteor (T2).**
   1. `BOND_BOUND_TRADING=false` en `/opt/backend-meteor/.env` de **todos** los nodos: una sola vez, en su propia línea, sin espacios alrededor del `=`, sin comentario en la misma línea y sin comillas tipográficas. Las mismas reglas de `MS_FEED_MATURITY` en el [corte de la maduración](corte-maduracion.md) §4 paso 2.
   2. En cada nodo, el comando del chequeo en seco 1 imprime `false`, y `sha256sum /opt/backend-meteor/.env` da lo mismo en todos.
   3. `docker -H <servidor>:12375 service update --force BACKEND_futurapps`, sin `-d`. Reinicia las réplicas de a una: hasta que cae la última réplica vieja, esa réplica sigue comprando y vendiendo.
3. **Portón: Meteor ya no compra ni vende.** Tiene que cumplirse todo antes de seguir.
   - `docker -H <servidor>:12375 service inspect BACKEND_futurapps --format '{{.UpdateStatus.State}}'` da `completed`. Se anota esa hora como T3. Con `rollback_completed` o `paused`, alguna réplica sigue vendiendo: no se sigue.
   - `docker -H <servidor>:12375 service ps BACKEND_futurapps --filter desired-state=running` muestra solo tareas creadas después de T2. Si una tarea nueva no arranca (`Failed` o `Rejected`), el `.env` está mal formado: se corrige en todos los nodos, se valida como en 2.2 y se repite el paso 2.
   - `docker -H <servidor>:12375 service logs --since <T2> BACKEND_futurapps 2>&1 | grep -c 'bondBound: compra y venta deshabilitadas con BOND_BOUND_TRADING=false'` da al menos el número de réplicas.
   - La consulta M con T3 da 0 dos minutos después de T3.
4. **Sembrar los contadores.**
   1. `./seedkeycounters -apply`, desde el mismo directorio del chequeo en seco 2. Se anota la línea `Se crearon N contadores y se subieron M.`
   2. Otra vez en seco: todos los tipos dicen `se mantiene`. Si alguno dice `contador X → Y`, Meteor compró después de 4.1: se repite el portón y la siembra.
   3. Si K1 no salió vacía, se sube a mano el contador de cada tipo que aparece en ella hasta su mayor sufijo: `db.bondKeyCounter.updateOne({_id: '<bondTypeId>'}, {$max: {seq: <mayorSufijo>}})`. La numeración salta, pero no se repite ninguna clave.
   4. En testing, la siembra puede informar claves repetidas con `_id` UUID. Son compras que hizo Go antes de BLM-0166, con el contador sin sembrar. Se listan para revisarlas y no se renumeran.
5. **Encender la compra y la venta de Go (T5).**
   1. `BLOOMING_TRADING=true` en el `.env.<ENV>` de ms-blooming-go, que pisa al `.env`, en **todos** los nodos. `sha256sum` da lo mismo en todos.
   2. `docker -H <servidor>:12375 service update --force <servicio de ms-blooming-go>`, sin `-d`, hasta `completed`. Se anota la hora de inicio como T5.
   3. En `service logs --since <T5>` ya no aparece `[bondBound] compra y venta deshabilitadas`. El `POST` del chequeo en seco 3, con body `{}`, ahora responde `400 ERR_ACCOUNT_ID_REQUIRED` en lugar de `503`, sin escribir nada.
6. **Desplegar el front** con la release del prerrequisito 3.
7. **Humo, con el perfil del equipo.**
   - Una compra del producto de prueba responde `201`. La `key` es `<key del producto>-<contador del tipo después del paso 4, más 1>`. En Mongo quedan:
     - el bono, con `keySeq`;
     - un `balanceDetail` por `-amount`;
     - dos `bondHistory` (`BOND_BOUND_SOLD` y `BOND_SOLD`);
     - dos `notice`.

     El email llega: en testing, al buzón del equipo.
   - En testing, la venta de un bono vendible del perfil responde `200`, y repetirla responde `409 ERR_BOND_ALREADY_SOLD`.
   - En una pestaña con el front anterior, abierta antes del paso 6, la compra muestra el mensaje de Meteor. En el log del backend aparece `bondBound.purchase: deshabilitado con BOND_BOUND_TRADING=false`.
   - En live, si el negocio no aprueba la compra de prueba, las consultas V se corren sobre las primeras compras reales.
8. **Numéricos.** `auditdata -apply -confirm <ENV>/<base> -only numericos -pausa 10ms` (auditoría §8 paso 7.3). Meteor ya no crea bonos, y Go los guarda con números.
9. **Verificación.** Consultas V con T5: V1, V2 y V3 dan 0.
10. **Cierre.** Se completa §7 y se avisa a soporte.

## 5. Seguimiento

Hasta el cierre de BLM-0138, con alertas o con una revisión diaria:
- `bond-bounds/(purchase|bulk-purchase|<id>/sale) 503` en ms-blooming-go: tiene que ser 0 desde T5. Si aparece, una réplica arrancó sin la variable (un nodo con otro `.env`): se corrige y se hace `--force`.
- `bondBound.<método>: deshabilitado con BOND_BOUND_TRADING=false` en el backend son clientes con el front anterior: tienen que desaparecer en horas. Si siguen días después, otro cliente llama por DDP y hay que identificarlo.
- La consulta M con T3 mayor que 0: una réplica de Meteor volvió a vender. Se corrige el `.env` de ese nodo, se hace `--force` del backend y se corre V2.
- V2 una vez por día.
- `B_BondBound_Audit_doubleProcess` contra la línea base del paso 1: no puede crecer.
- 5xx y `[handler] error interno` en estas rutas, y `[NotificationPort]` en ms-blooming-go: son emails perdidos.
- `422 ERR_USER_SETTING_ACCOUNT_REQUIRED` es esperado: muchos settings no tienen cuenta seleccionada (341 de 522 en el dump, auditoría §9). El front pide elegir una.

## 6. Rollback

Primero se apaga Go, después vuelve el front anterior y por último se reactiva Meteor, nunca al revés:
1. `BLOOMING_TRADING=false` en **todos** los nodos, con `sha256sum` igual en todos, y `--force` de ms-blooming-go hasta `completed`. El log de arranque vuelve a aparecer, y el `POST` del chequeo en seco 3 vuelve a dar `503`.
2. El front vuelve a su release anterior.
3. `BOND_BOUND_TRADING` vacía en todos los nodos y `--force` de `BACKEND_futurapps`. Con el update en `completed`, ninguna réplica loguea `compra y venta deshabilitadas` desde el rollback.

No hay datos que revertir. Los bonos que compró Go son válidos: tienen `_id` UUID, `keySeq`, historial y su movimiento de saldo. El `count + 1` de Meteor sigue por encima de sus claves, porque los cuenta. Para volver a cortar se repite el procedimiento desde el paso 1: la siembra sube los contadores por encima de lo que vendió Meteor mientras tanto.

Mientras BLM-0137 no retire los métodos, ninguna réplica del backend puede arrancar sin `BOND_BOUND_TRADING=false` después del corte. Lo mismo vale para una imagen anterior a BLM-0135, un `.env` restaurado de un respaldo previo o un nodo nuevo con un `.env` copiado sin esa línea. Vacía o ausente, la variable reactiva la compra de Meteor con Go activo.

| Resultado | Qué significa | Qué hacer |
|---|---|---|
| `503 ERR_TRADING_DISABLED` después de T5 | Una réplica de Go sin la variable | Corregir el `.env` de ese nodo y hacer `--force` |
| La segunda siembra en seco dice `contador X → Y` | Meteor compró después de la siembra | Repetir el portón y la siembra, y correr V2 |
| V2 > 0 | Claves repetidas | Apagar Go (paso 1 del rollback), listar las claves y ver si vendió Meteor (consulta M) o si un contador quedó bajo (V3). No se renumeran solas |
| V3 > 0 | Un contador quedó por debajo de un `keySeq` | Apagar Go y subir ese contador con `$max` al mayor `keySeq` |
| `404`, `400` o `429` de ms-notification, o `Notification errors` | Emails de compra o venta perdidos | Corregir ms-notification; la compra y la venta ya quedaron hechas |

## 7. Resultados

| Entorno | Fecha | Siembra (creados / subidos) | Primera compra | V1 / V2 / V3 | Observaciones |
|---|---|---|---|---|---|
| Local (ensayo, §8) | 2026-10-30 | 3 / 0 | `B003-2278` (contador 2277) | 0 / 0 / 0 | Emails desviados a un destino sin servicio |
| Testing | | | | | |
| Live | | | | | |

## 8. Ensayo local

El 2026-10-30 se ensayó el procedimiento contra `corte_ensayo`, una copia con `$out` de las colecciones de bonos, saldo, `profile`, `notice`, `agency` y `country`, tomadas de `futurapps`. Se usaron los binarios de ms-blooming-go con BLM-0166 y BLM-0167, sobre `1f8c242`.
- ms-blooming-go corrió con un `API_SECRET` aleatorio, con ms-notification y ms-exporter en `127.0.0.1:9` (ningún email salió) y con un ms-oauth falso que responde la sesión de un dueño de bonos del dump.
- En la copia se creó una cuenta de prueba con saldo para ese dueño, seleccionada en su setting, y se dejó activo uno de sus productos.
- Meteor no se levantó: el guard se probó con un arnés de stubs que carga `bond.bound.js` con `_constants.js`, los `Result` de `_variables.js` y `text.js`.

| Paso | Resultado |
|---|---|
| Meteor, arnés de stubs | Con `false`, `FALSE` y ` False `: los cuatro métodos responden `-2040` sin tocar ninguna colección, `insert` y `setStatus` lanzan `-2040`, y aparece el log de arranque. Con `true`, vacía, ausente o `no`, los seis llegan al modelo. Contra el backend sin BLM-0135, el arnés falla |
| Go con `BLOOMING_TRADING` ausente | Log de arranque. `purchase`, `bulk-purchase` (también con body `{}`) y `sale` responden `503 ERR_TRADING_DISABLED`; `pre-sale`, `200`. Ninguna escritura y ningún contador |
| K | K1 y K2 vacías: 3 tipos, con el mayor sufijo igual a la cantidad de bonos |
| Siembra | En seco: 1 clave repetida (la de la auditoría §5.1) y 3 tipos sin contador. Con `-apply`: 3 creados y 0 subidos. Otra vez en seco: los 3 `se mantiene` |
| Go con `true` | Sin el log de deshabilitado. Compra: `201`, `B003-2278`. Compra masiva de 3: `B003-2279` a `B003-2281`. Venta de un `'1'` legacy: `200`, pasa a `'5'`. La segunda venta: `409 ERR_BOND_ALREADY_SOLD`. Los 5 emails fallaron con `connection refused`, como se buscaba |
| V | V1 = 0 (4 bonos nuevos, todos con `keySeq`), V2 = 0 y V3 = 0. Contador del tipo en 2281. Quedaron 5 `balanceDetail`, 9 `bondHistory` y 5 `notice` nuevos |
| `auditdata -only numericos` | 2288 aplicados y 0 fallidos. Queda el bloqueante del dump (auditoría §5.7) |

Al terminar se borró `corte_ensayo`, y `futurapps` y `parity` mantuvieron su huella (`dbHash`).

## Apéndice: consultas

Son de solo lectura y van en una sola sesión de mongosh sobre la base del entorno.

```js
var meteorId = /^[0-9A-Za-z]{17}$/;   // _id de Meteor; Go usa UUID

// K. Claves futuras, antes del corte
// K1: tipos cuyo mayor sufijo numérico supera su cantidad de bonos, que es lo que siembra seedkeycounters
var sufijo = {$convert: {input: {$arrayElemAt: [{$split: [{$cond: [{$eq: [{$type: '$key'}, 'string']}, '$key', '']}, '-']}, -1]},
  to: 'int', onError: null, onNull: null}};
db.bondBound.aggregate([{$group: {_id: '$bondTypeId', bonos: {$sum: 1}, mayorSufijo: {$max: sufijo}}},
  {$match: {$expr: {$gt: ['$mayorSufijo', '$bonos']}}}], {allowDiskUse: true});   // vacía; si no, paso 4.3
// K2: keys de producto en más de un tipo
db.bondObject.aggregate([{$match: {key: {$type: 'string'}}}, {$group: {_id: '$key', tipos: {$addToSet: '$bondTypeId'}}},
  {$match: {'tipos.1': {$exists: true}}}]);   // vacía (prerrequisito 6)
// Bonos por tipo: los mismos números que imprime seedkeycounters (chequeo en seco 2)
db.bondBound.aggregate([{$group: {_id: '$bondTypeId', bonos: {$sum: 1}}}, {$sort: {_id: 1}}]);

// M. Meteor ya no compra ni vende: 0 dos minutos después de T3
var T3 = ISODate('<hora del completed del paso 3>');
db.bondBound.countDocuments({_id: meteorId, createdAt: {$gte: T3}});
db.bondHistory.countDocuments({_id: meteorId, createdAt: {$gte: T3}, text: {$in: ['BOND_BOUND_SOLD', 'BOND_BOUND_STATUS_CHANGED']}});

// V. Después de encender Go
var T5 = ISODate('<inicio del paso 5>');
var nuevos = {createdAt: {$gte: T5}};
// V1: nada de Meteor desde T5, y todos los bonos nuevos con keySeq
db.bondBound.countDocuments({...nuevos, _id: meteorId});                                                  // 0
db.bondHistory.countDocuments({...nuevos, _id: meteorId, text: {$in: ['BOND_BOUND_SOLD', 'BOND_BOUND_STATUS_CHANGED']}});   // 0
db.bondBound.countDocuments({...nuevos, keySeq: {$exists: false}});                                       // 0
// V2: claves nuevas repetidas, contra todos los bonos
var claves = db.bondBound.distinct('key', nuevos);
db.bondBound.aggregate([{$match: {key: {$in: claves}}}, {$group: {_id: '$key', n: {$sum: 1}}}, {$match: {n: {$gt: 1}}}]);   // vacía
// V3: tipos cuyo contador quedó por debajo de su mayor keySeq
db.bondBound.aggregate([{$match: {keySeq: {$type: 'number'}}}, {$group: {_id: '$bondTypeId', max: {$max: '$keySeq'}}},
  {$lookup: {from: 'bondKeyCounter', localField: '_id', foreignField: '_id', as: 'c'}},
  {$match: {$expr: {$lt: [{$ifNull: [{$first: '$c.seq'}, 0]}, '$max']}}}]);   // vacía
```
