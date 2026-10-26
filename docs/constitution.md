# Constitución de ms-blooming-go

Pautas no negociables para escribir código en `ms-blooming-go`. Toda spec, tarea y PR debe cumplirlas. Si una regla estorba, se cambia esta constitución primero (con PR propio), no se ignora.

- **DEBE** = obligatorio. **NO DEBE** = prohibido. **PUEDE** = permitido con justificación en el PR.
- Arquitectura de referencia: `ms-petty-cash-api` (hexagonal por dominio, `codecraftkit/nexus` + `codecraftkit/mongocc`). Lo que aquí no se diga, se resuelve como allí.

---

## 1. Stack

| Regla | Justificación |
|---|---|
| **DEBE** usar Go `1.27` y módulo `github.com/futurapps/ms-blooming-go`. | Mismo toolchain y convención de nombres que petty-cash y cronjob-go. |
| **DEBE** usar `codecraftkit/nexus` (HTTP), `codecraftkit/mongocc` (Mongo, driver v2), `joho/godotenv`, `rs/cors`, `google/uuid`. | Librerías ya operadas por el equipo. |
| **NO DEBE** agregar dependencias nuevas sin justificarlas en el PR (qué resuelve y por qué no alcanza la stdlib). | Mantener el binario chico y el mantenimiento acotado. |
| **NO DEBE** usar ORMs, frameworks de DI ni librerías de validación por reflexión. | Petty-cash valida a mano con `Validation()`; mismo criterio. |

## 2. Arquitectura hexagonal

```
cmd/
  main.go                         -> server.InitServer().Run()
  healthcheck/main.go             -> binario de healthcheck (package main)
  api/
    config/                       -> package api_config (env)
    server/                       -> composition root
      server.go                   -> InitServer: init de dominios en orden, handlers
      server.config.go            -> config, Mongo, nexus.Server, middlewares
      server.deps.go              -> struct serverDeps (contenedor de dependencias)
      server.<dominio>.go         -> init<Dominio>Domain(deps) + register<Dominio>Handlers(server, deps)
    handlers/<dominio>/           -> un archivo por endpoint
internal/
  core/
    architecture_test.go
    <dominio>/                    -> bondType, bondObject, bondBound, bondHistory, bondUserSetting, ...
      domain/                     -> entidades, DTOs, enums, errores, httpStatus
      ports/                      -> <D>ApplicationPorts + <D>MongoccRepositoryPorts
      application/                -> un caso de uso por archivo
      infrastructure/mongocc/     -> repositorio: un archivo por método + ensureIndexes
    session/domain/               -> UserSession + claves de contexto
    shared/                       -> adapters, middlewares, ports y dominio compartidos
```

| Regla | Justificación |
|---|---|
| **DEBE** existir un caso de uso por archivo en `application/` (`<d>.<acción>.go`). | Archivos chicos, revisión simple, trazables a un RF. |
| **DEBE** inyectarse toda dependencia por interfaz (`ports`) desde `cmd/api/server`. | Testeable con fakes y sin acoplar capas. |
| **NO DEBE** un paquete `*_domain` importar otro `*_domain` (salvo `shared_domain`). Lo verifica `internal/core/architecture_test.go`. | Dominios independientes; lo que cruza dominios vive en `application` o `ports`. |
| **NO DEBE** la capa `application` conocer `http`, `nexus` ni `mongo`. | La lógica de negocio no depende del transporte ni del motor. |
| **NO DEBE** el handler contener lógica de negocio: decodifica, llama al caso de uso y responde. | Un solo lugar para las reglas. |
| **DEBE** un caso de uso que necesite otro dominio usar su `ApplicationPorts`, no su repositorio. | Las reglas del otro dominio (soft delete, scoping) no se saltan. |

## 3. Nombres

