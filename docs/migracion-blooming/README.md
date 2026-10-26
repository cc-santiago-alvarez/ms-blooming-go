# Migración de Blooming (bonos) a ms-blooming-go

Documentación SDD (Spec-Driven Development) para migrar el módulo de bonos del backend Meteor (`backend/server/model/bond.*.js`) al microservicio Go `ms-blooming-go`.

| Documento | Para qué |
|---|---|
| [`../constitution.md`](../constitution.md) | Pautas no negociables de código del servicio |
| [`specs/`](specs/) | Una spec por funcionalidad (`NN-<nombre>.spec.md`): requisitos (`RF`), contrato API, reglas, bugs legacy, criterios de aceptación (`CA`) y tareas |
| [`tasks.md`](tasks.md) | Todas las tareas por fase, con la matriz de trazabilidad RF → tareas → tests. Lo genera [`tools/gen_tasks.py`](tools/gen_tasks.py) |

## 1. Contexto

- Hoy Blooming vive en el backend Meteor como **50 métodos DDP y 10 publicaciones** repartidos en 5 archivos. Tiene colecciones sin esquema ni índices, ningún control de permisos y respuestas heterogéneas (valores crudos, `Response.ok/error`, excepciones).
- El front lo consume por DDP.
- El gateway GraphQL (`ms-gateway-gql`) no lo llama: solo lee `bondBound` y `bondHistory` directo en Mongo para auditorías (`resolvers/B_BoundBound.js`).

**Objetivo:** reimplementar esas operaciones en `ms-blooming-go` con arquitectura hexagonal (referencia: `ms-petty-cash-api`), exponerlas como REST para el front, corregir los bugs críticos de dinero y seguridad, y retirar el código de bonos de Meteor.

### Decisiones tomadas

| Tema | Decisión |
|---|---|
| Consumidor | El front llama al servicio **por REST directo** (no a través del gateway). CORS configurable. |
| Autenticación | La sesión **la crea ms-oauth** (login). El servicio solo la valida (`x-token` → `GET /api/v1/session` de ms-oauth) y exige `x-secret`. |
| Autorización | Los endpoints de administración exigen `adminBlooming` en la sesión; los recursos de usuario exigen ser el dueño. |
| IDs | Los documentos nuevos usan UUID v4. El servicio también **consulta y opera las colecciones viejas**, cuyo `_id` es el random de 17 caracteres de Meteor, sin migrar esos `_id`. Los IDs de entrada se validan contra los dos formatos (`ERR_INVALID_ID` si no coincide ninguno). Ver spec 00, RF-00.19. |
| Datos | Mismas colecciones y nombres de campos que el legado. Soft delete con `isRemove`. Meteor y Go comparten la base durante el corte. |
| Bugs legacy | Se corrigen los **críticos** (dinero, seguridad, llamadas que nunca responden); el resto mantiene paridad. Cada spec documenta su decisión bug por bug. |
| Saldo | `BalancePort` escribe directo en `balanceDetail`, `balanceHistory`, `balanceOperation` y `balanceAccount`, replicando `balanceAccount.withdraw` y `balanceDetail.insert`, **dentro de una transacción** Mongo. |
| Maduración | La dispara ms-cronjob-go cada 5 minutos con `POST /internal/jobs/bond-maturity/run` (`x-secret`); el ms no usa librería de cron y toma un lock entre réplicas. Reemplaza al servicio externo `:8090` y al `SyncedCron` de Meteor. |
| Notificaciones | `notice` y `activity` se insertan directo; los emails van por **ms-notification** (handlers nuevos `blooming-*`). |
| Reportes | Por **ms-exporter**, con el mismo payload que hoy; el endpoint entrega el `.xlsx`. |

## 2. Specs

| # | Spec | Dominio | Fase |
|---|---|---|---|
| 00 | [Plataforma base](specs/00-plataforma-base.spec.md) | `shared`, `session`, `cmd/api` | F1 |
| 01 | [Tipos de bono](specs/01-bond-type.spec.md) | `bondType` | F3 |
| 02 | [Catálogo de bonos](specs/02-bond-object-catalogo.spec.md) | `bondObject` | F3 |
| 03 | [Proyección y gráficas](specs/03-proyeccion-y-graficas.spec.md) | `shared_domain`, `bondObject`, `bondBound` | F2, F3, F5 |
| 04 | [Compra](specs/04-bond-bound-compra.spec.md) | `bondBound` | F5 |
| 05 | [Venta anticipada y canje](specs/05-bond-bound-venta-canje.spec.md) | `bondBound` | F5 |
| 06 | [Consultas de bonos comprados](specs/06-bond-bound-consultas.spec.md) | `bondBound` | F6 |
| 07 | [Reportes y KPIs](specs/07-reportes-y-kpis.spec.md) | `bondBound` | F6 |
| 08 | [Historial](specs/08-bond-history.spec.md) | `bondHistory` | F2, F3, F5 |
| 09 | [Preferencias del inversionista](specs/09-bond-user-setting.spec.md) | `bondUserSetting` | F4 |
| 10 | [Maduración (cron)](specs/10-maduracion-cron.spec.md) | `bondBound`, ms-cronjob-go | F7 |

