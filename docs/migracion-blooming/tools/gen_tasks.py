#!/usr/bin/env python3
"""Genera docs/migracion-blooming/tasks.md a partir de las specs y valida la trazabilidad."""
import re
import sys
from pathlib import Path

DOCS = Path(sys.argv[1])
SPECS = sorted((DOCS / 'specs').glob('*.spec.md'))
OUT = DOCS / 'tasks.md'

RF_DEF = re.compile(r'^- \*\*(RF-\d\d\.\d+)\*\*')
CA_DEF = re.compile(r'^- \*\*(CA-\d\d\.\d+)\*\* \(([^)]*)\)')
ID_RE = re.compile(r'(RF-\d\d\.\d+|CA-\d\d\.\d+)')
TASK_RE = re.compile(r'BLM-\d{4}|T-PREP\.\d+')  # F0 conserva T-PREP.x; el resto usa BLM-xxxx


def cells(line):
    return [c.strip() for c in line.strip().strip('|').split('|')]


specs = {}
tasks = {}
PHASE_OF = {}
dup_ids = []
rfs = {}
cas = {}
bugs = []

for path in SPECS:
    num = path.name[:2]
    text = path.read_text()
    title = text.splitlines()[0].lstrip('# ').strip()  # archivo: NN-<nombre>.spec.md
    specs[num] = {'file': path.name, 'title': title}
    section = None
    for line in text.splitlines():
        if line.startswith('## '):
            section = line
        m = RF_DEF.match(line)
        if m:
            rfs[m.group(1)] = num
        m = CA_DEF.match(line)
        if m:
            cas[m.group(1)] = ID_RE.findall(m.group(2))
        if line.startswith('| BLM-') and section and 'Tareas' in section:
            c = cells(line)
            if c[0] in tasks:
                dup_ids.append(c[0])
            tasks[c[0]] = {
                'id': c[0], 'title': c[1], 'type': c[2], 'covers': ID_RE.findall(c[4]),
                'deps': TASK_RE.findall(c[5]), 'files': c[6], 'spec': num,
            }
            PHASE_OF[c[0]] = c[3]
        if section and 'Bugs legacy' in section and line.startswith('| ') and not line.startswith('| Bug') and not line.startswith('|---'):
            c = cells(line)
            bugs.append({'bug': c[0], 'desc': c[1], 'decision': c[2], 'tasks': TASK_RE.findall(c[4]), 'spec': num})