| Elemento | Convención | Ejemplo |
|---|---|---|
| Handler | `<d>.<acción>.<método http>.go` | `bondBounds.purchase.post.go` |
| Caso de uso | `<d>.<acción>.go` | `bondBound.purchase.go` |
| Método de repositorio | `<d>.<método>.adapter.go` | `bondBound.insertOne.adapter.go` |
| Repositorio | `<d>.mongocc.repository.go` (nombre de colección + struct) | `bondBound.mongocc.repository.go` |
| Errores | `<d>.errors.go` + `<d>.httpStatus.go` | `bondBound.errors.go` |
| DTOs | `<d>.dto.domain.go` | `bondBound.dto.domain.go` |
| Enums | `<d>.status.domain.go` | `bondBound.status.domain.go` |
| Paquetes | `<d>_<capa>` importados con alias explícito | `bondBound_domain`, `bondBound_application`, `bondBound_mongocc_repository` |
| JSON | camelCase, igual a los campos Mongo legacy | `maturityTimeInDays`, `earlyCashTerm` |

- **DEBE** cada nombre de campo Go mapear 1:1 al campo Mongo legacy (`bson:"..."`). No se renombran campos persistidos.

## 4. Errores

| Regla | Justificación |
|---|---|
| **DEBE** declararse cada error de negocio como sentinel `errors.New("ERR_UPPER_SNAKE")` en `<d>.errors.go`, agrupado por status HTTP. | Contrato estable para el front. |
| **DEBE** `<d>.httpStatus.go` exponer `MapToHTTPStatus(err) int` usando `errors.Is`; lo no mapeado es 500. | Un solo mapeo por dominio. |
| **NO DEBE** usarse `switch err.Error()` ni comparar strings de error. | Petty-cash lo tiene en dominios viejos y es frágil. |
| **DEBE** todo error nuevo registrarse en los dos archivos en el mismo PR. | Sin errores que caigan a 500 por olvido. |
| **DEBE** cada sentinel que reemplaza un código legacy documentar ese código en la spec (`-2010 → ERR_ACCOUNT_WITHOUT_FUNDS`). | El front migra desde los códigos numéricos de Meteor. |
| **DEBE** responderse con `nexus.ResponseWithError(w, status, err.Error())`. | Formato de error uniforme: `{code, message, code_name, errors}`. |
| **PUEDE** un error llevar un dato para el front en `errors.detail`, con `handlers_shared.ResponseWithErrorDetail`, si su spec lo pide. El resto del cuerpo es el de `nexus.ResponseWithError`. | El front muestra el dato sin parsear `message`, que sigue siendo el código. |
| **NO DEBE** exponerse al cliente el texto de un error interno (Mongo, HTTP): se loguea y se responde `ERR_INTERNAL`. | No filtrar detalles de infraestructura. |

## 5. Validación

| Regla | Justificación |
|---|---|
| **DEBE** cada DTO de entrada tener `Normalize()` (trim, defaults) y `Validation() error` (reglas). | Mismo patrón que petty-cash, sin reflexión. |
| **DEBE** el caso de uso llamar a `Normalize()` y `Validation()`; el handler solo decodifica JSON y parsea query/path. | Las reglas viven en la capa de aplicación y se testean sin HTTP. |
| **DEBE** devolverse `400` con `BODY_REQUIRED` / `INVALID_REQUEST_BODY` cuando el JSON no decodifica. | Contrato uniforme. |
| **DEBEN** los IDs de entrada (path, query y body: `id`, `bondObjectId`, `accountId`, `userId`, …) validarse con `shared_domain.IsValidID`, que acepta **solo** dos formatos: UUID v4 (`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-4[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}$`) y Meteor (`^[A-Za-z0-9]{17}$`). Cualquier otro formato responde `400 ERR_INVALID_ID`. | Los documentos nuevos usan UUID, pero el servicio también consulta las colecciones viejas, cuyo `_id` es el `Random.id()` de Meteor (17 caracteres). En el ecosistema no hay otros formatos (`@futurapps/models` usa `uuidv4`). |

## 6. Sesión, autenticación y autorización

La sesión **la crea y la administra ms-oauth**. El front hace login contra ms-oauth (`POST /api/v1/session/register`), recibe el token y con él llama a ms-blooming-go. ms-blooming-go nunca crea, renueva ni borra sesiones.

