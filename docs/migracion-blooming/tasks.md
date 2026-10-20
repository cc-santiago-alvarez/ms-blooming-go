# Tareas — Migración de Blooming a ms-blooming-go

> **Archivo generado.** La fuente de verdad de cada tarea es la tabla *Tareas* de su spec. Este archivo se regenera con el script de §6 y no se edita a mano. Las tareas de preparación, calidad, infraestructura y corte no salen de una spec funcional y se definen en el script (`EXTRA`).

## 1. Cómo leer este documento

- **ID:** `BLM-xxxx`, numeración global de 4 dígitos; es también el ticket del commit (`[BLM-0042] [MS-BLOOMING][ÁREA] …`). Una tarea nueva toma el siguiente ID libre (ver §6) y no se renumeran las existentes. Excepción: las tareas de F0 conservan su formato original `T-PREP.x`.
- **Fase:** la define la columna *Fase* de la tabla *Tareas* de cada spec; la spec de cada tarea se ve en el enlace al inicio de su fase.
- **Formato:** `- [ ] ID [P] tipo título`, seguido de una línea con lo que cubre y sus dependencias, y otra con los archivos.
- **`[P]`:** la tarea puede empezar al inicio de su fase, porque no depende de otra tarea de la misma fase. Las tareas `[P]` de una fase se pueden repartir en paralelo.
- **Cubre:** requisitos (`RF-NN.x`) que implementa, o criterios de aceptación (`CA-NN.x`) que verifica.
- **Tipo:** `impl` (código de producción), `test` u `ops` (repositorio, infraestructura, corte).
- **Reglas** (constitución §14):
  - toda tarea funcional cita al menos un RF o CA;
  - todo RF tiene al menos una tarea `impl` y una `test`;
  - un cambio de comportamiento se hace primero en la spec y después aquí (regenerando);
  - los comentarios del código no citan tareas, specs, RF ni CA; solo cada test cita en un comentario el CA que verifica.
- **PR:** cada PR cita sus `BLM-…`, sus `RF-…` y su spec.

## 2. Resumen

| Fase | Nombre | Tareas | impl | test | ops |
|---|---|---|---|---|---|
| F0 | Preparación del repositorio | 7 | 0 | 0 | 7 |
| F1 | Plataforma base (spec 00) | 28 | 22 | 6 | 0 |
| F2 | Historial y tipos compartidos (specs 08 y 03) | 11 | 9 | 2 | 0 |
| F3 | Catálogo (specs 01, 02 y 03) | 25 | 19 | 6 | 0 |
| F4 | Preferencias del inversionista (spec 09) | 11 | 9 | 2 | 0 |
| F5 | Compra, venta y canje (specs 04 y 05, más 03 y 08 sobre bondBound) | 26 | 18 | 8 | 0 |
| F6 | Consultas, reportes y KPIs (specs 06 y 07) | 16 | 11 | 5 | 0 |
| F7 | Maduración (spec 10) | 8 | 6 | 2 | 0 |
| F8 | Calidad transversal | 4 | 0 | 4 | 0 |
| F9 | Infraestructura (solo en `main`) | 4 | 0 | 0 | 4 |
| F10 | Corte y retiro del legado | 6 | 0 | 0 | 6 |
| | **Total** | **146** | **94** | **35** | **17** |

```
F0 ─▶ F1 ─┬─▶ F2 ─▶ F3 ─▶ F4 ─▶ F5 ─▶ F6 ─▶ F7 ─▶ F8 ─┬─▶ F10
          └─▶ F9 (infra en main) ──────────────────────────┘
```

F9 (infra, en `main`) puede empezar cuando BLM-0019 está terminada y correr en paralelo con F2–F8.

## 3. Tareas por fase

### F0 — Preparación del repositorio

- [ ] **T-PREP.1** [P] `ops` Inicializar el repositorio: `git init`, remoto `github.com/cc-santiago-alvarez/ms-blooming-go` y rama `master`. Los líderes de área lo llevan después a `futurapps`
  - cubre: — · depende de: —
  - archivos: `.git/`
- [ ] **T-PREP.2** `ops` `.gitignore` (`.claude/`, `.env*` salvo `.env.example`, `tmp/`). `.github/workflows/branch-flow.yml` lo agrega DevOps al integrar el repositorio en `futurapps`
  - cubre: — · depende de: T-PREP.1
  - archivos: `.gitignore`
- [ ] **T-PREP.3** `ops` `go.mod` (`module github.com/futurapps/ms-blooming-go`, `go 1.27.0`); eliminar `internal/database/mongo/connection.go`; `cmd/healthcheck` con `package main`; `go build ./...` compila
  - cubre: — · depende de: T-PREP.1
  - archivos: `go.mod`, `go.sum`, `cmd/healthcheck/main.go`, `internal/database/`
- [ ] **T-PREP.4** `ops` `.air.toml` para desarrollo con hot reload (sin `Makefile`)
  - cubre: — · depende de: T-PREP.3
  - archivos: `.air.toml`
- [ ] **T-PREP.5** `ops` `README.md` y `CLAUDE.md`: arquitectura, comandos, variables y enlace a `docs/constitution.md`
  - cubre: — · depende de: T-PREP.3
  - archivos: `README.md`, `CLAUDE.md`
- [ ] **T-PREP.6** `ops` Mongo local con replica set de un nodo (necesario para transacciones) para desarrollo y tests de integración: contenedor `atlas` (`mongodb/atlas`), conexión por las variables `FA_MONGO_*`
  - cubre: — · depende de: T-PREP.3
  - archivos: `README.md` (instrucciones)
- [ ] **T-PREP.7** [P] `ops` Dump anonimizado de testing de `bondType`, `bondObject`, `bondBound`, `bondHistory`, `bondUserSetting`, `balanceAccount` y `profile` para paridad e integración
  - cubre: — · depende de: —
  - archivos: fuera del repo (almacenamiento del equipo)

**Criterio de salida F0:** El esqueleto compila, está versionado y tiene entorno de pruebas con replica set.

### F1 — Plataforma base (spec 00)

Specs: [spec 00](specs/00-plataforma-base.spec.md).

- [ ] **BLM-0001** [P] `impl` Config: `ApiEnv` completo, carga de `.env` y `.env.<ENV>`, validación de requeridas, `.env.example` depurado
  - cubre: RF-00.1 · depende de: T-PREP.3
  - archivos: `cmd/api/config/config.go`, `cmd/api/config/environments.go`, `.env.example`
- [ ] **BLM-0002** `impl` Composition root: `serverDeps`, `InitServer`, conexión mongocc, `nexus.Server` con `/api/v1`, `cmd/main.go`
  - cubre: RF-00.1, RF-00.7 · depende de: BLM-0001
  - archivos: `cmd/main.go`, `cmd/api/server/server.go`, `server.config.go`, `server.deps.go`, `cmd/api/handlers/shared/handlers.response.go`
- [ ] **BLM-0003** `impl` Middleware `RequestLog` sin headers ni body
  - cubre: RF-00.13 · depende de: BLM-0002
  - archivos: `internal/core/shared/middlewares/requestLog.middleware.go`
- [ ] **BLM-0004** `impl` Middleware `ValidateApiSecret` con comparación en tiempo constante
  - cubre: RF-00.2 · depende de: BLM-0002
  - archivos: `internal/core/shared/middlewares/validateApiSecret.middleware.go`
- [ ] **BLM-0005** `impl` `session_domain` (`UserSession`, claves de contexto), cliente ms-oauth y middleware `VerifySession`
  - cubre: RF-00.3 · depende de: BLM-0002
  - archivos: `internal/core/session/domain/`, `internal/core/shared/adapters/oauth/session.client.go`, `shared/middlewares/verifySession.middleware.go`
- [ ] **BLM-0006** `impl` Resolución de `utcDiff` (sesión → país → `DEFAULT_UTC_DIFF`)
  - cubre: RF-00.4 · depende de: BLM-0005
  - archivos: `internal/core/session/domain/session.utcDiff.go`
- [ ] **BLM-0007** `impl` `RequireBloomingAdmin`, `IsOwnerOrAdmin` y `AgencyScope`
  - cubre: RF-00.5 · depende de: BLM-0005
  - archivos: `internal/core/shared/middlewares/requireBloomingAdmin.middleware.go`, `internal/core/shared/application/shared.authorization.go`
- [ ] **BLM-0008** `impl` CORS configurable con soporte de wildcard
  - cubre: RF-00.6 · depende de: BLM-0002
  - archivos: `cmd/api/server/server.config.go`, `internal/core/shared/middlewares/cors.go`
- [ ] **BLM-0009** [P] `impl` Helpers compartidos: `NewID`, `Now`, `Round2`, `StartOfDay`, `DaysBetween`, `MaskAccountNumber`, `RandomAlnum`, `Pagination`
  - cubre: RF-00.8, RF-00.18 · depende de: T-PREP.3
  - archivos: `internal/core/shared/domain/`, `internal/core/shared/adapters/utils/`
- [ ] **BLM-0010** `impl` `TransactionRunnerPort` y su adapter mongocc
  - cubre: RF-00.9 · depende de: BLM-0002
  - archivos: `internal/core/shared/ports/shared.ports.go`, `shared/infrastructure/mongocc/shared.transaction.adapter.go`
- [ ] **BLM-0011** `impl` Función pura `EffectiveDate(now, utcDiff)` con el cambio de año corregido
  - cubre: RF-00.10 · depende de: BLM-0009
  - archivos: `internal/core/shared/domain/balance.effectiveDate.go`
- [ ] **BLM-0012** `impl` `BalancePort`: `GetAccount` (con control de dueño), `Withdraw` y `Deposit` con escrituras en detail, history, operation y `$inc` condicional
  - cubre: RF-00.10 · depende de: BLM-0009, BLM-0010, BLM-0011
  - archivos: `internal/core/shared/domain/balance.domain.go`, `balance.dto.domain.go`, `balance.status.domain.go`, `shared.errors.go`, `shared.httpStatus.go`, `internal/core/shared/ports/shared.ports.go`, `internal/core/shared/infrastructure/mongocc/balance.mongocc.repository.go`, `balance.getAccount.adapter.go`, `balance.movement.adapter.go`, `balance.withdraw.adapter.go`, `balance.deposit.adapter.go`
- [ ] **BLM-0013** `impl` `NoticePort` y `ActivityPort` (inserts en `notice` y `activity`)
  - cubre: RF-00.11 · depende de: BLM-0009
  - archivos: `internal/core/shared/domain/notice.domain.go`, `notice.dto.domain.go`, `activity.domain.go`, `activity.dto.domain.go`, `internal/core/shared/ports/shared.ports.go`, `internal/core/shared/infrastructure/mongocc/notice.mongocc.repository.go`, `notice.add.adapter.go`, `activity.mongocc.repository.go`, `activity.add.adapter.go`
- [ ] **BLM-0014** `impl` `NotificationPort`: cliente HTTP de ms-notification, best-effort
  - cubre: RF-00.11 · depende de: BLM-0005
  - archivos: `internal/core/shared/ports/shared.ports.go`, `internal/core/shared/infrastructure/notification/notification.client.go`
- [ ] **BLM-0015** [P] `impl` **(repo ms-notification)** Handlers `notify-blooming-*` de §7.3 y grupo `to-raw-emails`
  - cubre: RF-00.11 · depende de: —
  - archivos: `ms-notification/src/api/modules/notify/handlers/notify-blooming-*.js`, `src/api/views/templates/notifyBlooming*.js`, `src/api/modules/notify/notify.helpers.js`, `notify.service.js`, `notify.constants.js`, `CLAUDE.md`