# Tareas que no salen de una spec funcional (preparación, calidad, corte): (fase, ID, tipo, título, cubre, depende de, archivos)
EXTRA = [
    ('F0', 'T-PREP.1', 'ops', 'Inicializar el repositorio: `git init`, remoto `github.com/cc-santiago-alvarez/ms-blooming-go` y rama `master`. Los líderes de área lo llevan después a `futurapps`', [], [], '`.git/`'),
    ('F0', 'T-PREP.2', 'ops', '`.gitignore` (`.claude/`, `.env*` salvo `.env.example`, `tmp/`). `.github/workflows/branch-flow.yml` lo agrega DevOps al integrar el repositorio en `futurapps`', [], ['T-PREP.1'], '`.gitignore`'),
    ('F0', 'T-PREP.3', 'ops', '`go.mod` (`module github.com/futurapps/ms-blooming-go`, `go 1.27.0`); eliminar `internal/database/mongo/connection.go`; `cmd/healthcheck` con `package main`; `go build ./...` compila', [], ['T-PREP.1'], '`go.mod`, `go.sum`, `cmd/healthcheck/main.go`, `internal/database/`'),
    ('F0', 'T-PREP.4', 'ops', '`.air.toml` para desarrollo con hot reload (sin `Makefile`)', [], ['T-PREP.3'], '`.air.toml`'),
    ('F0', 'T-PREP.5', 'ops', '`README.md` y `CLAUDE.md`: arquitectura, comandos, variables y enlace a `docs/constitution.md`', [], ['T-PREP.3'], '`README.md`, `CLAUDE.md`'),
    ('F0', 'T-PREP.6', 'ops', 'Mongo local con replica set de un nodo (necesario para transacciones) para desarrollo y tests de integración: contenedor `atlas` (`mongodb/atlas`), conexión por las variables `FA_MONGO_*`', [], ['T-PREP.3'], '`README.md` (instrucciones)'),
    ('F0', 'T-PREP.7', 'ops', 'Dump anonimizado de testing de `bondType`, `bondObject`, `bondBound`, `bondHistory`, `bondUserSetting`, `balanceAccount` y `profile` para paridad e integración', [], [], 'fuera del repo (almacenamiento del equipo)'),
    ('F8', 'BLM-0125', 'test', 'Arnés de paridad: ejecutar los métodos Meteor (DDP, backend en `origin/main`), los feeds de ms-feed y los endpoints Go sobre el mismo dump y comparar las fórmulas (`getDataChart`, `preSale`, `/bound/:id/feed`, derivados del listado)', ['RF-03.1', 'RF-03.6', 'RF-05.2', 'RF-06.11'], ['T-PREP.7', 'BLM-0031', 'BLM-0096', 'BLM-0107', 'BLM-0144'], '`cmd/tools/parity/`'),
    ('F8', 'BLM-0126', 'test', 'Suite end-to-end de flujos completos: compra → venta; compra → maduración → canje; compra masiva → listado → reporte', ['RF-04.1', 'RF-04.2', 'RF-05.3', 'RF-10.3', 'RF-07.1'], ['BLM-0088', 'BLM-0098', 'BLM-0124', 'BLM-0116'], '`test/e2e/`'),
    ('F8', 'BLM-0127', 'test', 'Matriz de autorización rol × endpoint (usuario, dueño, admin, otra agencia) ejecutada sobre todas las rutas registradas', ['RF-00.5', 'RF-04.1', 'RF-05.3', 'RF-06.6', 'RF-09.3'], ['BLM-0106', 'BLM-0071', 'BLM-0095', 'BLM-0158', 'BLM-0160', 'BLM-0162'], '`test/authz/`'),
    ('F8', 'BLM-0128', 'test', 'Prueba de carga: 50 compras concurrentes sobre la misma cuenta y p95 del listado de administración con 100k bonos', ['RF-04.5', 'RF-06.1'], ['BLM-0088', 'BLM-0109'], '`test/load/`'),
    ('F9', 'BLM-0133', 'ops', 'Auditoría de datos previa al corte: claves duplicadas (B8), settings duplicados (U3), numéricos guardados como string, bonos `\'2\'`/`\'6\'` con `soldDate` (B22), bonos `\'1\'` vencidos que ms-feed nunca maduró (B36) y que Go madura en su primera corrida, y bondObjects con `isPrivateBond` que no es booleano (O11), para normalizarlos antes del corte', ['RF-02.9', 'RF-04.3', 'RF-09.1', 'RF-10.3'], ['T-PREP.7', 'BLM-0079', 'BLM-0065'], '`cmd/tools/auditdata/`, `docs/migracion-blooming/auditoria-datos-corte.md` (informe), `docs/migracion-blooming/README.md`'),
    ('F9', 'BLM-0134', 'ops', 'Corte de la maduración en un mismo deploy: quitar el job `BondBoundCheckMaturity` de Meteor (deja de llamar a `PUT /bound/check-maturity` de ms-feed) y configurar `MS_BLOOMING_URL` en ms-cronjob-go, vacío hasta ese momento', ['RF-10.1', 'RF-10.3'], ['BLM-0124', 'BLM-0140', 'BLM-0141', 'BLM-0142', 'BLM-0149', 'BLM-0153', 'BLM-0133'], '`backend/server/lib/crons.js`, `.env` de ms-cronjob-go'),
    ('F9', 'BLM-0135', 'ops', 'Corte de compra y venta: correr la siembra de contadores (BLM-0079), habilitar los endpoints Go y deshabilitar `bondBound.purchase`, `bulkPurchase`, `preSale` y `sale` en Meteor **en el mismo deploy**', ['RF-04.3', 'RF-04.1', 'RF-05.3'], ['BLM-0079', 'BLM-0098', 'BLM-0133', 'BLM-0151'], '`backend/server/model/bond.bound.js`'),
    ('F9', 'BLM-0136', 'ops', 'Migrar el front de DDP a REST por funcionalidad (catálogo → preferencias e historial → compra y venta → consultas y reportes), con la guía de cambios de contrato de cada spec. Las gráficas dejan de leer ms-feed desde el navegador: `/bond/:id/feed` pasa a `GET /bond-objects/{id}/chart?simulation=true` y `/bound/:id/feed` a `GET /bond-bounds/{id}/chart`', ['RF-01.1', 'RF-02.9', 'RF-03.4', 'RF-03.7', 'RF-09.1', 'RF-04.1', 'RF-06.4'], ['BLM-0146'], 'repo del front'),
    ('F9', 'BLM-0137', 'ops', 'Retirar `bond.*.js` de Meteor y la constante `MS_FINOBONDS_FEED_URL` (variable `MS_FEED_URL`), una vez desplegado BLM-0072', [], ['BLM-0134', 'BLM-0135', 'BLM-0136', 'BLM-0072'], '`backend/server/model/bond.*.js`, `backend/server/lib/_constants.js`, `.env.example`'),
    ('F9', 'BLM-0152', 'ops', 'Retirar ms-feed: con la maduración en ms-cronjob-go (BLM-0134) y el front sin llamar a los feeds (BLM-0136), bajar los stacks `ms-feed` y `testing-ms-feed` y archivar el repo', ['RF-03.7', 'RF-10.1'], ['BLM-0134', 'BLM-0136'], 'servidores de ms-feed, repo `futurapps/ms-feed`'),
    ('F9', 'BLM-0138', 'ops', 'Monitoreo durante dos semanas: errores 5xx, `B_BondBound_Audit_doubleProcess` en el gateway y conciliación de `balanceAccount.available` contra la suma de `balanceDetail`', ['RF-05.5', 'RF-10.3'], ['BLM-0135'], 'dashboards y logs'),
]

