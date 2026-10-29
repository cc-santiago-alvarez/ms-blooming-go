# Auditoría de datos previa al corte (BLM-0133)

| | |
|---|---|
| Estado | Dump local auditado, con ensayo de `-apply` y `-revert`. Testing y live, pendientes |
| Herramienta | `cmd/tools/auditdata` |
| Lee | `bondType`, `bondObject`, `bondBound`, `bondHistory`, `bondUserSetting`, `profile`, `balanceAccount` |
| Escribe, solo con `-apply` | `bondObject`, `bondBound`, `bondHistory` |
| Antes de | BLM-0134 (maduración) y BLM-0135 (compra y venta). Es la línea base de BLM-0138 |
| Decisiones | Spec 02 §6 y O11, spec 04 §6, spec 08 RF-08.1, spec 10 §8 (B22) y §11 |

Este informe no tiene datos personales:
- los dueños se cuentan, nunca se identifican;
- los montos son sumas de `amount`, es decir, el capital de los bonos;
- solo aparecen ids de bonos, productos y settings.

El dump local sí tiene datos personales reales (§3).

## 1. Resumen

| Hallazgo | Acción | Dump local |
|---|---|---|
| Bonos `'2'` con `soldDate` (B22): Meteor todavía los paga al canjearlos | `-apply` los pasa a `'5'` | 290 bonos de 18 dueños, Σ 33.600.000 |
| Bonos `'6'` con `soldDate` y `tookDate`: se cobraron dos veces | Se informan a contabilidad | 244 bonos de 17 dueños, Σ 27.100.000 |
| Numéricos guardados como string | `-apply` los guarda como número | `maturityTimeInDays: "365"` en 6 productos y 2282 bonos |
| `isPrivateBond` que no es booleano (O11) | `-apply` lo guarda como booleano | 2 productos |
| Claves de bono repetidas (B8) | Se informan; no se renumeran | 1 clave |
| Settings duplicados (U3) | `dedupusersettings -apply` | 0 |
| Bonos `'1'` vencidos que madura la primera corrida de Go (B36) | Los madura BLM-0134 | 395 al 2023-07-06; 0 que ms-feed no madura |
| Documentos que Go no puede leer u operar | A mano | 1 producto, sin bonos |

- **B22 es urgente en live.** Los 290 bonos del dump vienen de producción y se maduraron antes de 2021-05-21, así que lo más probable es que existan hoy en live. Mientras Meteor atienda canjes, cada uno se puede cobrar otra vez.
- **Las normalizaciones están ensayadas.** En una copia del dump, `-apply` hizo los 2580 cambios con su historial. Una segunda corrida no encontró nada, y `-revert` dejó la copia idéntica a la original (§7).
- **Los números de live no se conocen.** Testing y live se auditan con la misma herramienta (§8) y sus resultados se agregan en §4.

## 2. Cómo se corre

```bash
CGO_ENABLED=0 go build -o auditdata ./cmd/tools/auditdata   # binario estático, desde un checkout limpio

./auditdata                                              # solo informa
./auditdata -at 2026-11-15                               # la primera corrida de la maduración, a esa fecha
./auditdata -apply -confirm <ENV>/<base>                 # normaliza numéricos, isPrivateBond y vendidos
./auditdata -apply -confirm production/futurapps -only vendidos -pausa 10ms
./auditdata -revert <respaldo>                           # muestra qué revertiría
./auditdata -revert <respaldo> -apply -confirm <ENV>/<base>
```