- [ ] **BLM-0016** `impl` `ExporterPort`: cliente HTTP de ms-exporter con conversión del Buffer
  - cubre: RF-00.14 · depende de: BLM-0005
  - archivos: `internal/core/shared/domain/shared.errors.go`, `shared.httpStatus.go`, `internal/core/shared/ports/shared.ports.go`, `internal/core/shared/infrastructure/exporter/exporter.client.go`
- [ ] **BLM-0017** `impl` `ProfileReader` y `AgencyReader` con proyecciones seguras
  - cubre: RF-00.15 · depende de: BLM-0002
  - archivos: `internal/core/shared/domain/profile.domain.go`, `internal/core/shared/ports/shared.ports.go`, `internal/core/shared/infrastructure/mongocc/profile.mongocc.repository.go`, `profile.getPublic.adapter.go`, `agency.mongocc.repository.go`, `agency.symbolCurrency.adapter.go`
- [ ] **BLM-0018** [P] `impl` Catálogo `Text`, `Format` y `FormatCurrency`
  - cubre: RF-00.12 · depende de: T-PREP.3
  - archivos: `internal/core/shared/domain/shared.text.go`, `shared.format.go`
- [ ] **BLM-0019** `impl` Graceful shutdown y binario `cmd/healthcheck` (`package main`, puerto de `API_PORT`)
  - cubre: RF-00.16 · depende de: BLM-0002
  - archivos: `cmd/api/server/server.go`, `cmd/api/config/config.go`, `cmd/healthcheck/main.go`
- [ ] **BLM-0020** [P] `impl` Regla de dependencias entre dominios en `architecture_test.go` (el test es la implementación de la regla)
  - cubre: RF-00.17 · depende de: T-PREP.3
  - archivos: `internal/core/architecture_test.go`
- [ ] **BLM-0021** `test` Tests de middlewares: secret, sesión, admin, scope de agencia, log, utcDiff
  - cubre: CA-00.2, CA-00.3, CA-00.4, CA-00.5, CA-00.6, CA-00.7, CA-00.8, CA-00.22, CA-00.34 · depende de: BLM-0003, BLM-0004, BLM-0005, BLM-0006, BLM-0007
  - archivos: `internal/core/shared/middlewares/*_test.go`, `internal/core/session/domain/*_test.go`, `internal/core/shared/application/*_test.go`
- [ ] **BLM-0022** `test` Tests de integración: transacción, `BalancePort` (incluida la concurrencia), notice, activity y readers
  - cubre: CA-00.13, CA-00.14, CA-00.15, CA-00.16, CA-00.18, CA-00.20, CA-00.24, CA-00.29, CA-00.33, CA-00.35, CA-00.36 · depende de: BLM-0010, BLM-0012, BLM-0013, BLM-0017
  - archivos: `internal/core/shared/infrastructure/mongocc/*_integration_test.go`
- [ ] **BLM-0024** `test` Tests de los clientes HTTP (ms-notification, ms-exporter) con `httptest`
  - cubre: CA-00.19, CA-00.23, CA-00.40, CA-00.41 · depende de: BLM-0014, BLM-0016
  - archivos: `internal/core/shared/infrastructure/notification/*_test.go`, `exporter/*_test.go`
- [ ] **BLM-0025** `test` Tests de arranque, CORS, errores, shutdown y healthcheck, más el caso negativo de la regla de dependencias (paquete de fixture con un import prohibido)
  - cubre: CA-00.1, CA-00.9, CA-00.10, CA-00.25, CA-00.26, CA-00.27, CA-00.42, CA-00.43 · depende de: BLM-0001, BLM-0002, BLM-0008, BLM-0019, BLM-0020
  - archivos: `cmd/api/config/*_test.go`, `cmd/api/server/*_test.go`, `cmd/healthcheck/main_test.go`, `internal/core/architecture_test.go`, `internal/core/testdata/`
- [ ] **BLM-0026** `impl` `IDFormat`/`IsValidID` en `shared_domain` y helper de handlers `PathID` (valida y responde `400 ERR_INVALID_ID`)
  - cubre: RF-00.19, RF-00.8 · depende de: BLM-0009, BLM-0002
  - archivos: `internal/core/shared/domain/shared.id.go`, `cmd/api/handlers/shared/handlers.pathId.go`
- [ ] **BLM-0027** `test` Tests de formatos de ID (tabla), del rechazo en el handler sin llegar al repositorio y de la lectura y actualización de documentos con los dos formatos
  - cubre: CA-00.30, CA-00.31, CA-00.32 · depende de: BLM-0026, BLM-0010
  - archivos: `internal/core/shared/domain/shared.id_test.go`, `cmd/api/handlers/shared/handlers.pathId_test.go`, `internal/core/shared/infrastructure/mongocc/shared.ids_integration_test.go`
- [ ] **BLM-0139** `impl` Paridad con el legado: `Round2` como `toFixed(2)` y plantilla `BOND_BOUND_MESSAGE_PURCHASED` literal
  - cubre: RF-00.8, RF-00.12 · depende de: BLM-0009, BLM-0018
  - archivos: `internal/core/shared/domain/shared.money.go`, `shared.text.go`
- [ ] **BLM-0023** `test` Tests unitarios de helpers, `EffectiveDate`, textos y paginación
  - cubre: CA-00.11, CA-00.12, CA-00.17, CA-00.21, CA-00.28, CA-00.37, CA-00.38, CA-00.39 · depende de: BLM-0009, BLM-0011, BLM-0018, BLM-0139
  - archivos: `internal/core/shared/domain/*_test.go`, `internal/core/shared/domain/testdata/`, `internal/core/shared/adapters/utils/*_test.go`

**Criterio de salida F1:** El servicio arranca, valida `x-secret` y la sesión de ms-oauth, y los ports compartidos (saldo, notificaciones, exportador) tienen tests.

- [ ] `go vet ./...` y `go test ./...` en verde; `go test -tags=integration ./...` en verde contra el replica set local.

### F2 — Historial y tipos compartidos (specs 08 y 03)

Specs: [spec 03](specs/03-proyeccion-y-graficas.spec.md), [spec 08](specs/08-bond-history.spec.md).

- [ ] **BLM-0028** [P] `impl` `shared_domain`: `EarlyCashTerm` (decodifica numéricos legacy con `FlexNumber`), `FlexNumber`, `ChartPoint` y `ChartGuide`
  - cubre: RF-03.1, RF-03.2, RF-03.5 · depende de: BLM-0009
  - archivos: `internal/core/shared/domain/shared.earlyCashTerm.go`, `shared.flexNumber.go`, `shared.chart.go`
- [ ] **BLM-0029** `impl` Función pura `SimulateProjection`
  - cubre: RF-03.1 · depende de: BLM-0028
  - archivos: `internal/core/shared/domain/shared.projection.go`
- [ ] **BLM-0030** `impl` Funciones puras `BuildObjectGuides` y `BuildBoundGuides`
  - cubre: RF-03.2, RF-03.5 · depende de: BLM-0028
  - archivos: `internal/core/shared/domain/shared.chartGuides.go`
- [ ] **BLM-0031** `test` Tests de tabla de `SimulateProjection` y de las guías, con casos calculados con el código legacy
  - cubre: CA-03.1, CA-03.2, CA-03.3, CA-03.4, CA-03.5 · depende de: BLM-0029, BLM-0030
  - archivos: `internal/core/shared/domain/shared.projection_test.go`, `shared.chartGuides_test.go`, `testdata/legacy-projection.json`, `legacy-guides.json`
- [ ] **BLM-0032** [P] `impl` Dominio: `BondHistory`, `HistoryType`, `ChangeEntry`, `AddParams` (con `Validation`) y errores
  - cubre: RF-08.1, RF-08.6 · depende de: BLM-0009
  - archivos: `internal/core/bondHistory/domain/*.go`, `internal/core/shared/domain/shared.text.go` (claves `BOND_SOLD` y `BOND_BOUND_STATUS_CHANGED`)
- [ ] **BLM-0033** `impl` Ports: `BondHistoryPort`, `BondHistoryApplicationPorts` y `BondHistoryMongoccRepositoryPorts`
  - cubre: RF-08.1, RF-08.2, RF-08.3 · depende de: BLM-0032
  - archivos: `internal/core/bondHistory/ports/bondHistory.ports.go`, `internal/core/bondHistory/domain/bondHistory.dto.domain.go` (tipo de respuesta `HistoryEntry`)
- [ ] **BLM-0034** `impl` Repositorio: `InsertOne`, `FindByRelated` y `EnsureIndexes`
  - cubre: RF-08.1, RF-08.2, RF-08.3 · depende de: BLM-0033
  - archivos: `internal/core/bondHistory/infrastructure/mongocc/bondHistory.mongocc.repository.go`, `*.adapter.go`
- [ ] **BLM-0035** `impl` Caso de uso `Add`, expuesto como `BondHistoryPort` (sesión o modo sistema) e inicialización en `server.bondHistory.go`
  - cubre: RF-08.1, RF-08.6 · depende de: BLM-0034, BLM-0005
  - archivos: `internal/core/bondHistory/application/bondHistory.add.go`, `bondHistory.application.go`, `internal/core/session/domain/session.context.go` (marca de modo sistema), `cmd/api/server/server.bondHistory.go`, `server.go`, `server.deps.go`
- [ ] **BLM-0036** [P] `impl` Textos faltantes en el catálogo y helper `FromNow` con los umbrales de moment
  - cubre: RF-08.4, RF-08.5 · depende de: BLM-0018
  - archivos: `internal/core/shared/domain/shared.text.go`, `shared.fromNow.go`
- [ ] **BLM-0037** `impl` Caso de uso `ListByRelated` con presentación (perfiles en lote, `Format`, nombres de estado y `FromNow`)
  - cubre: RF-08.2, RF-08.3, RF-08.4 · depende de: BLM-0034, BLM-0017, BLM-0036
  - archivos: `internal/core/bondHistory/application/bondHistory.listByRelated.go`, `bondHistory.present.go`, `bondHistory.application.go` (dependencia `ProfileReader`), `cmd/api/server/server.bondHistory.go`, `server.deps.go`, `server.config.go` (cableado del `ProfileReader`)
- [ ] **BLM-0038** `test` Tests unitarios de la presentación, los textos y `FromNow`
  - cubre: CA-08.5, CA-08.6, CA-08.7 · depende de: BLM-0037, BLM-0036
  - archivos: `internal/core/bondHistory/application/bondHistory.present_test.go`, `internal/core/shared/domain/shared.fromNow_test.go`, `testdata/legacy-fromnow.json`

**Criterio de salida F2:** Existe `BondHistoryPort.Add` y están listas las funciones puras de proyección que usan el catálogo y los bonos.

- [ ] `go vet ./...` y `go test ./...` en verde; `go test -tags=integration ./...` en verde contra el replica set local.

### F3 — Catálogo (specs 01, 02 y 03)

Specs: [spec 01](specs/01-bond-type.spec.md), [spec 02](specs/02-bond-object-catalogo.spec.md), [spec 03](specs/03-proyeccion-y-graficas.spec.md), [spec 08](specs/08-bond-history.spec.md).

- [ ] **BLM-0039** [P] `impl` Dominio: entidad `BondType`, tipo `AgencyIDs` (string o arreglo), DTOs con `Normalize`/`Validation`, errores y `httpStatus`
  - cubre: RF-01.4, RF-01.5, RF-01.7 · depende de: BLM-0009
  - archivos: `internal/core/bondType/domain/*.go`, `internal/core/shared/domain/shared.agencyIds.go`
- [ ] **BLM-0040** `impl` Ports: `BondTypeApplicationPorts`, `BondTypeMongoccRepositoryPorts`, `BondObjectCounterPort`
  - cubre: RF-01.1, RF-01.2, RF-01.3, RF-01.4, RF-01.5, RF-01.6 · depende de: BLM-0039
  - archivos: `internal/core/bondType/ports/bondType.ports.go`