PHASES = [
    ('F0', 'Preparación del repositorio', 'El esqueleto compila, está versionado y tiene entorno de pruebas con replica set.'),
    ('F1', 'Plataforma base (spec 00)', 'El servicio arranca, valida `x-secret` y la sesión de ms-oauth, y los ports compartidos (saldo, notificaciones, exportador) tienen tests.'),
    ('F2', 'Historial y tipos compartidos (specs 08 y 03)', 'Existe `BondHistoryPort.Add` y están listas las funciones puras de proyección que usan el catálogo y los bonos.'),
    ('F3', 'Catálogo (specs 01, 02 y 03)', 'Tipos y productos se administran y consultan por REST, con historial, proyección y permisos.'),
    ('F4', 'Preferencias del inversionista (spec 09)', 'El setting se autocrea, la cuenta destino está protegida (U6) y existe el endpoint interno para Meteor.'),
    ('F5', 'Compra, venta y canje (specs 04 y 05, más 03 y 08 sobre bondBound)', 'Compra individual y masiva, cotización, venta y canje son transaccionales y no hay doble cobro.'),
    ('F6', 'Consultas, reportes y KPIs (specs 06 y 07)', 'Todas las lecturas de bonos, reportes Excel y KPI responden con permisos y filtros de fecha correctos.'),
    ('F7', 'Maduración (spec 10)', 'La maduración pasa solo bonos `\'1\'`, una sola réplica a la vez, con avisos agrupados.'),
    ('F8', 'Calidad transversal', 'Paridad numérica con el legado comprobada y matriz de autorización en verde.'),
    ('F9', 'Corte y retiro del legado', 'El front usa solo REST, Meteor no tiene código de bonos, ms-feed está apagado y el monitoreo no muestra incidencias.'),
]