| Opción | Para qué |
|---|---|
| `-apply` | Aplica las normalizaciones, o la reversión con `-revert`. Sin él no escribe nada |
| `-confirm <ENV>/<base>` | Obligatorio con `-apply`: el entorno (`ENV`) y la base de la conexión, por ejemplo `testing/futurapps`. La base se llama igual en testing y en live, así que el entorno es lo que los distingue |
| `-db <base>` | Pisa `FA_MONGO_DBNAME` |
| `-at <fecha>` | Fecha de referencia de la primera corrida de la maduración (RFC 3339 o `AAAA-MM-DD`, UTC). Por defecto, ahora |
| `-only numericos,privados,vendidos` | Normalizaciones de `-apply`. Por defecto, las tres |
| `-backup <archivo>` | Respaldo de `-apply`. Por defecto `tmp/auditdata/<base>-<fecha>.jsonl` |
| `-revert <archivo>` | Revierte los cambios de un respaldo |
| `-pausa <duración>` | Pausa entre escrituras, para no saturar a los observers de Meteor |
| `-v` | Lista todos los ids. Sin él, hasta 20 por hallazgo |

**Entorno.**
- Lee `ENV`, `FA_MONGO_URL` y `FA_MONGO_DBNAME` de `.env` y `.env.<ENV>` en el directorio de trabajo, como el API.
- Al empezar muestra el entorno, la base, el replica set y el commit de la herramienta, y avisa si se compiló con cambios sin commitear.
- Nunca muestra la URL ni las credenciales. Un error de conexión o de red del driver sí puede mostrar el host y el puerto: los errores no se copian al informe.
- La salida no tiene datos personales. De perfiles y cuentas no muestra el `_id`, sino la causa (el campo y su tipo), para buscarlos dentro del clúster, por ejemplo con `db.profile.find({first_name: {$type: "int"}}, {_id: 1})`.
- Termina con una tabla Markdown para copiar en §4.

**Códigos de salida:**
- 0: no queda nada pendiente.
- 1: error.
- 2: quedan pendientes, es decir, algo que normaliza `-apply`, settings duplicados, revisión manual o bloqueantes. Las claves repetidas, los `'6'` y la maduración solo se informan y no cuentan.

**Qué hace `-apply`:**
- **Requisito.** Exige un replica set y aplica en orden: numéricos de `bondObject`, numéricos de `bondBound`, `isPrivateBond` y vendidos.
- **Guarda.** Cada documento se escribe con una guarda que repite la condición que justificó el cambio: el valor y el tipo que leyó la auditoría, y en B22 `{statusId: '2', soldDate: Date, tookDate: null}`. Si Meteor o ms-feed lo cambiaron en el medio, el documento no se toca y queda como "cambió durante la corrida", para la próxima.
- **Historial.**
  - B22: un cambio de estado de sistema `['2','5']` con `userId ""` y la agencia del bono.
  - O11: un `BOND_UPDATED` de sistema con el valor anterior. El historial va en la misma transacción que el cambio.
  - Los numéricos no registran historial: el valor no cambia.
- **`updatedAt`.** Todo cambio lo actualiza (constitución §7).
- **Respaldo.**
  - Antes de cada escritura se agrega una línea al respaldo y se baja a disco.
  - El formato es Extended JSON canónico, una línea por cambio, con los valores de antes y de después (con su tipo) y el `_id` del historial.
  - El archivo se crea con permisos `0600` y nunca pisa uno existente.
  - Va al almacenamiento del equipo, nunca a git.
- **Idempotencia.** Una segunda corrida no encuentra nada que normalizar y no crea otro respaldo.

**Revertir.**
- `-revert <respaldo>` recorre los cambios del último al primero.
- Cada documento vuelve a sus valores de antes, con su `updatedAt`, solo si sigue con los que dejó el cambio. Su historial de sistema se borra en la misma transacción.
- Lo que cambió después se omite y se informa.
- El respaldo tiene que ser del mismo entorno y la misma base.
- Si la última línea del respaldo quedó a medias (por ejemplo, con el disco lleno), se ignora: su cambio no llegó a la base.
- Sin `-apply` solo cuenta.

**Quién lo corre.**
- En testing y live lo corre DevOps.
- Se usa un binario compilado desde un checkout limpio del commit desplegado, dentro de la red del clúster y con los mismos archivos de entorno del servicio. El commit de la línea de inicio es el que se anota en §3.
- Ninguna imagen trae hoy `cmd/tools`, y las credenciales de live no se copian a otros equipos.