- [ ] **BLM-0041** `impl` Repositorio mongocc (un archivo por método) y `EnsureIndexes`
  - cubre: RF-01.1, RF-01.2, RF-01.3, RF-01.5, RF-01.6 · depende de: BLM-0040
  - archivos: `internal/core/bondType/infrastructure/mongocc/bondType.mongocc.repository.go`, `*.adapter.go`
- [ ] **BLM-0042** `impl` Casos de uso `List`, `ListActive` (con `numOfBonds`), `GetByID` y `GetManyByIDs`
  - cubre: RF-01.1, RF-01.2, RF-01.3 · depende de: BLM-0041
  - archivos: `internal/core/bondType/application/bondType.application.go`, `bondType.list.go`, `bondType.listActive.go`, `bondType.getById.go`, `bondType.getManyByIds.go`
- [ ] **BLM-0043** `impl` Casos de uso `Create` y `Update`
  - cubre: RF-01.4, RF-01.5 · depende de: BLM-0041
  - archivos: `internal/core/bondType/application/bondType.create.go`, `bondType.update.go`
- [ ] **BLM-0044** `test` Tests de integración del repositorio (filtros por agencia, `AgencyIDs` legacy e ids Meteor)
  - cubre: CA-01.1, CA-01.8, CA-01.9 · depende de: BLM-0041
  - archivos: `internal/core/bondType/infrastructure/mongocc/*_integration_test.go`
- [ ] **BLM-0045** `impl` Dominio: `BondObject` (con `FlexNumber` para numéricos legacy), DTOs con defaults legacy en `Normalize()` y validaciones de §5.1, errores y `httpStatus`
  - cubre: RF-02.4, RF-02.5 · depende de: BLM-0009, BLM-0039, BLM-0028
  - archivos: `internal/core/bondObject/domain/*.go`
- [ ] **BLM-0046** `impl` Ports: `BondObjectApplicationPorts`, `BondObjectMongoccRepositoryPorts`, `BondBoundStatsPort`
  - cubre: RF-02.1, RF-02.2, RF-02.3, RF-02.4, RF-02.5, RF-02.6, RF-02.7, RF-02.8, RF-02.9, RF-02.10, RF-02.11 · depende de: BLM-0045
  - archivos: `internal/core/bondObject/ports/bondObject.ports.go`, `internal/core/bondObject/domain/bondObject.dto.domain.go` (tipos de salida y `VisibleFilter`)
- [ ] **BLM-0047** `impl` Repositorio mongocc (`find`, agregación agrupada con `$lookup`, filtro de privados, `updateMany` de claves, `softDelete`, conteos por tipo) y `EnsureIndexes`
  - cubre: RF-02.1, RF-02.2, RF-02.6, RF-02.7, RF-02.9, RF-02.11, RF-01.2, RF-01.6 · depende de: BLM-0046
  - archivos: `internal/core/bondObject/infrastructure/mongocc/bondObject.mongocc.repository.go`, `*.adapter.go`
- [ ] **BLM-0048** `impl` Adapter de solo lectura `BondBoundStatsPort` e implementación de `BondObjectCounterPort` para la spec 01
  - cubre: RF-02.8, RF-02.10, RF-01.2, RF-01.6 · depende de: BLM-0047
  - archivos: `internal/core/bondObject/infrastructure/mongocc/bondObject.bondBoundStats.adapter.go`, `internal/core/bondObject/application/bondObject.application.go`, `bondObject.counter.go`
- [ ] **BLM-0049** `impl` Caso de uso `Remove` con guarda de bondObjects vivos y conexión `linkBondObjectToBondType`
  - cubre: RF-01.6 · depende de: BLM-0041, BLM-0048
  - archivos: `internal/core/bondType/application/bondType.remove.go`, `cmd/api/server/server.bondType.go`
- [ ] **BLM-0050** `impl` Handlers y registro de rutas (con `RequireBloomingAdmin` donde corresponde)
  - cubre: RF-01.1, RF-01.2, RF-01.3, RF-01.4, RF-01.5, RF-01.6 · depende de: BLM-0042, BLM-0043, BLM-0049, BLM-0007, BLM-0026
  - archivos: `cmd/api/handlers/bondTypes/*.go`, `cmd/api/server/server.bondType.go`
- [ ] **BLM-0051** `test` Tests unitarios de casos de uso y handlers con fakes
  - cubre: CA-01.2, CA-01.3, CA-01.4, CA-01.5, CA-01.6, CA-01.7 · depende de: BLM-0050
  - archivos: `internal/core/bondType/application/*_test.go`, `cmd/api/handlers/bondTypes/*_test.go`
- [ ] **BLM-0052** `impl` Casos de uso admin de consulta: `List`, `ListGroupedByType` (sin N+1), `GetByID` con `authorizedUser` seguro y `GetByIDIncludingRemoved`
  - cubre: RF-02.1, RF-02.2, RF-02.3 · depende de: BLM-0047, BLM-0017
  - archivos: `internal/core/bondObject/application/bondObject.list.go`, `bondObject.listGroupedByType.go`, `bondObject.getById.go`
- [ ] **BLM-0053** `impl` Función pura `BuildChanges(old, new)` con escalares, fechas, arreglos y `earlyCashTerm.*`
  - cubre: RF-02.5 · depende de: BLM-0045
  - archivos: `internal/core/bondObject/domain/bondObject.changes.go`
- [ ] **BLM-0054** `impl` Casos de uso `Create`, `Update`, `UpdateKeys` y `Remove`, con history y activity best-effort
  - cubre: RF-02.4, RF-02.5, RF-02.6, RF-02.7 · depende de: BLM-0047, BLM-0053, BLM-0035, BLM-0013, BLM-0042
  - archivos: `internal/core/bondObject/application/bondObject.create.go`, `bondObject.update.go`, `bondObject.updateKeys.go`, `bondObject.remove.go`
- [ ] **BLM-0055** `impl` Casos de uso `ListInvestors`, `ListByType` (privados y `maturityDateFormatted`), `MostPurchased`, `Latest` y `FindVisibleForPurchase`
  - cubre: RF-02.8, RF-02.9, RF-02.10, RF-02.11 · depende de: BLM-0048
  - archivos: `internal/core/bondObject/application/bondObject.listInvestors.go`, `bondObject.listByType.go`, `bondObject.mostPurchased.go`, `bondObject.latest.go`
- [ ] **BLM-0056** `impl` Handlers, registro de rutas, `server.bondObject.go` y `linkBondObjectToBondType`
  - cubre: RF-02.1, RF-02.2, RF-02.3, RF-02.4, RF-02.5, RF-02.6, RF-02.7, RF-02.8, RF-02.9, RF-02.10, RF-02.11 · depende de: BLM-0052, BLM-0054, BLM-0055, BLM-0007, BLM-0026
  - archivos: `cmd/api/handlers/bondObjects/*.go`, `cmd/api/server/server.bondObject.go`
- [ ] **BLM-0057** `test` Tests unitarios de casos de uso y handlers con fakes
  - cubre: CA-02.3, CA-02.4, CA-02.5, CA-02.6, CA-02.9, CA-02.14 · depende de: BLM-0056
  - archivos: `internal/core/bondObject/application/*_test.go`, `cmd/api/handlers/bondObjects/*_test.go`
- [ ] **BLM-0058** `test` Tests de integración del repositorio y de las agregaciones
  - cubre: CA-01.3, CA-02.1, CA-02.2, CA-02.7, CA-02.8, CA-02.10, CA-02.11, CA-02.12, CA-02.13, CA-02.15 · depende de: BLM-0047, BLM-0048
  - archivos: `internal/core/bondObject/infrastructure/mongocc/*_integration_test.go`
- [ ] **BLM-0059** `test` Test de tabla de `BuildChanges`
  - cubre: CA-02.6 · depende de: BLM-0053
  - archivos: `internal/core/bondObject/domain/bondObject.changes_test.go`
- [ ] **BLM-0060** `impl` Casos de uso `ChartPreview` y `ChartByID` en bondObject, con visibilidad de RF-02.9
  - cubre: RF-03.3, RF-03.4 · depende de: BLM-0029, BLM-0030, BLM-0055
  - archivos: `internal/core/bondObject/application/bondObject.chartPreview.go`, `bondObject.chartById.go`
- [ ] **BLM-0061** `impl` Handlers y rutas de bondObject: `POST /bond-objects/chart/preview` y `GET /bond-objects/{id}/chart`
  - cubre: RF-03.3, RF-03.4 · depende de: BLM-0060, BLM-0056, BLM-0026
  - archivos: `cmd/api/handlers/bondObjects/bondObjects.chartPreview.post.go`, `bondObjects.chart.get.go`
- [ ] **BLM-0062** `test` Tests de casos de uso y handlers de la proyección de bondObject (permisos, visibilidad, validación)
  - cubre: CA-03.6, CA-03.7, CA-03.9 · depende de: BLM-0061
  - archivos: `internal/core/bondObject/application/bondObject.chart*_test.go`, `cmd/api/handlers/bondObjects/bondObjects.chart*_test.go`
- [ ] **BLM-0063** `impl` Handler `GET /bond-objects/{id}/history` (admin)
  - cubre: RF-08.2 · depende de: BLM-0037, BLM-0056, BLM-0026
  - archivos: `cmd/api/handlers/bondObjects/bondObjects.history.get.go`

**Criterio de salida F3:** Tipos y productos se administran y consultan por REST, con historial, proyección y permisos.

- [ ] `go vet ./...` y `go test ./...` en verde; `go test -tags=integration ./...` en verde contra el replica set local.

### F4 — Preferencias del inversionista (spec 09)

Specs: [spec 09](specs/09-bond-user-setting.spec.md).

- [ ] **BLM-0064** [P] `impl` Dominio: `BondUserSetting`, DTOs (`SetSelectedAccountParams`, `SetLanguageParams`, …) con `Validation` y errores
  - cubre: RF-09.1, RF-09.3, RF-09.4, RF-09.5 · depende de: BLM-0009
  - archivos: `internal/core/bondUserSetting/domain/*.go`
- [ ] **BLM-0065** `impl` Ports y repositorio: `FindByUser`, `UpsertDefault`, `UpsertField`, `FindPage`, `FindOneByID`, `SoftDelete`, `EnsureIndexes` (índice único parcial) y script de auditoría de duplicados
  - cubre: RF-09.1, RF-09.3, RF-09.4, RF-09.5, RF-09.6, RF-09.7, RF-09.8 · depende de: BLM-0064
  - archivos: `internal/core/bondUserSetting/ports/*.go`, `internal/core/bondUserSetting/infrastructure/mongocc/*.adapter.go`, `cmd/tools/dedupusersettings/main.go`
- [ ] **BLM-0066** `impl` Adapters de solo lectura `BoundSummary` y `FirstAccountID`
  - cubre: RF-09.1, RF-09.2 · depende de: BLM-0065
  - archivos: `internal/core/bondUserSetting/infrastructure/mongocc/bondUserSetting.boundSummary.adapter.go`, `bondUserSetting.firstAccount.adapter.go`
- [ ] **BLM-0067** `impl` Casos de uso `GetMine` (autocreación atómica y datos agregados) y `GetByUser`
  - cubre: RF-09.1, RF-09.2, RF-09.10 · depende de: BLM-0066, BLM-0012, BLM-0017
  - archivos: `internal/core/bondUserSetting/application/bondUserSetting.getMine.go`, `bondUserSetting.getByUser.go`
- [ ] **BLM-0068** `impl` Casos de uso `SetSelectedAccount` (cuenta del usuario), `SetSelectedBondBound` y `SetLanguage`, con upsert
  - cubre: RF-09.3, RF-09.4, RF-09.5 · depende de: BLM-0065, BLM-0012
  - archivos: `internal/core/bondUserSetting/application/bondUserSetting.set*.go`
