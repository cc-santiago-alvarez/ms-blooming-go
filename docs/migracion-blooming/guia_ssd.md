# SDD — <nombre del feature>

```
Diseñado: YYYY-MM-DD · Caduca: YYYY-MM-DD (+60 días)
```

> El header no es decorativo: es el mecanismo de la regla 15, y `foundations status`
> cuenta los documentos que lo llevan. Un SDD sin fecha no existe para él.

> **Estado:** 📝 diseño · 🔨 en implementación · ✅ en producción · 🗄️ archivado
> **Cruza a otro equipo:** sí/no — <cuál y por qué> (regla 18). Coord: <link>

**Un archivo si lo escribís a mano.** Si lo genera el flujo de un asistente, que lo
parte en proposal / spec / design / tasks, esta plantilla no aplica tal cual: el
header va en `proposal.md`, el archivo que abre el cambio, y los demás no lo llevan.
Las secciones de abajo sirven de guía de qué tiene que quedar cubierto.

**Antes de escribir una línea, tres chequeos** (regla 15):

1. **¿Hay otro diseño grande sin arrancar?** El WIP limit es uno. Si lo hay,
   primero arrancá o archivá ese.
2. **¿Esto es el slice vertical o el épico completo?** Diseñá *un* camino
   funcionando de punta a punta; el resto, cuando el primero esté vivo.
3. **¿El bloqueo es real o asumido?** ¿Qué parte podés mandar hoy sin la
   dependencia?

---

## Changelog

Con el código de color de la regla 18: 🔴 afecta a otro equipo · 🔵 solo de este lado.

| Fecha | Cambio | |
|---|---|---|
| YYYY-MM-DD | versión inicial | 🔵 |

## 1. Problema

<qué duele hoy, con el caso concreto>

## 2. Lo que existe hoy

<verificado en el código el YYYY-MM-DD: qué hay, dónde, y qué no hay. Sin esta
sección el diseño describe el código que alguien recuerda, no el que está>

## 3. Alcance

**Entra:** <el slice vertical mínimo, de punta a punta>

**Fuera de alcance:** <lo que explícitamente NO entra, para que no vuelva por la ventana>

## 4. Decisiones

| Decisión | Se eligió | Se descartó, y por qué |
|---|---|---|
| <…> | <…> | <…> |

## 5. Contrato

<solo si cruza a otro equipo: endpoint, payload, errores, y si falla abierto o
cerrado. Si lo cambiás después, es 🔴 en el changelog>

## 6. Diseño por lado

**Backend:** <…>

**Mobile:** <…>

**Web:** <…>

## 7. Riesgos, trade-offs y casos borde

<qué se gana, qué se pierde, qué puede salir mal, qué entradas raras hay>

## 8. Decisiones abiertas

| # | Pregunta | Quién decide | Qué bloquea |
|---|---|---|---|
| 1 | <…> | <…> | <…> |

## 9. Criterios de aceptación

<qué tiene que ser verdad para darlo por hecho — incluye ver píxeles, regla 14>

## 10. Tareas

<una por línea, como instrucción ubicada (regla 17) y con estimación en bucket
(regla 16), agrupadas por equipo>

---

## Al vencerse

Cuatro salidas, y "sigue esperando" no es ninguna:

1. **Arrancar** el slice vertical mínimo.
2. **Recortar** el alcance a lo que sí se va a construir, y archivar el resto.
3. **Archivar** con una nota de por qué, para rediseñarlo cuando haga falta.
4. **Prorrogar**, con las tres condiciones de abajo.

**La prórroga** lleva razón escrita (por qué sigue vivo y por qué no es ahora), es
**una sola vez** —a la segunda toca recortar o archivar— y **no detiene el
envejecimiento**: el documento sigue describiendo el código de cuando se escribió,
así que al reactivarlo hay que releerlo contra el código actual (sección 2).

Y el header lo dice, incluida la fecha en que venció:

```
Diseñado: YYYY-MM-DD · Venció: YYYY-MM-DD · Prorrogado hasta: YYYY-MM-DD
```