for phase, tid, type_, title, covers, deps, files in EXTRA:
    if tid in tasks:
        dup_ids.append(tid)
    tasks[tid] = {'id': tid, 'title': title, 'type': type_,
                  'covers': covers, 'deps': deps, 'files': files, 'spec': None}
    PHASE_OF[tid] = phase

# ---------- validaciones ----------
errors = [f'{tid}: ID duplicado' for tid in dup_ids]
for tid, t in tasks.items():
    for d in t['deps']:
        if d not in tasks:
            errors.append(f'{tid}: dependencia inexistente {d}')
    for c in t['covers']:
        if c.startswith('RF') and c not in rfs:
            errors.append(f'{tid}: RF inexistente {c}')
        if c.startswith('CA') and c not in cas:
            errors.append(f'{tid}: CA inexistente {c}')
    if t['spec'] and not t['covers']:
        errors.append(f'{tid}: tarea funcional sin RF/CA')
    if t['spec'] and t['type'] not in ('impl', 'test'):
        errors.append(f'{tid}: tipo inválido {t["type"]}')
    if t['spec'] and t['type'] == 'test' and not any(c.startswith('CA') for c in t['covers']):
        errors.append(f'{tid}: tarea de test sin CA')
    if PHASE_OF.get(tid) not in [p[0] for p in PHASES]:
        errors.append(f'{tid}: fase inválida {PHASE_OF.get(tid)!r}')

order = [p[0] for p in PHASES]
for tid, t in tasks.items():
    for d in t['deps']:
        if d in PHASE_OF and order.index(PHASE_OF[d]) > order.index(PHASE_OF[tid]):
            errors.append(f'{tid} ({PHASE_OF[tid]}) depende de {d} ({PHASE_OF[d]}), que es de una fase posterior')

for ca, refs in cas.items():
    for r in refs:
        if r not in rfs:
            errors.append(f'{ca}: referencia a RF inexistente {r}')

rf_impl = {rf: [] for rf in rfs}
rf_test = {rf: [] for rf in rfs}
rf_ca = {rf: [] for rf in rfs}
rf_ops = {rf: [] for rf in rfs}
for ca, refs in cas.items():
    for r in refs:
        if r in rf_ca:
            rf_ca[r].append(ca)
for tid, t in tasks.items():
    for c in t['covers']:
        if c.startswith('RF') and c in rfs:
            if t['type'] == 'impl':
                rf_impl[c].append(tid)
            elif t['type'] == 'test':
                rf_test[c].append(tid)
            else:
                rf_ops[c].append(tid)
        if c.startswith('CA') and c in cas and t['type'] == 'test':
            for r in cas[c]:
                if r in rf_test and tid not in rf_test[r]:
                    rf_test[r].append(tid)

for rf in rfs:
    if not rf_impl[rf]:
        errors.append(f'{rf}: sin tarea de implementación')
    if not rf_test[rf]:
        errors.append(f'{rf}: sin tarea de test')
    if not rf_ca[rf]:
        errors.append(f'{rf}: sin criterio de aceptación')

tested_cas = {c for t in tasks.values() if t['type'] == 'test' for c in t['covers'] if c.startswith('CA')}
for ca in cas:
    if ca not in tested_cas:
        errors.append(f'{ca}: ningún test lo verifica')

for b in bugs:
    for tid in b['tasks']:
        if tid not in tasks:
            errors.append(f'Bug {b["bug"]}: tarea inexistente {tid}')
    if b['decision'].lower().startswith('corregir') and not b['tasks']:
        errors.append(f'Bug {b["bug"]}: se corrige pero no tiene tarea')

seen = {}
for b in bugs:
    seen.setdefault(b['bug'], []).append(b['spec'])