```
Front ──login──▶ ms-oauth  (crea la sesión, devuelve token)
Front ──x-secret + x-token──▶ ms-blooming-go
                               └─ VerifySession ──GET /api/v1/session (x-token)──▶ ms-oauth
                                   ◀── UserSession (userId, agencyId, agency.isMain, countryCode, adminBlooming, ...)
                               └─ UserSession en el context ▶ handler ▶ caso de uso
```

| Regla | Justificación |
|---|---|
| **DEBE** todo endpoint (salvo `/_health`, `/_routes` e internos) pasar `ValidateApiSecret` + `VerifySession`. | Mismo modelo que petty-cash. |
| **DEBE** la identidad (`userId`, `agencyId`, flags de admin) leerse del `UserSession` del contexto (`session_domain.GetUserSession(ctx)`). | El cliente no puede suplantar a otro usuario. |
| **NO DEBE** aceptarse `userId` o `agencyId` del body o query para operar en nombre del usuario. Solo los endpoints de administración PUEDEN recibir `userId` como **filtro**. | Corrige el modelo legacy, donde cualquier sesión operaba sobre cualquier usuario. |
| **DEBE** cada endpoint de administración envolverse con `RequireBloomingAdmin` (`session.adminBlooming == true`). | En el legado ningún método validaba permisos. |
| **DEBE** cada operación sobre un `bondBound` validar que pertenece al usuario de la sesión o que el usuario es admin. | Corrige la venta de bonos ajenos (bug B16). |
| **DEBE** toda consulta filtrarse por `agencyId` de la sesión, salvo que `agency.isMain` sea `true`. | Paridad con el scoping legacy por agencia. |
| **DEBE** `ctx` propagarse por todas las capas, y sus deadlines respetarse. | Cancelación y timeouts de punta a punta. |

## 7. Datos

| Regla | Justificación |
|---|---|
| **DEBE** cada documento nuevo tener `_id` UUID v4 string, asignado en `New()` de la entidad con `uuid.New().String()`. | Mismo criterio que petty-cash. |
| **DEBEN** los documentos legacy (con `_id` Meteor de 17 caracteres) poder leerse, actualizarse y referenciarse **sin migrar ni reescribir su `_id`**. Un documento nuevo puede apuntar a uno legacy, por ejemplo un `bondBound` UUID con `bondObjectId` de 17 caracteres. | Las colecciones viejas siguen vivas y compartidas con Meteor. |
| **DEBE** el código tratar `_id` y toda referencia como `string`. **NO DEBE** convertirse a ObjectId, ni asumirse un largo o formato fijo en índices, ordenamientos o claves de caché. | Conviven IDs Meteor y UUID en las mismas colecciones. |
| **DEBE** mantenerse los nombres de colección y de campos legacy: `bondType`, `bondObject`, `bondBound`, `bondHistory`, `bondUserSetting`. | Meteor y el ms conviven sobre la misma base durante el corte. |
| **DEBE** usarse soft delete con `isRemove: true`; toda consulta filtra `isRemove: false`. | Paridad con `Collection.hide` del legado. |
| **DEBE** `New()` asignar `createdAt`, `updatedAt` (UTC) e `isRemove=false`; toda actualización setea `updatedAt`. | Paridad con el wrapper `Collection` de Meteor. |
| **DEBE** `FindOneByID` devolver `nil, nil` si no existe; el caso de uso lo traduce a `ERR_<X>_NOT_FOUND`. | Convención petty-cash. |
| **DEBE** cada repositorio exponer `EnsureIndexes(ctx)`, llamado al arrancar; un fallo solo se loguea como warning. | Índices declarados en código y versionados. |
| **DEBE** toda respuesta usar proyección explícita sobre `profile`; **NO DEBE** devolverse nunca `password` ni el perfil completo. | Corrige B15 (hash de password expuesto). |
| **NO DEBE** el ms escribir en colecciones de otros módulos salvo las listadas en la spec 00 (`balance*`, `notice`, `activity`). | Superficie de escritura acotada y auditada. |

