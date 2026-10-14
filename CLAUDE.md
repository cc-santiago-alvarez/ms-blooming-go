# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`ms-blooming-go` is the Go microservice for **Blooming** (bonds) in Futurapps. It replaces the bond DDP
methods/publications of the Meteor backend (`backend/server/model/bond.*.js`) with a REST API the front
calls directly. Built on `codecraftkit/nexus` (HTTP) and `codecraftkit/mongocc` (MongoDB, driver v2).
Reference architecture: `ms-petty-cash-api` — anything not specified here or in the constitution is
resolved the way petty-cash does it.

**`docs/constitution.md` is the source of non-negotiable rules** (stack, architecture, naming, errors,
validation, auth, data, money/dates, transactions, security, testing, ops). Read it before writing code;
do not restate or contradict it. If a rule gets in the way, the constitution is changed first.

## Spec-driven process

- No code without an approved spec in `docs/migracion-blooming/specs/NN-<name>.spec.md`.
- Every task cites the `RF-NN.x` / `CA-NN.x` it implements or verifies; tests cite their `CA-NN.x`.
- Tasks are identified as `BLM-xxxx` (global 4-digit number, also the commit ticket); F0 tasks keep their original `T-PREP.x` IDs. Each spec's *Tareas*
  table declares the task's *Fase*; a new task takes the next free ID reported at the bottom of `tasks.md`.
  Existing IDs are never renumbered or reused.
- `docs/migracion-blooming/tasks.md` is **generated** — never edit it by hand. Task tables live in each
  spec; prep/QA/infra/cutover tasks live in `EXTRA` in the script. Regenerate and validate with:
  `python3 docs/migracion-blooming/tools/gen_tasks.py docs/migracion-blooming`
- When behavior changes: spec first, then its task table (and regenerate `tasks.md`), then code.

## Commands

```bash
air                               # hot reload (builds ./cmd -> ./tmp/server)
go build ./...
go vet ./...
go test ./...                     # unit tests, no infrastructure
go test -tags=integration ./...   # integration tests against local Mongo (replica set)
go test ./internal/core/<domain>/... -run TestName
```

Environment is loaded from `.env` **and then** `.env.$ENV`, where `ENV` is required in `.env` (`development`,
`testing` or `production`, no default); **both files must
exist** or the process `log.Fatal`s. `.env.example` is the template; the full variable list is in spec 00 §5.

Local Mongo is the `atlas` container (`mongodb/atlas`, port 27017, single-node replica set — required for
transactions). Start it with `docker start atlas`; the connection comes from `FA_MONGO_URL`/`FA_MONGO_DBNAME`.

## Architecture

Hexagonal, one directory per domain (`bondType`, `bondObject`, `bondBound`, `bondHistory`,
`bondUserSetting`, plus `session` and `shared`). Layout and dependency rules: constitution §2.

- `cmd/main.go` — API entrypoint (`server.InitServer().Run()`, pending BLM-0002).
- `cmd/healthcheck/main.go` — standalone TCP probe of `GET /_health` (`package main`).
- `cmd/api/config/` — env loading into `ApiEnv`.
- `cmd/api/server/` — composition root: `server.go`, `server.config.go`, `server.deps.go`
  (`serverDeps` container), `server.<domain>.go` with `init<Domain>Domain` + `register<Domain>Handlers`.
  Initialization order matters: a domain is initialized before any domain that uses its `ApplicationPorts`.
- `cmd/api/handlers/<domain>/` — one file per endpoint; decode, call the use case, respond.
- `internal/core/<domain>/` — `domain/`, `ports/`, `application/` (one use case per file),
  `infrastructure/mongocc/` (one file per repository method + `EnsureIndexes`).
- `internal/cron/` — scheduler and jobs (bond maturity, with a distributed lock).

## Conventions

- File/package naming and explicit import aliases (`bondBound_domain`, `bondBound_application`, …):
  constitution §3.
- Comments, logs, docs and commit messages are in Spanish.
- Legacy collections and field names are kept as-is; `_id` is always a `string` (UUID v4 for new docs,
  17-char Meteor ids for legacy ones). See constitution §7.

## Branch flow

- **Temporary:** the repo lives at `github.com/cc-santiago-alvarez/ms-blooming-go` with a single
  `master` branch. Area leads will move it to `futurapps`; DevOps then adds `branch-flow.yml` and the deploy
  infra, and the `cc-* ↔ integrations → main` flow from constitution §13 applies.
- The Go module path is already the final one: `github.com/futurapps/ms-blooming-go`.
- **`.claude/` is never versioned** (it is in `.gitignore`); never `git add -f` anything under it.
  `CLAUDE.md` itself is versioned.
- Commit format: `[BLM-xxxx] [MS-BLOOMING][ÁREA] descripción en español` (the task ID). The user makes the commits.