- [ ] **BLM-0069** `impl` Casos de uso admin `List`, `GetByID` y `Remove`
  - cubre: RF-09.6, RF-09.7, RF-09.8 · depende de: BLM-0065, BLM-0007
  - archivos: `internal/core/bondUserSetting/application/bondUserSetting.list.go`, `bondUserSetting.getById.go`, `bondUserSetting.remove.go`
- [ ] **BLM-0070** `impl` Caso de uso `SyncSelectedAccount` y endpoint interno con `NoRequiresAuthentication`
  - cubre: RF-09.9 · depende de: BLM-0065, BLM-0004, BLM-0026
  - archivos: `internal/core/bondUserSetting/application/bondUserSetting.syncSelectedAccount.go`, `cmd/api/handlers/bondUserSettings/bondUserSettings.internalSelectedAccount.put.go`
- [ ] **BLM-0071** `impl` Handlers y `server.bondUserSetting.go`
  - cubre: RF-09.1, RF-09.3, RF-09.4, RF-09.5, RF-09.6, RF-09.7, RF-09.8 · depende de: BLM-0067, BLM-0068, BLM-0069, BLM-0026
  - archivos: `cmd/api/handlers/bondUserSettings/*.go`, `cmd/api/server/server.bondUserSetting.go`
- [ ] **BLM-0072** `impl` **(repo backend, Meteor)** `balanceAccount.insert` llama por HTTP al endpoint interno en lugar de `Caller.call('bondUserSetting.setSelectedAccount')`
  - cubre: RF-09.9 · depende de: BLM-0070
  - archivos: `backend/server/model/balance.account.js`
- [ ] **BLM-0073** `test` Tests unitarios de casos de uso y handlers (validaciones, dueño de la cuenta, U6, permisos)
  - cubre: CA-09.2, CA-09.5, CA-09.6, CA-09.7, CA-09.8, CA-09.9, CA-09.11 · depende de: BLM-0071
  - archivos: `internal/core/bondUserSetting/application/*_test.go`, `cmd/api/handlers/bondUserSettings/*_test.go`
- [ ] **BLM-0074** `test` Tests de integración: autocreación concurrente, `boundSummary` y endpoint interno
  - cubre: CA-09.1, CA-09.3, CA-09.4, CA-09.10, CA-09.12 · depende de: BLM-0071, BLM-0070
  - archivos: `internal/core/bondUserSetting/infrastructure/mongocc/*_integration_test.go`

**Criterio de salida F4:** El setting se autocrea, la cuenta destino está protegida (U6) y existe el endpoint interno para Meteor.

- [ ] `go vet ./...` y `go test ./...` en verde; `go test -tags=integration ./...` en verde contra el replica set local.

### F5 — Compra, venta y canje (specs 04 y 05, más 03 y 08 sobre bondBound)

Specs: [spec 03](specs/03-proyeccion-y-graficas.spec.md), [spec 04](specs/04-bond-bound-compra.spec.md), [spec 05](specs/05-bond-bound-venta-canje.spec.md), [spec 08](specs/08-bond-history.spec.md).

- [ ] **BLM-0075** [P] `impl` Dominio: `BondBound`, `BondBoundStatus` (con nombres e `IsValid`), `BondKeyCounter`, DTOs `PurchaseParams`/`BulkPurchaseParams`, errores y `httpStatus`
  - cubre: RF-04.1, RF-04.2, RF-04.8 · depende de: BLM-0028, BLM-0009
  - archivos: `internal/core/bondBound/domain/*.go`
- [ ] **BLM-0076** `impl` Ports: `BondBoundApplicationPorts`, `BondBoundMongoccRepositoryPorts`, `KeyCounterPort`
  - cubre: RF-04.1, RF-04.2, RF-04.3, RF-04.8 · depende de: BLM-0075
  - archivos: `internal/core/bondBound/ports/bondBound.ports.go`
- [ ] **BLM-0077** `impl` Repositorio base: `InsertMany`, `FindOneByID`, `UpdateStatusIf` y `EnsureIndexes` (§6)
  - cubre: RF-04.8 · depende de: BLM-0076
  - archivos: `internal/core/bondBound/infrastructure/mongocc/bondBound.mongocc.repository.go`, `bondBound.insertMany.adapter.go`, `bondBound.findOneById.adapter.go`, `bondBound.updateStatusIf.adapter.go`, `bondBound.ensureIndexes.adapter.go`
- [ ] **BLM-0078** `impl` Caso de uso `ChartGuides` en bondBound, con control de dueño o admin
  - cubre: RF-03.5 · depende de: BLM-0030, BLM-0077, BLM-0007
  - archivos: `internal/core/bondBound/application/bondBound.chartGuides.go`
- [ ] **BLM-0079** `impl` `KeyCounterPort.Reserve`, script de siembra de contadores y auditoría de claves duplicadas existentes
  - cubre: RF-04.3 · depende de: BLM-0077
  - archivos: `internal/core/bondBound/infrastructure/mongocc/bondBound.keyCounter.adapter.go`, `cmd/tools/seedkeycounters/main.go`
- [ ] **BLM-0080** `impl` Función pura `NewBoundFromObject` (snapshot, `maturityDate` exacta, `token`)
  - cubre: RF-04.4 · depende de: BLM-0075
  - archivos: `internal/core/bondBound/domain/bondBound.new.go`
- [ ] **BLM-0081** `impl` Caso de uso `Purchase`: validaciones de §5.1 y transacción de §5.2
  - cubre: RF-04.1, RF-04.5 · depende de: BLM-0077, BLM-0079, BLM-0080, BLM-0010, BLM-0012, BLM-0055, BLM-0035
  - archivos: `internal/core/bondBound/application/bondBound.purchase.go`, `bondBound.validatePurchase.go`
- [ ] **BLM-0082** `impl` Caso de uso `BulkPurchase`: cantidad, validaciones compartidas y transacción única
  - cubre: RF-04.2, RF-04.5 · depende de: BLM-0081
  - archivos: `internal/core/bondBound/application/bondBound.bulkPurchase.go`
- [ ] **BLM-0083** `impl` Notificaciones post-commit de la compra individual y la masiva (armado de textos y tabla)
  - cubre: RF-04.6, RF-04.7 · depende de: BLM-0081, BLM-0082, BLM-0013, BLM-0014, BLM-0018, BLM-0015
  - archivos: `internal/core/bondBound/application/bondBound.purchase.notify.go`
- [ ] **BLM-0084** `impl` Handlers `purchase` y `bulk-purchase`, y `server.bondBound.go` (inicialización del dominio y registro)
  - cubre: RF-04.1, RF-04.2 · depende de: BLM-0081, BLM-0082, BLM-0026
  - archivos: `cmd/api/handlers/bondBounds/bondBounds.handler.go`, `bondBounds.purchase.post.go`, `bondBounds.bulkPurchase.post.go`, `cmd/api/server/server.bondBound.go`
- [ ] **BLM-0085** `impl` Handler y ruta `GET /bond-bounds/{id}/chart-guides`
  - cubre: RF-03.5 · depende de: BLM-0078, BLM-0084, BLM-0026
  - archivos: `cmd/api/handlers/bondBounds/bondBounds.chartGuides.get.go`
- [ ] **BLM-0086** `test` Tests del caso de uso y del handler de guías de bondBound (dueño, admin, 404)
  - cubre: CA-03.8 · depende de: BLM-0085
  - archivos: `internal/core/bondBound/application/bondBound.chartGuides_test.go`
- [ ] **BLM-0087** `test` Tests unitarios de validaciones (tabla y orden), de `NewBoundFromObject` y de los handlers
  - cubre: CA-04.2, CA-04.3, CA-04.4, CA-04.6, CA-04.8 · depende de: BLM-0084
  - archivos: `internal/core/bondBound/application/bondBound.purchase_test.go`, `internal/core/bondBound/domain/bondBound.new_test.go`
- [ ] **BLM-0088** `test` Tests de integración: compra completa, rollback, concurrencia, contador e índices
  - cubre: CA-04.1, CA-04.5, CA-04.7, CA-04.9, CA-04.10, CA-04.13, CA-04.14 · depende de: BLM-0084
  - archivos: `internal/core/bondBound/application/bondBound.purchase_integration_test.go`, `internal/core/bondBound/infrastructure/mongocc/*_integration_test.go`
- [ ] **BLM-0089** `test` Tests de las notificaciones (textos, tabla y tolerancia a fallos)
  - cubre: CA-04.11, CA-04.12 · depende de: BLM-0083
  - archivos: `internal/core/bondBound/application/bondBound.purchase.notify_test.go`
- [ ] **BLM-0090** `impl` Función pura `QuoteSale` y `SaleProjection`, con la corrección de B1 y sin truncados
  - cubre: RF-05.2 · depende de: BLM-0075, BLM-0009
  - archivos: `internal/core/bondBound/domain/bondBound.quoteSale.go`
- [ ] **BLM-0091** `impl` Función pura `SaleTransitionFor(bound)` (tabla de §5.3) y sus errores
  - cubre: RF-05.3 · depende de: BLM-0075
  - archivos: `internal/core/bondBound/domain/bondBound.saleTransition.go`, `bondBound.errors.go`
- [ ] **BLM-0092** `impl` Caso de uso `PreSale`, con control de dueño o admin
  - cubre: RF-05.1 · depende de: BLM-0090, BLM-0091, BLM-0077, BLM-0006, BLM-0007
  - archivos: `internal/core/bondBound/application/bondBound.preSale.go`
- [ ] **BLM-0093** `impl` Caso de uso `Sale`: dueño, transición, cotización, cuenta destino y transacción con `UpdateStatusIf`, `Deposit` y el historial
  - cubre: RF-05.3, RF-05.4, RF-05.5, RF-05.6 · depende de: BLM-0092, BLM-0067, BLM-0010, BLM-0012, BLM-0035
  - archivos: `internal/core/bondBound/application/bondBound.sale.go`
- [ ] **BLM-0094** `impl` Notificaciones post-commit de venta y canje (11 parámetros, snapshot y fechas)
  - cubre: RF-05.7 · depende de: BLM-0093, BLM-0013, BLM-0014, BLM-0015, BLM-0018
  - archivos: `internal/core/bondBound/application/bondBound.sale.notify.go`
- [ ] **BLM-0095** `impl` Handlers `pre-sale` (GET) y `sale` (POST)
  - cubre: RF-05.1, RF-05.3 · depende de: BLM-0092, BLM-0093, BLM-0084, BLM-0026
  - archivos: `cmd/api/handlers/bondBounds/bondBounds.preSale.get.go`, `bondBounds.sale.post.go`
- [ ] **BLM-0096** `test` Tests de tabla de `QuoteSale` contra casos calculados con el código legacy
  - cubre: CA-05.1, CA-05.2, CA-05.3, CA-05.4 · depende de: BLM-0090
  - archivos: `internal/core/bondBound/domain/bondBound.quoteSale_test.go`
- [ ] **BLM-0097** `test` Tests unitarios de `PreSale` y `Sale` con fakes (dueño, estados, cotización inválida, cuenta destino)
  - cubre: CA-05.5, CA-05.6, CA-05.9, CA-05.10, CA-05.11, CA-05.12 · depende de: BLM-0095
  - archivos: `internal/core/bondBound/application/bondBound.preSale_test.go`, `bondBound.sale_test.go`
- [ ] **BLM-0098** `test` Tests de integración: venta, canje, concurrencia, formato del historial para el gateway y notificaciones
  - cubre: CA-05.7, CA-05.8, CA-05.13, CA-05.14, CA-05.15, CA-05.16 · depende de: BLM-0095, BLM-0094
  - archivos: `internal/core/bondBound/application/bondBound.sale_integration_test.go`