Todas las specs usan la misma plantilla:
1. Resumen y alcance.
2. Origen legacy.
3. Requisitos funcionales `RF-NN.x`.
4. Contrato API.
5. Reglas de negocio.
6. Modelo de datos.
7. Ports.
8. Bugs legacy.
9. Criterios de aceptación `CA-NN.x`.
10. Tareas `BLM-xxxx` (numeración global; ver [`tasks.md`](tasks.md) §1).
11. Fuera de alcance y preguntas abiertas.

## 3. Inventario: operación legacy → destino

**Leyenda de acciones:**
- **Migrar:** se convierte en un endpoint.
- **Fusionar:** una publicación o método que se une a otro endpoint, o que pasa a ser lógica interna.
- **Descartar:** no se porta. El motivo está en la spec.

| Archivo | Operación | Tipo | Destino | Spec | Acción |
|---|---|---|---|---|---|
| `bond.type.js` | `bondTypes` | pub | `GET /bond-types/active` | 01 | Fusionar |
| | `bondType.list` | método | `GET /bond-types` | 01 | Migrar |
| | `bondType.get` | método | `GET /bond-types/{id}` | 01 | Migrar |
| | `bondType.insert` | método | `POST /bond-types` | 01 | Migrar |
| | `bondType.update` | método | `PUT /bond-types/{id}` | 01 | Migrar |
| | `bondType.remove` | método | `DELETE /bond-types/{id}` | 01 | Migrar |
| | `bondType.listByActive` | método | `GET /bond-types/active` | 01 | Migrar |
| `bond.object.js` | `bondObjects` | pub | — (sin filtros ni permisos) | 02 | Descartar |
| | `getBondObjectsByBondTypeId` | pub | `GET /bond-types/{bondTypeId}/bond-objects` | 02 | Fusionar |
| | `getMostPurchased` | pub | `GET /bond-objects/most-purchased` | 02 | Fusionar |
| | `getLatest` | pub | `GET /bond-objects/latest` | 02 | Fusionar |
| | `bondObject.list` | método | `GET /bond-objects` | 02 | Migrar |
| | `bondObject.listGroupedByType` | método | `GET /bond-objects/grouped-by-type` | 02 | Migrar |
| | `bondObject.listInvestor` | método | `GET /bond-objects/{id}/investors` | 02 | Migrar |
| | `bondObject.get` | método | `GET /bond-objects/{id}` | 02 | Migrar |
| | `bondObject.getDataChart` | método | `POST /bond-objects/chart/preview`, `GET /bond-objects/{id}/chart` | 03 | Migrar |
| | `bondObject.insert` | método | `POST /bond-objects` | 02 | Migrar |
| | `bondObject.update` | método | `PUT /bond-objects/{id}` | 02 | Migrar |
| | `bondObject.massiveUpdateOfKeys` | método | `PATCH /bond-objects/keys` | 02 | Migrar |
| | `bondObject.remove` | método | `DELETE /bond-objects/{id}` | 02 | Migrar |
| | `bondObject.getChart` | método | — (stub que devuelve `{}`) | 02 | Descartar |
| | `bondObject.getBondObjectsByBondTypeId` | método | `GET /bond-types/{bondTypeId}/bond-objects` | 02 | Migrar |
| | `bondObject.getMostPurchased` | método | `GET /bond-objects/most-purchased` | 02 | Migrar |
| | `bondObject.getLatest` | método | `GET /bond-objects/latest` | 02 | Migrar |
| `bond.history.js` | `bondHistories` | pub | — (filtra por un campo inexistente) | 08 | Descartar |
| | `bondHistory.listByType` | método | `GET /bond-objects/{id}/history`, `GET /bond-bounds/{id}/history` | 08 | Migrar |
| `bond.userSetting.js` | `bondUserSettings` | pub | `GET /bond-user-settings/me` | 09 | Fusionar |
| | `bondUserSetting.list` | método | `GET /bond-user-settings` | 09 | Migrar |
| | `bondUserSetting.getByUserLogged` | método | `GET /bond-user-settings/me` | 09 | Migrar |
| | `bondUserSetting.get` | método | `GET /bond-user-settings/{id}` | 09 | Migrar |
| | `bondUserSetting.insert` | método | Autocreación interna | 09 | Fusionar |
| | `bondUserSetting.update` | método | — (`$set {}` vacío) | 09 | Descartar |
| | `bondUserSetting.setSelectedAccount` | método | `PATCH /bond-user-settings/me/selected-account`, `PUT /internal/bond-user-settings/{userId}/selected-account` | 09 | Migrar |
| | `bondUserSetting.setSelectedBondBound` | método | `PATCH /bond-user-settings/me/selected-bond-bound` | 09 | Migrar |
| | `bondUserSetting.setLanguage` | método | `PATCH /bond-user-settings/me/language` | 09 | Migrar |
| | `bondUserSetting.remove` | método | `DELETE /bond-user-settings/{id}` | 09 | Migrar |
| `bond.bound.js` | `bondBounds` | pub | — (sin filtros ni permisos) | 06 | Descartar |
| | `bondBoundByUserLogged` | pub | `GET /bond-bounds/me/recent` | 06 | Fusionar |
| | `getInvestorsById` | pub | `GET /bond-objects/{id}/bond-bounds` | 06 | Fusionar |
| | `bondBound.list` | método | `GET /bond-bounds` | 06 | Migrar |
| | `bondBound.listStatus` | método | `GET /bond-bounds/statuses` | 06 | Migrar |
| | `bondBound.listByUserIdAndBondObjectId` | método | `GET /users/{userId}/bond-bounds?bondObjectId=` | 06 | Migrar |
| | `bondBound.listByUserlogged` | método | `GET /bond-bounds/me` | 06 | Migrar |
| | `bondBound.get` | método | `GET /bond-bounds/{id}` | 06 | Migrar |
| | `bondBound.getDetails` | método | `GET /bond-bounds/{id}/details` | 06 | Migrar |
| | `bondBound.listByBondObjectId` | método | `GET /bond-objects/{id}/bond-bounds` | 06 | Migrar |
| | `bondBound.insert` | método | Caso de uso interno de la compra | 04 | Fusionar |
| | `bondBound.update` | método | — (`$set {}` vacío) | 06 | Descartar |
| | `bondBound.setStatus` | método | Caso de uso interno de la venta | 05 | Fusionar |
| | `bondBound.remove` | método | `DELETE /bond-bounds/{id}` | 06 | Migrar |
| | `bondBound.purchase` | método | `POST /bond-bounds/purchase` | 04 | Migrar |
| | `bondBound.preSale` | método | `GET /bond-bounds/{id}/pre-sale` | 05 | Migrar |
| | `bondBound.sale` | método | `POST /bond-bounds/{id}/sale` | 05 | Migrar |
| | `bondBound.getChartGuides` | método | `GET /bond-bounds/{id}/chart-guides` | 03 | Migrar |
| | `bondBound.reportPurchasedBonds` | método | `GET /bond-bounds/reports/purchased` | 07 | Migrar |
| | `bondBound.reportSummaryPurchasedBonds` | método | `GET /bond-bounds/reports/purchased-summary` | 07 | Migrar |
| | `bondBound.countByStatusId` | método | `GET /bond-bounds/kpis/count-by-status` | 07 | Migrar |
| | `bondBound.getBondPurchasedForChart` | método | `GET /bond-bounds/timeline` | 06 | Migrar |
| | `bondBound.byUserLogged` | método | `GET /bond-bounds/me/recent` | 06 | Migrar |
| | `bondBound.bulkPurchase` | método | `POST /bond-bounds/bulk-purchase` | 04 | Migrar |