## 3. Entornos

El dump local no es el anonimizado que describe T-PREP.7:
- `futurapps` y `parity` son un dump completo de la base de testing, que se clonó de producción el 2022-09-07;
- las escrituras de bonos terminan el 2023-07-06;
- el PII de `profile` es real;
- `balanceDetail` tiene 13 documentos, así que en local no se puede seguir el dinero.

Los bonos de B22 y de los `'6'` son registros de producción anteriores al clon. Los `'1'` vencidos, en cambio, son un efecto de esta base congelada: testing dejó de madurar después del clon.

| Entorno | Base | Fecha (UTC) | Referencia (`-at`) | Commit | Modo | Respaldo | Responsable |
|---|---|---|---|---|---|---|---|
| Dump local | `futurapps` | 2026-10-29 | 2023-07-06 y 2026-10-29 | local, antes del commit | Solo lectura | — | Backend |
| Ensayo local | `auditdata_ensayo` (copia de `futurapps`, ya borrada) | 2026-10-29 | 2023-07-06 | local, antes del commit | `-apply` y `-revert` | Fuera del repo | Backend |
| Testing | | | | | | | DevOps |
| Live | | | | | | | DevOps |

## 4. Resultados por entorno

| Indicador | Acción | Dump local | Testing | Live |
|---|---|---|---|---|
| Claves repetidas (B8) | Informar | 1 clave, 2 bonos | | |
| Settings duplicados (U3) | `dedupusersettings -apply` | 0 | | |
| Settings vivos sin `userId` / con `userId` que no es string | Informar | 1 / 1 | | |
| Numéricos string canónicos | `-apply` (`numericos`) | 6 `bondObject`, 2282 `bondBound` | | |
| Numéricos string no canónicos | A mano | 0 | | |
| `isPrivateBond` no booleano (O11) | `-apply` (`privados`) | 2 (0 manuales) | | |
| `'2'` con `soldDate` (B22) | `-apply` (`vendidos`) | 290 (0 manuales) | | |
| `'6'` con `soldDate` y `tookDate` | Informar a contabilidad | 244 (la auditoría del gateway cuenta 244, de 17 dueños) | | |
| Primera corrida de Go (B36) | Informar | 395 bonos de 11 dueños al 2023-07-06; 497 de 12 al 2026-10-29; 0 que ms-feed no madura | | |
| Bloqueantes de Go | A mano | 1 | | |

Después de `-apply` (ensayo), los numéricos, O11 y B22 quedan en 0, y solo queda el bloqueante.

## 5. Detalle

### 5.1 Claves de bono repetidas (B8)

- La clave `B001-1731` la tienen los bonos `PSKMHAAhWyaaZiCds` y `JtkiWycF3RMvoK5kJ`, los dos en `'6'`.
- **Causa: bajas físicas manuales.** Entre el 2021-09-01 y el 2021-09-02 se borraron bonos a mano: el `count` del tipo bajó y Meteor repitió los números 1729 a 1731. Faltan los sufijos 997, 998 y 1007, y `bondHistory` tiene registros de 4 bonos que ya no existen.
- **No se renumeran:** el usuario ve las claves, y los avisos y el historial las citan.
- La siembra de contadores (BLM-0079) usa el `count` de cada tipo. En el dump, ese `count` es igual al mayor sufijo de cada tipo, así que las claves nuevas de Go no repiten ninguna (§9).

### 5.2 Settings (U3)

- **Sin duplicados.** Ningún usuario tiene más de un setting vivo, y el índice único parcial `userId_1` ya existe en las dos bases locales (lo creó Go).
- **Settings sin `userId` usable.** El setting `ErWs6eeCeNfYg7KPb` tiene `userId: null`, y `meLMmXdPjxDhQP7XN` tiene un `userId` ObjectId que no tiene `profile`. Go no los encuentra desde ninguna sesión. Se informan y no se tocan.
- **U3 en live.** Hay que medirlo allí. Si aparecen duplicados, se corrige con `dedupusersettings -apply` (spec 09 §5).