## 8. Dinero y fechas

| Regla | Justificación |
|---|---|
| **DEBE** representarse el dinero como `float64` (así está en Mongo) y redondearse a 2 decimales **solo** al persistir montos y al responder. | Paridad con `toFixed(2)` sin acumular errores en pasos intermedios. |
| **DEBE** un único helper `shared.Round2(v float64) float64` hacer el redondeo. | Un solo criterio de redondeo. |
| **NO DEBE** truncarse porcentajes ni montos con conversiones a entero. | Corrige B14 (`parseInt` en tasas y `parseCurrency`). |
| **DEBE** persistirse toda fecha en UTC (`time.Now().UTC()`). | Sin ambigüedad de zona. |
| **DEBE** todo cálculo "por día" (días transcurridos, inicio de día, filtros por fecha) usar la zona de la sesión mediante el helper único `shared.StartOfDay(t, utcDiff)`. | Paridad con `moment().utcOffset(session.utcDiff)` del legado. |
| **DEBE** cada fórmula financiera vivir en una función pura del dominio, sin I/O, con tabla de tests. | Se puede comparar contra el legado caso por caso. |

## 9. Transacciones y efectos colaterales

| Regla | Justificación |
|---|---|
| **DEBE** toda operación que mueva saldo (compra, compra masiva, venta, canje) ejecutarse dentro de `TransactionRunnerPort.RunInTransaction`. | El legado podía crear bonos sin cobrar (B7) o cobrar parcialmente (bulk). |
| **DEBE** el callback de la transacción ser idempotente ante reintentos (`WithTransaction` puede reintentarlo). | Semántica del driver de Mongo. |
| **DEBEN** las notificaciones, emails, `notice` y `activity` ejecutarse **después** del commit, fuera del callback. | No notificar operaciones que hicieron rollback ni duplicar avisos por reintento. |
| **NO DEBE** un fallo de notificación revertir ni hacer fallar la operación de negocio; se loguea. | La notificación es best-effort, como en el legado. |

## 10. Seguridad

| Regla | Justificación |
|---|---|
| **NO DEBEN** loguearse los headers `x-secret`, `x-token` ni bodies con datos de cuenta. | Petty-cash loguea headers completos en fallos de auth; no se copia. |
| **NO DEBE** haber secretos hardcodeados ni valores de fallback para secretos; si falta un secreto requerido, el proceso termina al arrancar. | Fallar rápido en lugar de operar inseguro. |
| **DEBE** CORS leer los orígenes permitidos de `CORS_ORIGINS` (lista, con soporte de `*.dominio.com`). | El front llama directo al ms (no vía gateway). |
| **DEBE** cada número de cuenta que sale del ms enmascararse (`****************1234`). | Paridad con `getHideNumber`. |
| **DEBEN** los endpoints internos (servicio a servicio) usar `NoRequiresAuthentication` y exigir `x-secret`. | Mismo patrón que petty-cash. |

## 11. Testing

| Regla | Justificación |
|---|---|
| **DEBEN** usarse `testing` + `httptest` de la stdlib. Los fakes embeben la interfaz del port y solo implementan lo que el test usa. | Patrón de petty-cash, sin librerías de mocks. |
| **DEBE** cada caso de uso tener tests unitarios, y cada test citar en un comentario el `CA-NN.x` que verifica (no en el nombre). | Trazabilidad spec → test; es la única cita permitida en un comentario (§14). |
| **DEBE** cada fórmula financiera (preSale, gráficas, acumulados, effectiveDate) tener un test de tabla con casos calculados con el código legacy. | Paridad numérica comprobable. |
| **DEBEN** los tests de integración llevar `//go:build integration` y hacer `t.Skip` si Mongo no está disponible. Las transacciones requieren replica set. | `go test ./...` corre sin infraestructura. |
| **DEBEN** los tests de integración de cada repositorio usar fixtures con los dos formatos de `_id` (UUID y Meteor de 17 caracteres), incluidas referencias cruzadas mixtas. | Garantiza que se pueden consultar las colecciones viejas. |
| **DEBE** `go vet ./...` y `go test ./...` pasar antes de abrir el PR. | Puerta mínima de calidad. |