**Totales:** 60 operaciones (50 métodos y 10 publicaciones): **44 migrar**, **10 fusionar** y **6 descartar**.

**Fuera de los 5 archivos**, también se migran:
- `Model.bondHistory.add` → `BondHistoryPort.Add` (spec 08);
- el cron `BondBoundCheckMaturity` y el servicio externo `:8090` → `POST /internal/jobs/bond-maturity/run`, disparado por ms-cronjob-go (spec 10);
- `balanceAccount.withdraw` y `balanceDetail.insert` (solo el camino de bonos) → `BalancePort` (spec 00).

## 4. Dependencias

```
                         ┌────────────── ms-oauth (GET /session) ◀── VerifySession
 Front ──REST──▶ ms-blooming-go
                         ├────────────── ms-notification (emails blooming-*)
                         └────────────── ms-exporter (Excel)

 Dentro del servicio (orden de inicialización):
   bondHistory ◀──────────────┬──────────────┬──────────────┐
   bondUserSetting ◀──────────│──────┐       │              │
   bondType ◀── bondObject ◀──┴── bondBound ─┘ (compra, venta, consultas, reportes, maduración)
        ▲          │
        └──────────┘  BondObjectCounterPort (link posterior a la inicialización)

 Colecciones Mongo:
   propias (lectura/escritura): bondType, bondObject, bondBound, bondHistory, bondUserSetting,
                                bondKeyCounter (nueva), bondJobLock (nueva)
   externas con escritura:      balanceAccount ($inc), balanceDetail, balanceHistory, balanceOperation, notice, activity
   externas solo lectura:       profile, agency, country
```