### 5.3 Numéricos guardados como string

| Colección | Campo | Tipos en el dump |
|---|---|---|
| `bondObject` | `maturityTimeInDays` | string 6 (`"365"`), int 5 |
| `bondBound` | `maturityTimeInDays` | string 2282 (`"365"`), int 3 |
| `bondObject` | `maturityRewardsPercentage` | int 10, double 1 |
| `bondBound` | `earlyCashTerm.initialDayForWithdraw` | int 2207, double 78 (`0.0`) |
| Ambas | `amount`, el resto de `earlyCashTerm` | int en todos |

- **Por qué se normalizan.** El legado nunca parseó estos campos: guarda lo que manda el formulario, y la compra copia los del producto en el bono. Go los lee con `FlexNumber`.
- **Qué toca `-apply`.** Solo los strings canónicos (`^(0|[1-9][0-9]*)(\.[0-9]*[1-9])?$`, y enteros en `maturityTimeInDays`, `inflectionPoint` e `initialDayForWithdraw`). Los guarda como int32 si son enteros y entran en int32; si no, como double.
- **Qué no toca.** `""`, `" 365"`, `"365.0"` o `"1,5"` se informan para revisión manual. Canónico es el formato que Go, `Number`, `parseFloat` y `parseInt` leen igual. Fuera de él, algunos strings se leen distinto: `""` es 0 para Go y `NaN` para `parseInt`; `"1,5"` falla en Go y es 1 para `parseInt`; `"1e3"` es 1000 para Go y 1 para `parseInt`.
- **Efecto en Meteor.** Pasar `"365"` a `365` no cambia ningún cálculo de Meteor: todas sus lecturas usan `parseInt` o coerción aritmética.
- **Efecto en los clientes.** Cambia el tipo que reciben el front por DDP y la celda del Excel de ms-exporter.
- **`amount` como string.** No hay ninguno en el dump. Si aparece en live, es prioritario: Meteor lo concatena en la venta (`"1500" + x`) y `$sum` lo ignora.

### 5.4 `isPrivateBond` que no es booleano (O11)

| Producto | Valor guardado | Queda |
|---|---|---|
| `WizkTxQg8LPFpnp58` | string de 17 caracteres, igual a su `bondTypeId` | `true` |
| `dcte7RdtHCxsn9oKe` | string de 17 caracteres, igual a su `bondTypeId` | `true` |

- **Origen.** Son productos de prueba creados en testing en enero y febrero de 2023, mientras se desarrollaba el campo; el primero se creó un día antes del commit de Meteor que lo agrega (`364f90f`). En los dos el valor es el `bondTypeId` del propio producto, y Meteor guarda lo que recibe (`bond.object.js:367, 427`), así que lo más probable es que lo enviara el formulario de administración en desarrollo. No se puede confirmar: el front no está en los repos y el historial de esa época no guarda los cambios.
- **Por qué `true`.** Meteor (`bond.object.js:572-576`, también para `numOfBonds`) y Go ya los tratan como privados, así que pasarlos a `true` no cambia lo que ve nadie.
- **Revisión de negocio.** Hay que decidir si deberían ser públicos. `dcte7RdtHCxsn9oKe` no tiene autorizados, así que hoy ningún inversionista lo ve.
- **Productos sin el campo.** 6 productos no tienen `isPrivateBond`: Go y Meteor los leen como públicos, y no se tocan.
- **Mientras Meteor administre el catálogo.** Su alta y su edición guardan cualquier string no vacío, así que la auditoría se repite al final (§8).

### 5.5 Bonos madurados que ya estaban vendidos (B22)

| Año de `soldDate` | Bonos | Σ `amount` |
|---|---|---|
| 2018 | 27 | 5.500.000 |
| 2019 | 61 | 6.100.000 |
| 2020 | 183 | 20.100.000 |
| 2021 | 19 | 1.900.000 |
| **Total** | **290** (18 dueños, 0 borrados) | **33.600.000** |

