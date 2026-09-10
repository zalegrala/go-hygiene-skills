---
name: tracing-hygiene
description: Use when auditing or fixing Go code instrumented with OpenTelemetry where spans are created inside a loop (traces exploding to hundreds/thousands of spans for bulk or batch processing) or where error-returning functions never mark their span as errored (missing SetStatus/RecordError), making traces unreliable to query or trust for both humans and automated analysis.
---

# Tracing Hygiene (Go / OpenTelemetry)

## Overview

Two specific defects make Go traces unusable: creating a span per loop
iteration (span-count explosion), and creating a span that can fail but never
marking it errored (silent failures in traces). Both have a small, mechanical
fix. See `references/patterns.md` for the full before/after and the decision
between a count attribute vs. a span event when collapsing a loop.

## When to use

- Reviewing or cleaning up Go tracing instrumentation for a service.
- A trace looks huge / slow to load / has repeated near-identical span names.
- `grep`-ing a codebase turns up many `defer span.End()` but few
  `SetStatus(codes.Error, ...)` calls relative to that.
- Someone asks to "fix our tracing" or reduce span count.

## Hard constraint: tracing-only diff

Every change under this skill touches **only** tracing/instrumentation
statements: span creation, attributes, events, status/error calls, and the
minimal control-flow change needed to expose `err` to a defer (e.g.
converting `func f() error` to `func f() (err error)`). Nothing else.

This constraint exists so the resulting change is a small, reviewable,
tracing-only PR a team can merge with confidence that behavior didn't
change. It holds even when other cleanup is visible, tempting, or explicitly
invited ("clean up anything else you notice").

| Rationalization | Reality |
|---|---|
| "This unused import/variable is right there, I'll remove it too" | Out of scope. A tracing PR that also touches imports/dead code stops being reviewable as "just tracing." Leave it, or note it separately. |
| "Renaming this makes the tracing fix clearer" | Only rename if the *tracing* fix requires it (e.g. resolving a shadowed `err`). Renaming for general clarity is a separate change. |
| "I collapsed the return with err to be more idiomatic" | Don't simplify unrelated logic while you're in the function. Touch only the lines the tracing fix requires. |
| "The user said I could clean up anything else" | An invitation to clean up doesn't extend to this skill's changes — report other findings separately instead of folding them into the tracing diff. |

**Red flags — stop and split the change:** a diff hunk with no `span`,
`tracer`, `otel`, `trace.`, or `codes.` token in it; a renamed identifier
that isn't the `err` involved in the error-status fix; a removed import
unrelated to tracing packages; a control-flow rewrite beyond exposing `err`
to a defer.

## Quick reference

| Detector | Symptom | Fix |
|---|---|---|
| Span-per-loop-iteration | `tracer.Start` inside a `for`/`range` body, or inside a goroutine launched from one | One span around the loop; count attribute if per-item identity has no debugging value, span event per iteration if it does. See patterns.md Pattern 1. |
| Missing/incomplete span error status | Function can return non-nil error but no `SetStatus(codes.Error, ...)`; or `RecordError` without `SetStatus`; or no explicit `codes.Ok` on success | One `defer` right after span creation using `references/error-helper.go`'s `RecordErr`, reusing/exposing the function's `err`. See patterns.md Pattern 2. |

## Workflow

1. **Audit**: scan the target package(s) for both detectors (grep for
   `tracer.Start`/`.Start(ctx` near `for `/`range `; grep for `defer
   span.End()` sites without a matching `SetStatus` in the same function).
   Bucket findings by package and pattern.
2. **Report** findings with file:line and which detector matched, before
   changing anything.
3. **Apply on request**, one bucket at a time, respecting the tracing-only
   constraint above. Skip and flag (don't force) any site where the fix
   would require non-trivial control-flow changes — e.g. an `err` shadowed
   in a nested block that can't be exposed to a defer without restructuring.
4. Keep each applied bucket small enough to be its own PR (e.g. "tracing
   hygiene: collapse loop spans in `livestore`").

## Reference files

- `references/patterns.md` — detailed before/after for both detectors, the
  count-attribute-vs-span-event decision, and the shadowed-`err` caveat.
- `references/error-helper.go` — the local helper to copy into the target
  repo (not a dependency to import) combining `RecordError` + `SetStatus`.