**Consumidores externos que hay que mantener compatibles:**
- `ms-gateway-gql/src/resolvers/B_BoundBound.js`: lee `bondBound` (`soldDate`, `tookDate`, `statusId`, `amount`) y `bondHistory` (`text: 'BOND_BOUND_STATUS_CHANGED'`, `params` de 2 elementos). Specs 05 (RF-05.6) y 10 (RF-10.4).
- `backend/server/model/balance.account.js` (`balanceAccount.insert`): actualiza `selectedAccountId`. Pasa a usar el endpoint interno de la spec 09 (RF-09.9).

## 5. Bugs críticos que corrige la migración

Cada spec tiene la tabla completa (63 bugs). `tasks.md` §5 los mapea a sus tareas. Los más graves son estos:

| Bug | Spec | Problema |
|---|---|---|
| B16 | 05 | `sale` no valida el dueño del bono: cualquier sesión vende un bono ajeno y cobra en **su** cuenta. |
| U6 | 09 | `setSelectedAccount` acepta `userId` del cliente: se puede redirigir a una cuenta propia el dinero de las ventas de otro usuario. |
| B18 | 04 | `bulkPurchase` debita la cuenta de **otro** usuario de la misma agencia. |
| B22 | 10 | La maduración legacy incluía bonos vendidos: un bono vendido volvía a `'2'` y se canjeaba otra vez (doble cobro). |
| B7 / B19 | 04 | Compra sin transacción: bonos sin cobro, o compras masivas parciales. |
| B3 / B4 | 05 | La venta acredita `NaN` si la cotización falla, y a veces nunca responde. |
| B15 / O9 | 02, 06 | Respuestas con el perfil completo, incluido el hash del password. |
| P3 | 00 | El retiro lee el saldo y después escribe; con compras concurrentes el saldo puede quedar negativo. |

## 6. Convivencia y corte

1. **Misma base de datos.** Meteor y Go leen y escriben las mismas colecciones; Go no renombra campos.
2. **Lecturas primero.** Catálogo, preferencias, historial y consultas se pueden habilitar en Go mientras Meteor sigue operando, porque son lecturas o escrituras sin carreras.
3. **Compra, venta y canje en un solo corte (BLM-0135).** Las claves de los bonos se generan con un contador nuevo (spec 04) que no es compatible con el `count + 1` de Meteor. Por eso se habilita Go y se deshabilitan los métodos de Meteor en el mismo deploy, después de sembrar los contadores.
4. **Maduración en un solo corte (BLM-0134).** Se apagan el `SyncedCron` y el servicio `:8090` y se activa el job de maduración de ms-cronjob-go (`MS_BLOOMING_URL`).
5. **Front por funcionalidad (BLM-0136).** DDP → REST, siguiendo los "cambios de contrato" de cada spec.
6. **Retiro (BLM-0137).** Se borran `bond.*.js` de Meteor.
7. **Monitoreo (BLM-0138).** Durante dos semanas: errores, auditoría de doble cobro del gateway y conciliación de saldos.

Antes del corte se ejecuta la auditoría de datos **BLM-0133**: claves y settings duplicados, numéricos guardados como string y bonos vendidos que maduraron (B22).

## 7. Fuera de alcance

- Login y sesiones (ms-oauth).
- El resto del módulo de saldo: depósitos de nómina, anulaciones, retiros manuales.
- Los resolvers `B_*` del gateway: siguen leyendo Mongo directo; solo se garantiza compatibilidad de formato.
- Los cambios en el front: se hacen en su repositorio, guiados por los contratos de estas specs.

## 8. Preguntas abiertas transversales

| # | Pregunta | Dónde |
|---|---|---|
| 1 | ¿Qué puerto usará el servicio? El esqueleto copió `25072` de petty-cash. | Spec 00 |
| 2 | ¿ms-oauth puede agregar `gmt`/`utcDiff` a la sesión? Hoy se usa un mapa por país. | Spec 00 |
| 3 | ¿Qué hace exactamente el servicio `:8090`? ¿Maduró bonos vendidos? | Spec 10, BLM-0133 |
| 4 | Los avisos de retiro al admin van a direcciones que quizá no son usuarios: ¿ms-notification necesita un grupo de emails crudos? | Spec 00 |
| 5 | ¿Se mantiene la compra de bonos de otra agencia? | Spec 04 |
| 6 | ¿Un bono `'1'` vencido que aún no maduró se vende o se canjea? | Spec 05 |