- **Origen.** Entre `87b4593` (2018-01-09) y `a33478e` (2021-05-21), ms-feed maduraba también los bonos vendidos (`'5'`). Todos estos se vendieron antes de su vencimiento y vencieron antes del 2021-05-21.
- **El historial confirma la venta.** 289 tienen un único cambio de estado `['1','5']` del dueño, y uno tiene además dos `['5','5']`. La maduración de ms-feed no dejaba rastro.
- **Riesgo.** El canje de Meteor (`bond.bound.js:730-735, 816-824`, `origin/main`) solo bloquea un `'5'` con `soldDate` o un `'6'`, así que un `'2'` con `soldDate` se vuelve a pagar. Go ya lo rechaza (`409 ERR_BOND_ALREADY_SOLD`).
- **Cuándo `-apply` lo pasa a `'5'`.** Solo si el historial registra la venta y el bono tiene agencia (la venta de Meteor escribe ese registro después de acreditar). Sin esa evidencia, el bono pudo quedar sin cobrar y va a revisión manual. En el dump fueron 0.
- **Efecto visible** (requiere el visto bueno de negocio y soporte):
  - los bonos pasan de "Madurado" a "Vendido" y salen de los listados de bonos activos (`'1'`/`'2'`);
  - dejan de contar en el monto madurado del resumen del usuario (spec 09) y en el KPI por estado;
  - no se mueve saldo;
  - el historial muestra un cambio de estado del sistema, que `B_BondBound_Audit_details` no cuenta, porque descarta los registros sin perfil.

### 5.6 Primera corrida de la maduración de Go (B36)

- **Qué cuenta.** Los bonos con el filtro de la maduración de Go (`{isRemove: false, statusId: '1', maturityDate: {$lt: corte}, soldDate: null, tookDate: null}`, con el corte al inicio del día siguiente a `-at` en UTC).
- **Qué no maduraría ms-feed.** Los que no tienen `profile`, `bondType` o `bondObject`: sus `$lookup` no filtran `isRemove`, así que un producto borrado sigue cruzando.
- **Dump.** 395 bonos de 11 dueños al 2023-07-06, y 497 de 12 al 2026-10-29. Ninguno falta en los cruces de ms-feed, y Go los decodifica todos.
- **Live.** Como ms-feed madura cada 30 s, en live esta cifra debería ser casi solo B36. Ningún flujo de las apps borra en duro ni reescribe referencias, así que los B36 de live solo pueden venir de ediciones manuales. Antes de BLM-0134 se corre con `-at` igual a la fecha del corte.

### 5.7 Bloqueantes de Go

| Bloqueante | Efecto en Go | Dump | Cómo se resuelve |
|---|---|---|---|
| No decodifica con los tipos de Go: las 5 colecciones de bonos, las cuentas, y los perfiles como los leen Go por id y los cruces con el dueño (también borrados) | El listado que lo contiene responde 500. Un `'1'` vencido corta cada corrida de la maduración en su lote: no maduran ni ese lote ni los siguientes, y con los volúmenes normales (menos de 500 vencidos) no madura ningún bono | 0 | A mano, según el campo |
| `isRemove` no es booleano | El documento es invisible: Go siempre filtra `isRemove: false` | 0 | A mano |
| Fecha que no es Date | Una fecha nula se lee como tiempo cero, y un string queda fuera de los filtros por fecha | 0 | A mano |
| `statusId` desconocido | El bono no se puede vender ni canjear | 0 | A mano |
| `'1'` con `soldDate` o `tookDate` | Go nunca lo madura ni lo vende | 0 | A mano; es una fuente de B22 |
| `'1'` sin `agencyId` | El historial de sistema falla, así que nunca madura | 0 | A mano |
| Plazo menor o igual al punto de inflexión | La venta y el canje responden `422 ERR_BOND_INVALID_TERMS`; editar el producto en Go responde 400 | 1: producto `dcte7RdtHCxsn9oKe` (365/365, activo, sin bonos) | Corregir los términos o desactivarlo antes de que alguien lo compre |
| Saldo (`available` o `global`) que no es número | Go mueve el saldo con `$inc`, que falla: la compra, la venta y el canje con esa cuenta responden 500. Meteor lo recalculaba en JavaScript | 0 (3 cuentas en el dump) | A mano |