- [ ] **BLM-0099** `impl` Caso de uso `HistoryOf` en bondBound (control de dueño y delegación) y handler `GET /bond-bounds/{id}/history`
  - cubre: RF-08.3 · depende de: BLM-0037, BLM-0077, BLM-0084, BLM-0026
  - archivos: `internal/core/bondBound/application/bondBound.history.go`, `cmd/api/handlers/bondBounds/bondBounds.history.get.go`
- [ ] **BLM-0100** `test` Tests de integración: `Add` (con transacción), listados, permisos y lectura en lote
  - cubre: CA-08.1, CA-08.2, CA-08.3, CA-08.4, CA-08.8, CA-08.9 · depende de: BLM-0063, BLM-0099
  - archivos: `internal/core/bondHistory/infrastructure/mongocc/*_integration_test.go`, `cmd/api/handlers/*/*.history_test.go`

**Criterio de salida F5:** Compra individual y masiva, cotización, venta y canje son transaccionales y no hay doble cobro.

- [ ] `go vet ./...` y `go test ./...` en verde; `go test -tags=integration ./...` en verde contra el replica set local.

### F6 — Consultas, reportes y KPIs (specs 06 y 07)

Specs: [spec 06](specs/06-bond-bound-consultas.spec.md), [spec 07](specs/07-reportes-y-kpis.spec.md).

- [ ] **BLM-0101** [P] `impl` Dominio: funciones puras `Accumulated`, `Progress` y `FormatShortDate`; DTOs `ListFilters` y `DateFilter` con `Normalize`/`Validation`
  - cubre: RF-06.1, RF-06.11, RF-06.12 · depende de: BLM-0075, BLM-0009
  - archivos: `internal/core/bondBound/domain/bondBound.derived.go`, `bondBound.filters.domain.go`
- [ ] **BLM-0102** `impl` Constructor de rangos de fecha `DateRangeStage(filter, field, u)`, reutilizable por la spec 07
  - cubre: RF-06.12 · depende de: BLM-0101
  - archivos: `internal/core/bondBound/infrastructure/mongocc/bondBound.dateRange.go`
- [ ] **BLM-0103** `impl` Adapters de consulta: listado admin (`$facet`, lookups, búsqueda escapada), por usuario y producto, míos paginados, recientes, detalle, por producto, línea de tiempo (`%Y` en `u`) y `softDelete`
  - cubre: RF-06.1, RF-06.3, RF-06.4, RF-06.5, RF-06.7, RF-06.8, RF-06.9, RF-06.10 · depende de: BLM-0102, BLM-0077
  - archivos: `internal/core/bondBound/infrastructure/mongocc/bondBound.list*.adapter.go`, `bondBound.findDetails.adapter.go`, `bondBound.timeline.adapter.go`, `bondBound.softDelete.adapter.go`
- [ ] **BLM-0104** `impl` Casos de uso admin: `List`, `ListByUserAndObject`, `ListByObject` (perfil público) y `Remove`
  - cubre: RF-06.1, RF-06.3, RF-06.8, RF-06.10 · depende de: BLM-0103, BLM-0017
  - archivos: `internal/core/bondBound/application/bondBound.list.go`, `bondBound.listByUserAndObject.go`, `bondBound.listByObject.go`, `bondBound.remove.go`
- [ ] **BLM-0105** `impl` Casos de uso del inversionista: `Statuses`, `ListMine`, `ListRecent`, `GetByID`, `GetDetails` y `Timeline`, con control de dueño
  - cubre: RF-06.2, RF-06.4, RF-06.5, RF-06.6, RF-06.7, RF-06.9, RF-06.11 · depende de: BLM-0103, BLM-0007
  - archivos: `internal/core/bondBound/application/bondBound.statuses.go`, `bondBound.listMine.go`, `bondBound.listRecent.go`, `bondBound.getById.go`, `bondBound.getDetails.go`, `bondBound.timeline.go`
- [ ] **BLM-0106** `impl` Handlers y rutas, incluidas `/users/{userId}/bond-bounds` y `/bond-objects/{id}/bond-bounds`
  - cubre: RF-06.1, RF-06.2, RF-06.3, RF-06.4, RF-06.5, RF-06.6, RF-06.7, RF-06.8, RF-06.9, RF-06.10 · depende de: BLM-0104, BLM-0105, BLM-0084, BLM-0026
  - archivos: `cmd/api/handlers/bondBounds/bondBounds.*.get.go`, `bondBounds.remove.delete.go`
- [ ] **BLM-0107** `test` Tests de tabla de los derivados y de los rangos de fecha
  - cubre: CA-06.3, CA-06.4, CA-06.5 · depende de: BLM-0101, BLM-0102
  - archivos: `internal/core/bondBound/domain/bondBound.derived_test.go`, `internal/core/bondBound/infrastructure/mongocc/bondBound.dateRange_test.go`
- [ ] **BLM-0108** `test` Tests unitarios de casos de uso y handlers (permisos, dueño, 404)
  - cubre: CA-06.6, CA-06.7, CA-06.10, CA-06.12 · depende de: BLM-0106
  - archivos: `internal/core/bondBound/application/bondBound.list*_test.go`, `bondBound.get*_test.go`
- [ ] **BLM-0109** `test` Tests de integración de las agregaciones (paginación, búsqueda escapada, scope de agencia, proyecciones)
  - cubre: CA-06.1, CA-06.2, CA-06.8, CA-06.9, CA-06.11, CA-06.13, CA-06.14 · depende de: BLM-0103
  - archivos: `internal/core/bondBound/infrastructure/mongocc/bondBound.list*_integration_test.go`
- [ ] **BLM-0110** `impl` DTOs `ReportFilters` y `KpiFilters`, con la normalización de `statusId` de §5 y sus errores
  - cubre: RF-07.1, RF-07.2, RF-07.3 · depende de: BLM-0101
  - archivos: `internal/core/bondBound/domain/bondBound.reports.domain.go`
- [ ] **BLM-0111** `impl` Adapters `ReportPurchasedRows`, `ReportSummaryRows` y `CountByStatus` (con `DateRangeStage` y búsqueda escapada)
  - cubre: RF-07.1, RF-07.2, RF-07.3 · depende de: BLM-0110, BLM-0102, BLM-0103
  - archivos: `internal/core/bondBound/infrastructure/mongocc/bondBound.report*.adapter.go`, `bondBound.countByStatus.adapter.go`
- [ ] **BLM-0112** `impl` Casos de uso `ReportPurchased` y `ReportSummary` (moneda, `ExporterPort` y nombre de archivo)
  - cubre: RF-07.1, RF-07.2, RF-07.4 · depende de: BLM-0111, BLM-0016, BLM-0017
  - archivos: `internal/core/bondBound/application/bondBound.reportPurchased.go`, `bondBound.reportSummary.go`
- [ ] **BLM-0113** `impl` Caso de uso `CountByStatus`, con el alcance por rol
  - cubre: RF-07.3 · depende de: BLM-0111, BLM-0007
  - archivos: `internal/core/bondBound/application/bondBound.countByStatus.go`
- [ ] **BLM-0114** `impl` Handlers de reportes (respuesta binaria y headers) y del KPI
  - cubre: RF-07.1, RF-07.2, RF-07.3, RF-07.4 · depende de: BLM-0112, BLM-0113, BLM-0084, BLM-0026
  - archivos: `cmd/api/handlers/bondBounds/bondBounds.reportPurchased.get.go`, `bondBounds.reportSummary.get.go`, `bondBounds.countByStatus.get.go`
- [ ] **BLM-0115** `test` Tests unitarios: normalización, permisos, fallo de ms-exporter y headers
  - cubre: CA-07.1, CA-07.4, CA-07.6, CA-07.7 · depende de: BLM-0114
  - archivos: `internal/core/bondBound/application/bondBound.report*_test.go`, `bondBound.countByStatus_test.go`
- [ ] **BLM-0116** `test` Tests de integración de las agregaciones (rangos, búsqueda, agrupado)
  - cubre: CA-07.2, CA-07.3, CA-07.5, CA-07.8 · depende de: BLM-0111
  - archivos: `internal/core/bondBound/infrastructure/mongocc/bondBound.report*_integration_test.go`

**Criterio de salida F6:** Todas las lecturas de bonos, reportes Excel y KPI responden con permisos y filtros de fecha correctos.

- [ ] `go vet ./...` y `go test ./...` en verde; `go test -tags=integration ./...` en verde contra el replica set local.

### F7 — Maduración (spec 10)

Specs: [spec 10](specs/10-maduracion-cron.spec.md).

- [ ] **BLM-0117** [P] `impl` Dominio: `MaturityResult` y función pura `BuildMaturityTable(rows)` (HTML del legado)
  - cubre: RF-10.3, RF-10.5 · depende de: BLM-0075
  - archivos: `internal/core/bondBound/domain/bondBound.maturity.go`
- [ ] **BLM-0118** [P] `impl` Adapter `FindMaturable` (lotes con cursor por `_id`), con transición condicional solo desde `'1'`
  - cubre: RF-10.3 · depende de: BLM-0077
  - archivos: `internal/core/bondBound/infrastructure/mongocc/bondBound.findMaturable.adapter.go`
- [ ] **BLM-0119** [P] `impl` `JobLockPort` y su adapter mongocc (`bondJobLock`)
  - cubre: RF-10.2 · depende de: BLM-0010
  - archivos: `internal/core/shared/ports/shared.jobLock.ports.go`, `internal/core/shared/infrastructure/mongocc/shared.jobLock.adapter.go`
- [ ] **BLM-0120** `impl` Caso de uso `RunMaturity`: lock, lotes, transición, historial de sistema, agrupación y notificaciones
  - cubre: RF-10.3, RF-10.4, RF-10.5, RF-10.6 · depende de: BLM-0117, BLM-0118, BLM-0119, BLM-0035, BLM-0013, BLM-0014, BLM-0015
  - archivos: `internal/core/bondBound/application/bondBound.runMaturity.go`
- [ ] **BLM-0121** `impl` Scheduler `internal/cron` (robfig, `BOND_MATURITY_CRON`, parada en el shutdown) y registro del job
  - cubre: RF-10.1 · depende de: BLM-0120, BLM-0019
  - archivos: `internal/cron/infra/scheduler.go`, `internal/cron/jobs/bondMaturity.job.go`, `cmd/api/server/server.go`
- [ ] **BLM-0122** `impl` Endpoint interno de ejecución manual
  - cubre: RF-10.7 · depende de: BLM-0120, BLM-0004
  - archivos: `cmd/api/handlers/jobs/jobs.bondMaturity.post.go`
- [ ] **BLM-0123** `test` Tests unitarios: tabla HTML, agrupación por usuario, estados excluidos, fallo parcial
  - cubre: CA-10.3, CA-10.6, CA-10.7 · depende de: BLM-0120
  - archivos: `internal/core/bondBound/application/bondBound.runMaturity_test.go`, `internal/core/bondBound/domain/bondBound.maturity_test.go`
- [ ] **BLM-0124** `test` Tests de integración: idempotencia, lock concurrente, historial de sistema, registro del cron y endpoint
  - cubre: CA-10.1, CA-10.2, CA-10.4, CA-10.5, CA-10.8, CA-10.9 · depende de: BLM-0121, BLM-0122
  - archivos: `internal/core/bondBound/application/bondBound.runMaturity_integration_test.go`, `internal/cron/infra/scheduler_test.go`

**Criterio de salida F7:** El cron madura solo bonos `'1'`, una sola réplica a la vez, con avisos agrupados.

- [ ] `go vet ./...` y `go test ./...` en verde; `go test -tags=integration ./...` en verde contra el replica set local.

### F8 — Calidad transversal