## 12. Operación

| Regla | Justificación |
|---|---|
| **DEBE** existir `GET /_health` (de nexus) y el binario `cmd/healthcheck` (`package main`) que lo consulte por TCP en el puerto de `API_PORT`. | El healthcheck de Swarm usa el binario. |
| **DEBE** el proceso hacer graceful shutdown con `signal.NotifyContext` (SIGINT/SIGTERM): deja de aceptar requests, espera los que están en curso y desconecta Mongo. | Petty-cash no lo tiene; en deploys rolling corta requests de compra a la mitad. |
| **DEBE** el log de requests incluir método, ruta, status y duración, y **NO DEBE** incluir headers ni body. | Diagnóstico sin datos sensibles. |
| **NO DEBE** el servicio programar jobs propios: los programa ms-cronjob-go, que llama a endpoints internos con `x-secret`. Cada job **DEBE** tomar un lock distribuido antes de ejecutarse. | Menos dependencias y un solo lugar de programación en la plataforma; el disparo puede llegar a cualquiera de las 3 réplicas o repetirse. |

## 13. Ramas y commits

- Flujo `cc-* ↔ integrations → main`. La infra (`.deploy*/`, `.github/workflows/docker-build.yml`, `.dockerignore`, `.env*` salvo `.env.example`) vive **solo en `main`**.
- `.claude/` **NO DEBE** versionarse en ninguna rama.
- Formato de commit: `[BLM-xxxx] [MS-BLOOMING][ÁREA] descripción en español`, donde `BLM-xxxx` es el ID de la tarea en `tasks.md`. Ejemplo: `[BLM-0081] [MS-BLOOMING][BOND-BOUND] compra con transacción`.

## 14. Proceso SDD

| Regla | Justificación |
|---|---|
| **NO DEBE** escribirse código sin una spec aprobada en `docs/migracion-blooming/specs/`. | La spec es el contrato. |
| **DEBEN** las specs nombrarse `NN-<nombre>.spec.md` (por ejemplo `04-bond-bound-compra.spec.md`). `NN` es el número de la spec y el prefijo de sus requisitos y criterios (`RF-04.x`, `CA-04.x`). | El generador de `tasks.md` solo lee `*.spec.md`. |
| **DEBEN** las tareas identificarse como `BLM-xxxx` (numeración global de 4 dígitos, sin prefijo de spec; las de F0 conservan su formato `T-PREP.x`) y declarar su fase en la columna *Fase* de la tabla *Tareas*. Una tarea nueva toma el siguiente ID libre que informa `tasks.md`; los IDs existentes no se renumeran ni se reutilizan. | El ID es también el ticket del commit y tiene que ser estable. |
| **DEBE** cada tarea implementar o verificar uno o más `RF-NN.x` / `CA-NN.x` de una spec, y citarlos. | Trazabilidad requisito → tarea → commit → test. |
| **DEBEN** los comentarios del código describir en pocas palabras qué hace el código. **NO DEBEN** citar tareas (`BLM-xxxx`, `T-PREP.x`), specs, requisitos (`RF-NN.x`), criterios (`CA-NN.x`), bugs (`Bxx`) ni secciones de esta constitución. Única excepción: el `CA-NN.x` en el comentario de cada test (§11). | La trazabilidad vive en la spec, `tasks.md`, los commits y los PR; un comentario con IDs queda desactualizado cuando estos cambian. |
| **DEBE**, si cambia un comportamiento, actualizarse primero la spec, luego sus tareas (en la spec y en `tasks.md`) y después el código. | La documentación no queda atrás del código. |
| **DEBE** cada PR citar en la descripción las tareas (`BLM-xxxx`), los requisitos (`RF-NN.x`) y la spec que cubre. | Revisión contra el contrato. |
| **DEBE** la tabla *Tareas* de cada spec coincidir con `tasks.md` (mismos IDs, títulos y dependencias). | Una sola fuente de verdad, vista desde dos lados. |