## 6. Doble cobro (`'6'` con `soldDate` y `tookDate`)

| Año de `tookDate` | Bonos | Σ `amount` |
|---|---|---|
| 2019 | 17 | 1.700.000 |
| 2020 | 142 | 16.900.000 |
| 2021 | 85 | 8.500.000 |
| **Total** | **244** (17 dueños) | **27.100.000** |

- **Qué son.** Bonos vendidos que la regla vieja de ms-feed pasó a `'2'` y que después se canjearon.
- **Relación con la auditoría del gateway.** En el dump coinciden con lo que cuenta `B_BondBound_Audit_doubleProcess` (244 bonos de 17 dueños), pero los criterios no son iguales: el resolver no mira `statusId`, usa `$exists`, excluye los borrados y descarta a los dueños sin perfil. Por eso la herramienta informa también lo que cuenta el resolver, con su mismo filtro.
- **Cobros de más.** Según el historial, 235 se cobraron dos veces y 9 entre tres y doce veces. La herramienta no los modifica: `soldDate` y `tookDate` son lo que usa la auditoría del gateway.
- **En live.** Se cruzan con `balanceDetail` para la conciliación contable. La línea base de BLM-0138 es lo que cuenta el resolver, medido justo antes de BLM-0135: no debe crecer.

## 7. Ensayo de `-apply` y `-revert`

Se copiaron las 7 colecciones de `futurapps` a `auditdata_ensayo`, con `$out` y la misma huella (`dbHash`), y se corrió:

| Paso | Resultado |
|---|---|
| `-apply -confirm auditdata_ensayo` | 2288 numéricos, 2 `isPrivateBond` y 290 vendidos: 2580 cambios, sin fallas. 292 registros de sistema en `bondHistory` (290 `['2','5']` y 2 `BOND_UPDATED`) |
| Auditoría después de aplicar | Numéricos, O11 y B22 en 0. Queda el bloqueante de §5.7 (sale con 2) |
| Segunda corrida de `-apply` | 0 cambios, sin respaldo nuevo |
| `-revert` sin `-apply` | Se revertirían 2580, sin omitidos |
| `-revert -apply` | 2580 revertidos. Las 7 colecciones quedaron con la misma huella que `futurapps`, incluidos los 290 bonos con dos cambios en la misma corrida (numérico y vendido), que se revirtieron en cadena. En el dump ningún producto tuvo dos cambios: ese caso lo cubre el test de integración `TestRevert` |

La copia se borró al terminar. `futurapps` y `parity` no se modificaron: siguen con la misma huella entre sí.

## 8. Orden de ejecución

1. **Ahora, en testing y live: solo lectura.**
   - Se corre la auditoría y se completa §4.
   - Negocio y soporte revisan el efecto visible de B22 (§5.5) y los productos de O11 (§5.4).
   - Contabilidad recibe los `'6'` (§6; la lista de ids sale con `-v`).
2. **Testing: ensayo de B22.**
   1. `-apply -confirm testing/futurapps -only vendidos`.
   2. Se verifica que el canje de Meteor de un bono normalizado responde que ya se vendió, y que Go sigue respondiendo 409.
3. **Live, cuanto antes: B22.**
   1. Snapshot de DevOps.
   2. `-apply -confirm production/futurapps -only vendidos -pausa 10ms`.
   3. Se guarda el respaldo en el almacenamiento del equipo.
   4. Se vuelve a auditar: B22 tiene que quedar en 0.
4. **Settings (U3), antes del primer arranque de Go en cada entorno.**
   - `dedupusersettings` y, si hay duplicados, `-apply`.
   - Si Go ya arrancó, hay que reiniciarlo y verificar con `getIndexes()` que existe `userId_1`: `EnsureIndexes` solo deja un warning.