- [ ] **BLM-0125** [P] `test` Arnés de paridad: ejecutar los métodos Meteor (DDP) y los endpoints Go sobre el mismo dump y comparar las fórmulas (`getDataChart`, `preSale`, derivados del listado)
  - cubre: RF-03.1, RF-05.2, RF-06.11 · depende de: T-PREP.7, BLM-0031, BLM-0096, BLM-0107
  - archivos: `cmd/tools/parity/`
- [ ] **BLM-0126** [P] `test` Suite end-to-end de flujos completos: compra → venta; compra → maduración → canje; compra masiva → listado → reporte
  - cubre: RF-04.1, RF-04.2, RF-05.3, RF-10.3, RF-07.1 · depende de: BLM-0088, BLM-0098, BLM-0124, BLM-0116
  - archivos: `test/e2e/`
- [ ] **BLM-0127** [P] `test` Matriz de autorización rol × endpoint (usuario, dueño, admin, otra agencia) ejecutada sobre todas las rutas registradas
  - cubre: RF-00.5, RF-04.1, RF-05.3, RF-06.6, RF-09.3 · depende de: BLM-0106, BLM-0071, BLM-0095
  - archivos: `test/authz/`
- [ ] **BLM-0128** [P] `test` Prueba de carga: 50 compras concurrentes sobre la misma cuenta y p95 del listado de administración con 100k bonos
  - cubre: RF-04.5, RF-06.1 · depende de: BLM-0088, BLM-0109
  - archivos: `test/load/`

**Criterio de salida F8:** Paridad numérica con el legado comprobada y matriz de autorización en verde.

- [ ] `go vet ./...` y `go test ./...` en verde; `go test -tags=integration ./...` en verde contra el replica set local.

### F9 — Infraestructura (solo en `main`)

- [ ] **BLM-0129** [P] `ops` `.deploy/Dockerfile` multi-stage (`golang:1.27-alpine` → `scratch`, binario de healthcheck, `USER 1001`, `EXPOSE ${API_PORT}`) y `.dockerignore` — **solo en `main`**
  - cubre: RF-00.16 · depende de: BLM-0019
  - archivos: `.deploy/Dockerfile`, `.dockerignore`
- [ ] **BLM-0130** `ops` `.deploy/docker-compose.yml` (Swarm, 3 réplicas, healthcheck, `/opt/ms-blooming-go/.env`), `docker-stack-deploy.sh` y `.deploy-testing/` — **solo en `main`**
  - cubre: — · depende de: BLM-0129
  - archivos: `.deploy/`, `.deploy-testing/`
- [ ] **BLM-0131** `ops` `.github/workflows/docker-build.yml` (release `X.Y.Z-live`/`-testing`, Jenkins, Discord), con `go vet` y `go test` antes del build — **solo en `main`**
  - cubre: — · depende de: BLM-0129
  - archivos: `.github/workflows/docker-build.yml`
- [ ] **BLM-0132** `ops` Entornos: `.env` en `/opt/ms-blooming-go/` (testing y live), replica set confirmado, exposición pública del servicio (proxy y DNS) para el front y `CORS_ORIGINS`
  - cubre: RF-00.1, RF-00.6 · depende de: BLM-0130
  - archivos: servidores

**Criterio de salida F9:** La imagen se construye y despliega en testing con healthcheck en verde.

### F10 — Corte y retiro del legado

- [ ] **BLM-0133** [P] `ops` Auditoría de datos previa al corte: claves duplicadas (B8), settings duplicados (U3), numéricos guardados como string, bonos `'2'`/`'6'` con `soldDate` (B22)
  - cubre: RF-04.3, RF-09.1, RF-10.3 · depende de: T-PREP.7, BLM-0079, BLM-0065
  - archivos: informe en `docs/migracion-blooming/`
- [ ] **BLM-0134** [P] `ops` Revisar el servicio `:8090` (`MS_FINOBONDS_FEED_URL`), desactivarlo junto con el `SyncedCron` `BondBoundCheckMaturity` de Meteor y activar `BOND_MATURITY_CRON` en el mismo deploy
  - cubre: RF-10.1 · depende de: BLM-0124, BLM-0132
  - archivos: `backend/server/lib/crons.js`, `.env` de servidores
- [ ] **BLM-0135** `ops` Corte de compra y venta: correr la siembra de contadores (BLM-0079), habilitar los endpoints Go y deshabilitar `bondBound.purchase`, `bulkPurchase`, `preSale` y `sale` en Meteor **en el mismo deploy**
  - cubre: RF-04.3, RF-04.1, RF-05.3 · depende de: BLM-0079, BLM-0098, BLM-0133
  - archivos: `backend/server/model/bond.bound.js`
- [ ] **BLM-0136** [P] `ops` Migrar el front de DDP a REST por funcionalidad (catálogo → preferencias e historial → compra y venta → consultas y reportes), con la guía de cambios de contrato de cada spec
  - cubre: RF-01.1, RF-02.9, RF-09.1, RF-04.1, RF-06.4 · depende de: BLM-0132
  - archivos: repo del front
- [ ] **BLM-0137** `ops` Retirar `bond.*.js` de Meteor y la referencia a `MS_FINOBONDS_FEED_URL`, una vez desplegado BLM-0072
  - cubre: — · depende de: BLM-0134, BLM-0135, BLM-0136, BLM-0072
  - archivos: `backend/server/model/bond.*.js`, `backend/server/lib/_constants.js`
- [ ] **BLM-0138** `ops` Monitoreo durante dos semanas: errores 5xx, `B_BondBound_Audit_doubleProcess` en el gateway y conciliación de `balanceAccount.available` contra la suma de `balanceDetail`
  - cubre: RF-05.5, RF-10.3 · depende de: BLM-0135
  - archivos: dashboards y logs

**Criterio de salida F10:** El front usa solo REST, Meteor no tiene código de bonos y el monitoreo no muestra incidencias.

## 4. Matriz de trazabilidad: requisito → tareas