for bug, sp in seen.items():
    if len(sp) > 1:
        errors.append(f'Bug {bug} aparece en varias specs: {sp}')

if errors:
    print('\n'.join(errors))
    sys.exit(1)


# ---------- render ----------
def sort_key(tid):
    if tid.startswith('T-PREP.'):
        return (0, int(tid[7:]))
    return (1, int(tid[4:]))


def topo_order(ids):
    """Orden topológico dentro de la fase; a igualdad, por ID."""
    pending = sorted(ids, key=sort_key)
    done, result = set(), []
    while pending:
        for tid in pending:
            if all(d in done or d not in ids for d in tasks[tid]['deps']):
                result.append(tid)
                done.add(tid)
                pending.remove(tid)
                break
        else:
            raise SystemExit('ciclo de dependencias en: ' + ', '.join(pending))
    return result


def ready_at_phase_start(tid):
    ph = PHASE_OF[tid]
    return all(PHASE_OF[d] != ph for d in tasks[tid]['deps'])


def spec_link(num):
    return f'[spec {num}](specs/{specs[num]["file"]})'


out = []
w = out.append
w('# Tareas — Migración de Blooming a ms-blooming-go')
w('')
w('> **Archivo generado.** La fuente de verdad de cada tarea es la tabla *Tareas* de su spec. Este archivo se regenera con el script de §6 y no se edita a mano. Las tareas de preparación, calidad y corte no salen de una spec funcional y se definen en el script (`EXTRA`).')
w('')
w('## 1. Cómo leer este documento')
w('')
w('- **ID:** `BLM-xxxx`, numeración global de 4 dígitos; es también el ticket del commit (`feat: [BLM-0042] impl …`). Una tarea nueva toma el siguiente ID libre (ver §6) y no se renumeran las existentes. Excepción: las tareas de F0 conservan su formato original `T-PREP.x`.')
w('- **Fase:** la define la columna *Fase* de la tabla *Tareas* de cada spec; la spec de cada tarea se ve en el enlace al inicio de su fase.')
w('- **Formato:** `- [ ] ID [P] tipo título`, seguido de una línea con lo que cubre y sus dependencias, y otra con los archivos.')
w('- **`[P]`:** la tarea puede empezar al inicio de su fase, porque no depende de otra tarea de la misma fase. Las tareas `[P]` de una fase se pueden repartir en paralelo.')
w('- **Cubre:** requisitos (`RF-NN.x`) que implementa, o criterios de aceptación (`CA-NN.x`) que verifica.')
w('- **Tipo:** `impl` (código de producción), `test` u `ops` (repositorio, corte).')
w('- **Reglas** (constitución §14):')
w('  - toda tarea funcional cita al menos un RF o CA;')
w('  - todo RF tiene al menos una tarea `impl` y una `test`;')
w('  - un cambio de comportamiento se hace primero en la spec y después aquí (regenerando);')
w('  - los comentarios del código no citan tareas, specs, RF ni CA; solo cada test cita en un comentario el CA que verifica.')
w('- **PR:** cada PR cita sus `BLM-…`, sus `RF-…` y su spec.')
w('')
w('## 2. Resumen')
w('')
w('| Fase | Nombre | Tareas | impl | test | ops |')
w('|---|---|---|---|---|---|')
for code, name, _ in PHASES:
    ids = [t for t in tasks if PHASE_OF[t] == code]
    cnt = lambda ty: sum(1 for t in ids if tasks[t]['type'] == ty)
    w(f'| {code} | {name} | {len(ids)} | {cnt("impl")} | {cnt("test")} | {cnt("ops")} |')