5. **Antes de BLM-0134.** Auditoría con `-at` igual a la fecha del corte:
   - 0 bloqueantes entre los `'1'`;
   - revisar los B36 y el volumen de la primera corrida (bonos y dueños, con un aviso y un email por dueño).
6. **Después de BLM-0134.** Otra vez `-apply -only vendidos`: el `updateMany` de ms-feed no repite la condición de estado y pudo dejar un `'2'` con `soldDate` antes de apagarse.
7. **En la ventana de BLM-0135.**
   1. Auditoría.
   2. `seedkeycounters -apply`.
   3. Con la compra de Meteor ya apagada, `-apply -only numericos -pausa 10ms`.
8. **Después de BLM-0136 y BLM-0137,** con el catálogo fuera de Meteor: `-apply` completo y auditoría final. Tiene que salir con 0, o solo con bloqueantes ya revisados.
9. **BLM-0138.** `B_BondBound_Audit_doubleProcess` se compara contra la línea base de §6, es decir, contra lo que contaba el resolver antes de BLM-0135.

## 9. Otros hallazgos

Fuera del alcance de la herramienta. Se dejan para decidir aparte.

- **Choque de claves futuras.**
  - La siembra (BLM-0079) usa el `count` de cada tipo. Si en live el mayor sufijo numérico de un tipo supera su `count` (bajas físicas como las de §5.1), las claves nuevas de Go repetirían claves existentes. Lo mismo pasa si dos tipos tienen productos con la misma `key`.
  - En el dump no pasa. En live conviene medirlo antes de BLM-0135.
- **Settings en Meteor.**
  - Meteor lee y escribe el setting con `findOne({userId})`, sin `isRemove` ni orden (`bond.userSetting.js:38`). Si `dedupusersettings -apply` conserva un documento distinto del que lee Meteor, los dos sistemas usan settings distintos hasta BLM-0137.
  - Además, 341 de los 522 settings del dump no tienen `selectedAccountId`: en Go, su venta o canje pide elegir una cuenta.
- **Zona horaria de ms-feed.**
  - Entre 2020-06 y 2022-09, los avisos "Bono Madurado" salen a las 04:00 y 05:00 UTC, siguiendo el horario de verano de EE. UU. ms-feed corría en hora de Nueva York, no en UTC como supone la spec 10.
  - Desde su despliegue en Docker (2024) probablemente corre en UTC. Hay que confirmarlo en live agrupando esos avisos por hora antes de BLM-0134.
- **`FindMaturable` contradice RF-10.6.** Decodifica cada lote con `cursor.All`, y `RunMaturity` corta la corrida si no puede leer un lote. Un solo `'1'` que no decodifica deja sin madurar su lote y los siguientes en cada corrida, y con los volúmenes normales frena toda la maduración. Propuesta: tarea nueva (BLM-0164) para decodificar bono por bono, antes de BLM-0134. Mientras tanto, la auditoría lo detecta (§5.7), pero Meteor sigue creando bonos hasta BLM-0135.
- **Specs desactualizadas.**
  - El pseudocódigo de `numOfBonds` en la spec 01 usa `isPrivateBond: {$ne: true}`, pero el código usa el criterio de la spec 02 §6.
  - La spec 07 dice "Σ amount": el KPI y el resumen de reportes suman con `$convert`, y el resumen del usuario y el legado, con `$sum`, que ignora los strings.
- **`matured` nunca se guardó.** ms-feed nunca persistió `matured`: su esquema de Mongoose no lo declara. Ningún bono del dump lo tiene.
- **El dump no está anonimizado.** T-PREP.7 dice "Dump anonimizado", pero el dump local tiene datos personales reales.
- **Pregunta:** ¿quién conserva los respaldos de live y por cuánto tiempo?
- **Pregunta:** ¿se mantienen privados los dos productos de §5.4?