| Spec | RF | Implementan | Verifican (tests) | CA | Operación / corte |
|---|---|---|---|---|---|
| 00 | RF-00.1 | BLM-0001, BLM-0002 | BLM-0025 | CA-00.1 | BLM-0132 |
| 00 | RF-00.2 | BLM-0004 | BLM-0021, BLM-0025 | CA-00.2, CA-00.43 | — |
| 00 | RF-00.3 | BLM-0005 | BLM-0021 | CA-00.3, CA-00.4, CA-00.5 | — |
| 00 | RF-00.4 | BLM-0006 | BLM-0021 | CA-00.6 | — |
| 00 | RF-00.5 | BLM-0007 | BLM-0021, BLM-0127 | CA-00.7, CA-00.8, CA-00.34 | — |
| 00 | RF-00.6 | BLM-0008 | BLM-0025 | CA-00.9 | BLM-0132 |
| 00 | RF-00.7 | BLM-0002 | BLM-0025 | CA-00.10, CA-00.42 | — |
| 00 | RF-00.8 | BLM-0009, BLM-0026, BLM-0139 | BLM-0023, BLM-0027 | CA-00.11, CA-00.12, CA-00.30, CA-00.37 | — |
| 00 | RF-00.9 | BLM-0010 | BLM-0022 | CA-00.13 | — |
| 00 | RF-00.10 | BLM-0011, BLM-0012 | BLM-0022, BLM-0023 | CA-00.14, CA-00.15, CA-00.16, CA-00.17, CA-00.18, CA-00.29, CA-00.33 | — |
| 00 | RF-00.11 | BLM-0013, BLM-0014, BLM-0015 | BLM-0022, BLM-0024 | CA-00.19, CA-00.20, CA-00.35, CA-00.40 | — |
| 00 | RF-00.12 | BLM-0018, BLM-0139 | BLM-0023 | CA-00.21, CA-00.38 | — |
| 00 | RF-00.13 | BLM-0003 | BLM-0021 | CA-00.22 | — |
| 00 | RF-00.14 | BLM-0016 | BLM-0024 | CA-00.23, CA-00.41 | — |
| 00 | RF-00.15 | BLM-0017 | BLM-0022 | CA-00.24, CA-00.36 | — |
| 00 | RF-00.16 | BLM-0019 | BLM-0025 | CA-00.25, CA-00.26 | BLM-0129 |
| 00 | RF-00.17 | BLM-0020 | BLM-0025 | CA-00.27 | — |
| 00 | RF-00.18 | BLM-0009 | BLM-0023 | CA-00.28, CA-00.39 | — |
| 00 | RF-00.19 | BLM-0026 | BLM-0027 | CA-00.30, CA-00.31, CA-00.32 | — |
| 01 | RF-01.1 | BLM-0040, BLM-0041, BLM-0042, BLM-0050 | BLM-0044, BLM-0051 | CA-01.1, CA-01.2 | BLM-0136 |
| 01 | RF-01.2 | BLM-0040, BLM-0041, BLM-0042, BLM-0047, BLM-0048, BLM-0050 | BLM-0051, BLM-0058 | CA-01.3 | — |
| 01 | RF-01.3 | BLM-0040, BLM-0041, BLM-0042, BLM-0050 | BLM-0044, BLM-0051 | CA-01.2, CA-01.4, CA-01.9 | — |
| 01 | RF-01.4 | BLM-0039, BLM-0040, BLM-0043, BLM-0050 | BLM-0051 | CA-01.2, CA-01.5 | — |
| 01 | RF-01.5 | BLM-0039, BLM-0040, BLM-0041, BLM-0043, BLM-0050 | BLM-0044, BLM-0051 | CA-01.6, CA-01.9 | — |
| 01 | RF-01.6 | BLM-0040, BLM-0041, BLM-0047, BLM-0048, BLM-0049, BLM-0050 | BLM-0044, BLM-0051 | CA-01.7, CA-01.9 | — |
| 01 | RF-01.7 | BLM-0039 | BLM-0044 | CA-01.8 | — |
| 02 | RF-02.1 | BLM-0046, BLM-0047, BLM-0052, BLM-0056 | BLM-0058 | CA-02.1 | — |
| 02 | RF-02.2 | BLM-0046, BLM-0047, BLM-0052, BLM-0056 | BLM-0058 | CA-02.2 | — |
| 02 | RF-02.3 | BLM-0046, BLM-0052, BLM-0056 | BLM-0057, BLM-0058 | CA-02.3, CA-02.15 | — |
| 02 | RF-02.4 | BLM-0045, BLM-0046, BLM-0054, BLM-0056 | BLM-0057 | CA-02.4, CA-02.5, CA-02.14 | — |
| 02 | RF-02.5 | BLM-0045, BLM-0046, BLM-0053, BLM-0054, BLM-0056 | BLM-0057, BLM-0058, BLM-0059 | CA-02.6, CA-02.7, CA-02.14 | — |
| 02 | RF-02.6 | BLM-0046, BLM-0047, BLM-0054, BLM-0056 | BLM-0058 | CA-02.8 | — |
| 02 | RF-02.7 | BLM-0046, BLM-0047, BLM-0054, BLM-0056 | BLM-0057 | CA-02.9, CA-02.14 | — |
| 02 | RF-02.8 | BLM-0046, BLM-0048, BLM-0055, BLM-0056 | BLM-0058 | CA-02.10 | — |
| 02 | RF-02.9 | BLM-0046, BLM-0047, BLM-0055, BLM-0056 | BLM-0058 | CA-02.11, CA-02.15 | BLM-0136 |
| 02 | RF-02.10 | BLM-0046, BLM-0048, BLM-0055, BLM-0056 | BLM-0058 | CA-02.12 | — |
| 02 | RF-02.11 | BLM-0046, BLM-0047, BLM-0055, BLM-0056 | BLM-0058 | CA-02.13 | — |
| 03 | RF-03.1 | BLM-0028, BLM-0029 | BLM-0031, BLM-0125 | CA-03.1, CA-03.2, CA-03.3 | — |
| 03 | RF-03.2 | BLM-0028, BLM-0030 | BLM-0031 | CA-03.4, CA-03.5 | — |
| 03 | RF-03.3 | BLM-0060, BLM-0061 | BLM-0062 | CA-03.6 | — |
| 03 | RF-03.4 | BLM-0060, BLM-0061 | BLM-0062 | CA-03.7, CA-03.9 | — |
| 03 | RF-03.5 | BLM-0028, BLM-0030, BLM-0078, BLM-0085 | BLM-0086 | CA-03.8 | — |
| 04 | RF-04.1 | BLM-0075, BLM-0076, BLM-0081, BLM-0084 | BLM-0087, BLM-0088, BLM-0126, BLM-0127 | CA-04.1, CA-04.2, CA-04.3, CA-04.4, CA-04.14 | BLM-0135, BLM-0136 |
| 04 | RF-04.2 | BLM-0075, BLM-0076, BLM-0082, BLM-0084 | BLM-0087, BLM-0088, BLM-0126 | CA-04.3, CA-04.5, CA-04.6 | — |
| 04 | RF-04.3 | BLM-0076, BLM-0079 | BLM-0088 | CA-04.7, CA-04.14 | BLM-0133, BLM-0135 |
| 04 | RF-04.4 | BLM-0080 | BLM-0087, BLM-0088 | CA-04.1, CA-04.8, CA-04.14 | — |
| 04 | RF-04.5 | BLM-0081, BLM-0082 | BLM-0088, BLM-0128 | CA-04.9, CA-04.10 | — |
| 04 | RF-04.6 | BLM-0083 | BLM-0089 | CA-04.11 | — |
| 04 | RF-04.7 | BLM-0083 | BLM-0089 | CA-04.12 | — |
| 04 | RF-04.8 | BLM-0075, BLM-0076, BLM-0077 | BLM-0088 | CA-04.13 | — |
| 05 | RF-05.1 | BLM-0092, BLM-0095 | BLM-0097 | CA-05.5, CA-05.6 | — |
| 05 | RF-05.2 | BLM-0090 | BLM-0096, BLM-0097, BLM-0125 | CA-05.1, CA-05.2, CA-05.3, CA-05.4, CA-05.5 | — |
| 05 | RF-05.3 | BLM-0091, BLM-0093, BLM-0095 | BLM-0097, BLM-0098, BLM-0126, BLM-0127 | CA-05.7, CA-05.8, CA-05.9, CA-05.10, CA-05.11, CA-05.16 | BLM-0135 |
| 05 | RF-05.4 | BLM-0093 | BLM-0097, BLM-0098 | CA-05.7, CA-05.12 | — |
| 05 | RF-05.5 | BLM-0093 | BLM-0098 | CA-05.7, CA-05.8, CA-05.13, CA-05.16 | BLM-0138 |
| 05 | RF-05.6 | BLM-0093 | BLM-0098 | CA-05.14, CA-05.16 | — |
| 05 | RF-05.7 | BLM-0094 | BLM-0098 | CA-05.15 | — |
| 06 | RF-06.1 | BLM-0101, BLM-0103, BLM-0104, BLM-0106 | BLM-0109, BLM-0128 | CA-06.1, CA-06.2 | — |
| 06 | RF-06.2 | BLM-0105, BLM-0106 | BLM-0108 | CA-06.6 | — |
| 06 | RF-06.3 | BLM-0103, BLM-0104, BLM-0106 | BLM-0108 | CA-06.7 | — |
| 06 | RF-06.4 | BLM-0103, BLM-0105, BLM-0106 | BLM-0109 | CA-06.8, CA-06.14 | BLM-0136 |
| 06 | RF-06.5 | BLM-0103, BLM-0105, BLM-0106 | BLM-0109 | CA-06.9 | — |
| 06 | RF-06.6 | BLM-0105, BLM-0106 | BLM-0108, BLM-0127 | CA-06.10 | — |
| 06 | RF-06.7 | BLM-0103, BLM-0105, BLM-0106 | BLM-0108, BLM-0109 | CA-06.10, CA-06.14 | — |
| 06 | RF-06.8 | BLM-0103, BLM-0104, BLM-0106 | BLM-0109 | CA-06.11 | — |
| 06 | RF-06.9 | BLM-0103, BLM-0105, BLM-0106 | BLM-0108 | CA-06.12 | — |
| 06 | RF-06.10 | BLM-0103, BLM-0104, BLM-0106 | BLM-0109 | CA-06.13 | — |
| 06 | RF-06.11 | BLM-0101, BLM-0105 | BLM-0107, BLM-0125 | CA-06.4, CA-06.5 | — |
| 06 | RF-06.12 | BLM-0101, BLM-0102 | BLM-0107 | CA-06.3 | — |
| 07 | RF-07.1 | BLM-0110, BLM-0111, BLM-0112, BLM-0114 | BLM-0115, BLM-0116, BLM-0126 | CA-07.1, CA-07.2, CA-07.7, CA-07.8 | — |
| 07 | RF-07.2 | BLM-0110, BLM-0111, BLM-0112, BLM-0114 | BLM-0115, BLM-0116 | CA-07.3, CA-07.7 | — |
| 07 | RF-07.3 | BLM-0110, BLM-0111, BLM-0113, BLM-0114 | BLM-0115, BLM-0116 | CA-07.5, CA-07.6, CA-07.8 | — |
| 07 | RF-07.4 | BLM-0112, BLM-0114 | BLM-0115 | CA-07.1, CA-07.4 | — |
| 08 | RF-08.1 | BLM-0032, BLM-0033, BLM-0034, BLM-0035 | BLM-0100 | CA-08.1, CA-08.2 | — |
| 08 | RF-08.2 | BLM-0033, BLM-0034, BLM-0037, BLM-0063 | BLM-0100 | CA-08.3, CA-08.9 | — |
| 08 | RF-08.3 | BLM-0033, BLM-0034, BLM-0037, BLM-0099 | BLM-0100 | CA-08.4, CA-08.9 | — |
| 08 | RF-08.4 | BLM-0036, BLM-0037 | BLM-0038, BLM-0100 | CA-08.5, CA-08.6, CA-08.7, CA-08.8 | — |
| 08 | RF-08.5 | BLM-0036 | BLM-0038 | CA-08.6 | — |
| 08 | RF-08.6 | BLM-0032, BLM-0035 | BLM-0100 | CA-08.1 | — |
| 09 | RF-09.1 | BLM-0064, BLM-0065, BLM-0066, BLM-0067, BLM-0071 | BLM-0073, BLM-0074 | CA-09.1, CA-09.2, CA-09.3, CA-09.12 | BLM-0133, BLM-0136 |
| 09 | RF-09.2 | BLM-0066, BLM-0067 | BLM-0074 | CA-09.4 | — |
| 09 | RF-09.3 | BLM-0064, BLM-0065, BLM-0068, BLM-0071 | BLM-0073, BLM-0074, BLM-0127 | CA-09.5, CA-09.6, CA-09.12 | — |
| 09 | RF-09.4 | BLM-0064, BLM-0065, BLM-0068, BLM-0071 | BLM-0073 | CA-09.6 | — |
| 09 | RF-09.5 | BLM-0064, BLM-0065, BLM-0068, BLM-0071 | BLM-0073 | CA-09.6, CA-09.7 | — |
| 09 | RF-09.6 | BLM-0065, BLM-0069, BLM-0071 | BLM-0073 | CA-09.8 | — |
| 09 | RF-09.7 | BLM-0065, BLM-0069, BLM-0071 | BLM-0073 | CA-09.9 | — |
| 09 | RF-09.8 | BLM-0065, BLM-0069, BLM-0071 | BLM-0073 | CA-09.9 | — |
| 09 | RF-09.9 | BLM-0070, BLM-0072 | BLM-0074 | CA-09.10 | — |
| 09 | RF-09.10 | BLM-0067 | BLM-0073 | CA-09.11 | — |
| 10 | RF-10.1 | BLM-0121 | BLM-0124 | CA-10.1 | BLM-0134 |
| 10 | RF-10.2 | BLM-0119 | BLM-0124 | CA-10.2 | — |
| 10 | RF-10.3 | BLM-0117, BLM-0118, BLM-0120 | BLM-0123, BLM-0124, BLM-0126 | CA-10.3, CA-10.4, CA-10.9 | BLM-0133, BLM-0138 |
| 10 | RF-10.4 | BLM-0120 | BLM-0124 | CA-10.5, CA-10.9 | — |
| 10 | RF-10.5 | BLM-0117, BLM-0120 | BLM-0123, BLM-0124 | CA-10.4, CA-10.6 | — |
| 10 | RF-10.6 | BLM-0120 | BLM-0123 | CA-10.7 | — |
| 10 | RF-10.7 | BLM-0122 | BLM-0124 | CA-10.8 | — |

## 5. Bugs del legado → tareas

