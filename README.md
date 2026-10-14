# ms-blooming-go

Microservicio Go del módulo **Blooming** (bonos) de Futurapps. Reemplaza los métodos y publicaciones DDP de bonos del backend Meteor (`backend/server/model/bond.*.js`) por una API REST que el front consume directo.

- Pautas de código no negociables: [`docs/constitution.md`](docs/constitution.md).
- Specs, tareas y trazabilidad de la migración: [`docs/migracion-blooming/`](docs/migracion-blooming/README.md).

## Stack

Go 1.27, [`codecraftkit/nexus`](https://github.com/codecraftkit/nexus) (HTTP) y [`codecraftkit/mongocc`](https://github.com/codecraftkit/mongocc) (MongoDB, driver v2). Arquitectura de referencia: `ms-petty-cash-api`.

## Arquitectura

Hexagonal, un directorio por dominio. El detalle y las reglas están en la [constitución §2](docs/constitution.md#2-arquitectura-hexagonal).

```
cmd/
  main.go                 -> entrypoint del API
  healthcheck/main.go     -> binario de healthcheck (GET /_health por TCP)
  api/
    config/               -> carga de .env y ApiEnv
    server/               -> composition root (serverDeps, init de dominios, middlewares)
    handlers/<dominio>/   -> un archivo por endpoint
internal/
  core/<dominio>/         -> domain/, ports/, application/, infrastructure/mongocc/
  core/shared/            -> adapters, middlewares, ports y dominio compartidos
  cron/                   -> scheduler y jobs
```

## Comandos

```bash
air                                   # desarrollo con hot reload (compila ./cmd en ./tmp/server)
go build ./...
go vet ./...
go test ./...                         # unitarios, sin infraestructura
go test -tags=integration ./...       # integración contra Mongo (replica set)
```

Antes de abrir un PR, `go vet ./...` y `go test ./...` tienen que pasar.

## Variables de entorno

Se cargan de `.env` y después de `.env.<ENV>` (`ENV=development` por defecto). **Tienen que existir los dos archivos**: si falta alguno, el proceso termina al arrancar. La plantilla es [`.env.example`](.env.example) y la lista completa, con requeridas y defaults, está en la [spec 00 §5](docs/migracion-blooming/specs/00-plataforma-base.spec.md#5-configuración-envexample). Los valores se piden al equipo.

## Desarrollo local

Levantar con `air`.

## Ramas

Formato de commit: `[BLM-xxxx] [MS-BLOOMING][ÁREA] descripción en español`.