w(f'| | **Total** | **{len(tasks)}** | **{sum(1 for t in tasks.values() if t["type"]=="impl")}** | **{sum(1 for t in tasks.values() if t["type"]=="test")}** | **{sum(1 for t in tasks.values() if t["type"]=="ops")}** |')
w('')
w('```')
w('F0 ─▶ F1 ─▶ F2 ─▶ F3 ─▶ F4 ─▶ F5 ─▶ F6 ─▶ F7 ─▶ F8 ─▶ F9')
w('```')
w('')
w('F9 (corte) requiere ms-blooming-go desplegado en testing y live. El despliegue lo hace DevOps y no forma parte de este plan (ver el [README de la migración](README.md), §7).')
w('')
w('## 3. Tareas por fase')
w('')
for code, name, exit_ in PHASES:
    ids = topo_order([t for t in tasks if PHASE_OF[t] == code])
    w(f'### {code} — {name}')
    w('')
    specs_in = sorted({tasks[t]['spec'] for t in ids if tasks[t]['spec']})
    if specs_in:
        w('Specs: ' + ', '.join(spec_link(s) for s in specs_in) + '.')
        w('')
    for tid in ids:
        t = tasks[tid]
        p = ' [P]' if ready_at_phase_start(tid) else ''
        covers = ', '.join(t['covers']) if t['covers'] else '—'
        deps = ', '.join(t['deps']) if t['deps'] else '—'
        w(f'- [ ] **{tid}**{p} `{t["type"]}` {t["title"]}')
        w(f'  - cubre: {covers} · depende de: {deps}')
        w(f'  - archivos: {t["files"]}')
    w('')
    w(f'**Criterio de salida {code}:** {exit_}')
    w('')
    if code not in ('F0', 'F9'):
        w(f'- [ ] `go vet ./...` y `go test ./...` en verde; `go test -tags=integration ./...` en verde contra el replica set local.')
        w('')

w('## 4. Matriz de trazabilidad: requisito → tareas')
w('')
w('| Spec | RF | Implementan | Verifican (tests) | CA | Operación / corte |')
w('|---|---|---|---|---|---|')
for rf in sorted(rfs, key=lambda r: (r[3:5], int(r.split('.')[1]))):
    num = rfs[rf]
    w(f'| {num} | {rf} | {", ".join(sorted(rf_impl[rf], key=sort_key))} | {", ".join(sorted(rf_test[rf], key=sort_key))} | {", ".join(sorted(rf_ca[rf], key=lambda c: int(c.split(".")[1])))} | {", ".join(sorted(rf_ops[rf], key=sort_key)) or "—"} |')
w('')
w('## 5. Bugs del legado → tareas')
w('')
w('| Bug | Spec | Decisión | Tarea | Descripción |')
w('|---|---|---|---|---|')
for b in bugs:
    w(f'| {b["bug"]} | {spec_link(b["spec"])} | {b["decision"]} | {", ".join(b["tasks"]) or "—"} | {b["desc"]} |')
w('')
w('## 6. Regenerar y validar')
w('')
w('El script lee las tablas *Tareas*, *Bugs legacy*, los `RF` y los `CA` de cada spec. Escribe este archivo y falla (código 1, con la lista de problemas) si encuentra:')
w('- un ID duplicado o una fase inexistente;')
w('- una dependencia o un RF/CA inexistente;')
w('- un RF sin tarea `impl`, sin tarea `test` o sin CA;')
w('- un CA que ningún test verifica;')
w('- una dependencia hacia una fase posterior;')
w('- un bug corregido sin tarea, o un bug repetido en dos specs.')
w('')
w('```bash')
w('python3 docs/migracion-blooming/tools/gen_tasks.py docs/migracion-blooming')
w('```')
w('')
w(f'Última generación: {len(specs)} specs, {len(rfs)} RF, {len(cas)} CA, {len(tasks)} tareas, {len(bugs)} bugs. Validación sin errores. Siguiente ID libre: `BLM-{max(sort_key(t)[1] for t in tasks if t.startswith("BLM-")) + 1:04d}`.')

OUT.write_text('\n'.join(out) + '\n')
print(f'OK: {len(specs)} specs, {len(rfs)} RF, {len(cas)} CA, {len(tasks)} tareas, {len(bugs)} bugs')