| Bug | Spec | Decisión | Tarea | Descripción |
|---|---|---|---|---|
| P1 | [spec 00](specs/00-plataforma-base.spec.md) | Corregir | BLM-0011 | `effectiveDate` en diciembre (día ≥ 26) genera `año-1-15` sin incrementar el año. |
| P2 | [spec 00](specs/00-plataforma-base.spec.md) | Corregir | BLM-0005 | `getSession` deja `session = false` si el token no existe, y los métodos fallan con TypeError. |
| P3 | [spec 00](specs/00-plataforma-base.spec.md) | Corregir | BLM-0012 | `withdraw` lee el saldo y después escribe; dos compras simultáneas pueden dejar el saldo negativo. |
| P4 | [spec 00](specs/00-plataforma-base.spec.md) | Corregir | BLM-0012 | `balanceDetail.insert` hace `future.return` antes de `setAmounts`; el saldo se actualiza después de responder. |
| T1 | [spec 01](specs/01-bond-type.spec.md) | Corregir | BLM-0049 | `remove` cuenta los bondObjects borrados como bloqueantes. |
| T2 | [spec 01](specs/01-bond-type.spec.md) | Mantener | — | `numOfBonds` cuenta bondObjects de todas las agencias, pero el listado del inversionista filtra por agencia. |
| T3 | [spec 01](specs/01-bond-type.spec.md) | Corregir | BLM-0042 | `get` no filtra `isRemove` y devuelve `undefined` si no existe. |
| T4 | [spec 01](specs/01-bond-type.spec.md) | Corregir | BLM-0043 | `update` escribe `undefined` (`null`) en los campos que no llegan. |
| O1 | [spec 01](specs/01-bond-type.spec.md) | Corregir | BLM-0039 | `agencyId` se filtra con `$all` en tipos y con `$in: [agencyId]` en objetos (arreglo dentro de arreglo). |
| O2 | [spec 02](specs/02-bond-object-catalogo.spec.md) | Corregir | BLM-0053 | El diff de `update` no detecta cambios en `earlyCashTerm` ni en fechas. |
| O3 | [spec 02](specs/02-bond-object-catalogo.spec.md) | Corregir | BLM-0055 | Los privados se filtran en el método `getBondObjectsByBondTypeId`, pero no en la publicación ni en `getMostPurchased`/`getLatest`. |
| O4 | [spec 02](specs/02-bond-object-catalogo.spec.md) | Corregir | BLM-0054 | `massiveUpdateOfKeys` nunca responde, y el cliente queda colgado. |
| O5 | [spec 02](specs/02-bond-object-catalogo.spec.md) | Corregir | BLM-0052 | `listGroupedByType` hace N+1 consultas a `bondType` y filtra por `data.agencyId` en lugar de la sesión. |
| O6 | [spec 02](specs/02-bond-object-catalogo.spec.md) | Corregir | BLM-0045 | Sin validaciones en alta y edición: `inflectionPoint == maturityTimeInDays` produce división por cero en las fórmulas. |
| O9 | [spec 02](specs/02-bond-object-catalogo.spec.md) | Corregir | BLM-0052 | `get` devuelve los perfiles completos de `authorizedUser`, incluido el hash del password. |
| O10 | [spec 03](specs/03-proyeccion-y-graficas.spec.md) | Corregir | BLM-0029 | `getDataChart` y `getChartGuides` aplican `parseInt` sobre `amount` y porcentajes: un bono de 1000.50 con 7.5% se simula como 1000 con 7%. |
| O7 | [spec 03](specs/03-proyeccion-y-graficas.spec.md) | Corregir | BLM-0060 | `getDataChart` con `simulation` pero sin `preview` devuelve `data: []`. |
| O8 | [spec 03](specs/03-proyeccion-y-graficas.spec.md) | Corregir | BLM-0030 | `getChartGuides` formatea las fechas con la zona del servidor; `getDataChart` usa `session.utcDiff`. |
| B17 | [spec 03](specs/03-proyeccion-y-graficas.spec.md) | Corregir | BLM-0078 | `getChartGuides` no valida que el bono exista (crashea) ni que sea del usuario. |
| B7 | [spec 04](specs/04-bond-bound-compra.spec.md) | Corregir | BLM-0081 | `insert` no verifica el resultado del retiro: se puede crear un bono sin cobro. Sin transacción ni idempotencia. |
| B8 | [spec 04](specs/04-bond-bound-compra.spec.md) | Corregir (contador atómico); se mantiene contar los borrados en la siembra | BLM-0079 | La clave se arma con `count(bondTypeId) + 1`: hay carrera bajo compras concurrentes, sin índice único, y cuenta los borrados. |
| B9 | [spec 04](specs/04-bond-bound-compra.spec.md) | Corregir | BLM-0081 | No se valida `isPrivateBond`/`authorizedUserIds` ni `saleEndDate`, ni se hace null-check de la cuenta y el producto (crash). |
| B13 | [spec 04](specs/04-bond-bound-compra.spec.md) | Corregir | BLM-0080 | `maturityDate` usa `format('YYYY-MM-DD hh:mm:ss')` (12 h, sin am/pm): las compras de la tarde vencen 12 h antes. |
| B18 | [spec 04](specs/04-bond-bound-compra.spec.md) | Corregir | BLM-0082 | `bulkPurchase` busca la cuenta solo por `agencyId` y retira con `userId: account.userId`: se puede comprar debitando la cuenta de otro usuario de la misma agencia. |
| B19 | [spec 04](specs/04-bond-bound-compra.spec.md) | Corregir | BLM-0082 | La compra masiva crea los bonos y retiros uno por uno, y si uno falla quedan compras parciales; además crashea si un retiro devuelve error (`withdrawResponse.data` es undefined). |
| B20 | [spec 04](specs/04-bond-bound-compra.spec.md) | Corregir | BLM-0081 | El retiro de la compra individual no tiene `comment`. |
| B1 | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Corregir | BLM-0090 | `preSale` compara `totalDaysPassed` (días) con `maturityDate` (ms): la rama penalizada siempre corre y en el vencimiento `profit` sale penalizado. |
| B2 | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Corregir | BLM-0094 | `setStatus` usa `bondBound.initalDate` (typo): la fecha de compra del email es "ahora". |
| B3 | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Corregir | BLM-0093 | `sale` no verifica el error de `preSale`: con la fecha de retiro inválida acredita `NaN` y cambia el estado igual. |
| B4 | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Corregir | BLM-0093 | `sale` traga las excepciones en `catch {}` y nunca responde. |
| B4b | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Corregir | BLM-0094 | "Disponible actual" del email suma el monto dos veces. |
| B5 | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Corregir | BLM-0091 | `sale` sobre un estado `'3'` o `'4'` envía `status: ''` y lanza `TypeError`. |
| B6 | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Mantener la cuenta destino y corregir el crash | BLM-0093 | Se acredita en `selectedAccountId` y no en la cuenta de compra; crashea si no hay setting. |
| B14 | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Corregir | BLM-0090 | `parseInt` en `maturityTimeInDays`, `inflectionPoint` y `maturityRewardsPercentage`. |
| B16 | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Corregir | BLM-0093 | `preSale` y `sale` no validan que el bono sea del usuario. `sale` acredita el valor de un bono ajeno en la cuenta de quien llama. |
| B21 | [spec 05](specs/05-bond-bound-venta-canje.spec.md) | Corregir | BLM-0094 | El email usa el rendimiento y el plazo actuales del producto, no los del bono comprado. |
| B10 | [spec 06](specs/06-bond-bound-consultas.spec.md) | Corregir | BLM-0105 | `listByUserlogged` con `skip` salta siempre 5, y `limitBondResults` reordena después de saltar. |
| B15 | [spec 06](specs/06-bond-bound-consultas.spec.md) | Corregir | BLM-0104 | `listByBondObjectId` y `getInvestorsById` devuelven el `profile` completo, con el hash del password. Tampoco filtran `isRemove` ni agencia. |
| B23 | [spec 06](specs/06-bond-bound-consultas.spec.md) | Corregir | BLM-0102 | `aMonth` y `aYear` usan `$month`/`$year` en UTC: una compra del 31-oct a las 22:00 (UTC-5) cae en noviembre. |
| B24 | [spec 06](specs/06-bond-bound-consultas.spec.md) | Corregir | BLM-0103 | La búsqueda usa el texto del usuario como `$regex` sin escapar (regex inválida que hace fallar la agregación, o ReDoS). |
| B25 | [spec 06](specs/06-bond-bound-consultas.spec.md) | Corregir | BLM-0103 | `getBondPurchasedForChart` formatea con `%G` (año ISO por semana): las fechas de fines de diciembre salen con el año siguiente. También usa UTC. |
| B26 | [spec 06](specs/06-bond-bound-consultas.spec.md) | Corregir | BLM-0105 | `get`, `getDetails`, `getInvestorsById` y `getBondPurchasedForChart` no validan dueño ni permisos: cualquier sesión lee bonos ajenos. |
| B27 | [spec 06](specs/06-bond-bound-consultas.spec.md) | Corregir | BLM-0105 | `getDetails` devuelve el `bondObject` completo, incluido `authorizedUserIds`. |
| B11 | [spec 07](specs/07-reportes-y-kpis.spec.md) | Corregir | BLM-0110 | `reportSummaryPurchasedBonds` lee `data.data.statusId` (crashea sin `data.data`) y `countByStatusId` exige `data.query`. |
| B12 | [spec 07](specs/07-reportes-y-kpis.spec.md) | Corregir | BLM-0111 | El `betweenDates` del KPI compara año, mes y día por separado: del 25-ene al 5-feb no incluye el 30-ene. Además deriva `utcDiff` de la zona del servidor o del cliente. |
| B34 | [spec 07](specs/07-reportes-y-kpis.spec.md) | Corregir | BLM-0111 | Búsqueda con `$regex` sin escapar en los reportes y el KPI (mismo problema que B24 del listado). |
| B29 | [spec 07](specs/07-reportes-y-kpis.spec.md) | Corregir | BLM-0111 | Los filtros `aDate` y `betweenDates` del reporte ignoran `utcDiff` (zona del servidor), y `month`/`year` salen de `initialDate` en UTC. |
| B30 | [spec 07](specs/07-reportes-y-kpis.spec.md) | Corregir | BLM-0112 | Si falla ms-exporter, se responde `code -1000` con HTTP 200 e incluye el objeto de error interno (`err`). |
| B31 | [spec 07](specs/07-reportes-y-kpis.spec.md) | Corregir | BLM-0113 | El KPI sin permisos permite que cualquier sesión consulte totales de la agencia o de otro usuario. |
| H1 | [spec 08](specs/08-bond-history.spec.md) | Corregir (se descarta la publicación y se usan los endpoints nuevos) | BLM-0063 | La publicación `bondHistories` filtra por `bondId`, que no existe, y nunca devuelve datos. |
| H2 | [spec 08](specs/08-bond-history.spec.md) | Corregir | BLM-0036 | `BOND_SOLD` y `BOND_BOUND_STATUS_CHANGED` no existen en `Text` y se muestran vacíos. |
| H3 | [spec 08](specs/08-bond-history.spec.md) | Corregir | BLM-0037 | Una consulta de `profile` por registro (N+1). |
| H4 | [spec 08](specs/08-bond-history.spec.md) | Corregir | BLM-0063 | `listByType` con un `type` desconocido lanza `TypeError`. |
| H5 | [spec 08](specs/08-bond-history.spec.md) | Corregir | BLM-0063, BLM-0099 | `listByType` no valida permisos: cualquier sesión lee el historial de cualquier producto o bono de su agencia. |
| U1 | [spec 09](specs/09-bond-user-setting.spec.md) | Corregir (upsert) | BLM-0068 | `setSelectedAccount`, `setSelectedBondBound` y `setLanguage` lanzan `TypeError` si no existe el setting. |
| U2 | [spec 09](specs/09-bond-user-setting.spec.md) | Corregir | BLM-0069 | `bondUserSetting.list` no filtra por agencia ni por permisos. |
| U3 | [spec 09](specs/09-bond-user-setting.spec.md) | Corregir | BLM-0065, BLM-0067 | La autocreación de `getByUserLogged` no es atómica (duplicados) y no filtra `isRemove`. |
| U4 | [spec 09](specs/09-bond-user-setting.spec.md) | Corregir | BLM-0067 | `user` devuelve el perfil completo (sin `password`, pero con el resto de los campos), y `account` el documento entero. |
| U6 | [spec 09](specs/09-bond-user-setting.spec.md) | Corregir | BLM-0068 | `setSelectedAccount` acepta `data.userId`: cualquier sesión puede cambiar la cuenta destino de **otro** usuario y desviar a su propia cuenta el dinero de las ventas y canjes de la víctima. |
| U7 | [spec 09](specs/09-bond-user-setting.spec.md) | Corregir | BLM-0068 | `setSelectedAccount` no valida que la cuenta sea del usuario. |
| B22 | [spec 10](specs/10-maduracion-cron.spec.md) | Corregir (solo `'1'` y sin `soldDate`/`tookDate`) | BLM-0118 | La regla legacy (`checkMaturity`) maduraba también bonos **vendidos** (`'5'`) y desligados (`'4'`). Un bono vendido volvía a `'2'` y se podía canjear de nuevo: es el doble cobro que audita `B_BondBound_Audit_doubleProcess`. |
| B32 | [spec 10](specs/10-maduracion-cron.spec.md) | Corregir | BLM-0121 | `SyncedCron.start()` solo corre en `Cron.init()`, que está comentado en `main.js`. La maduración solo arranca después de editar un cronjob, y depende de un servicio externo que no está versionado en el workspace. |
| B33 | [spec 10](specs/10-maduracion-cron.spec.md) | Corregir (prevención) | BLM-0119 | Con 3 réplicas sin lock, la maduración y los avisos se ejecutarían tres veces. |

## 6. Regenerar y validar

El script lee las tablas *Tareas*, *Bugs legacy*, los `RF` y los `CA` de cada spec. Escribe este archivo y falla (código 1, con la lista de problemas) si encuentra:
- un ID duplicado o una fase inexistente;
- una dependencia o un RF/CA inexistente;
- un RF sin tarea `impl`, sin tarea `test` o sin CA;
- un CA que ningún test verifica;
- una dependencia hacia una fase posterior;
- un bug corregido sin tarea, o un bug repetido en dos specs.

```bash
python3 docs/migracion-blooming/tools/gen_tasks.py docs/migracion-blooming
```

Última generación: 11 specs, 96 RF, 158 CA, 146 tareas, 63 bugs. Validación sin errores. Siguiente ID libre: `BLM-0140`.
