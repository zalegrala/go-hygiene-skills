---
name: tracing-hygiene
description: Use when auditing or fixing Go code instrumented with OpenTelemetry — spans created inside a loop (traces exploding to hundreds/thousands of spans for bulk or batch processing), error-returning functions that never mark their span as errored (missing SetStatus/RecordError), or an operation with real internal structure that has no span at all, making traces unreliable to query or trust for both humans and automated analysis.
---

# Tracing Hygiene (Go / OpenTelemetry)

## Overview

Tracing hygiene isn't only about removing spans — it's about improving the
signal. Three defects covered here: creating a span per loop iteration
(span-count explosion), creating a span that can fail but never marking it
errored (silent failures in traces), and an operation worth isolating that
has no span at all (missing signal). The first two have a small, mechanical
fix; the third needs judgment about the surrounding code, not a mechanical
rule. See `references/patterns.md` for full before/afters and decision
guidance.

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
invited ("clean up anything else you notice"). Adding a *new* span (Pattern
3) is still in scope — it's a tracing/instrumentation statement — as long as
nothing else in the diff moves.

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
| Span-per-loop-iteration | `tracer.Start` inside a `for`/`range` body or a goroutine launched from one, **or** reachable exactly once per iteration through a callback one or more calls away from the loop (e.g. a `blockFn`/closure the loop invokes) | One span around the loop; count attribute if per-item identity has no debugging value, span event per iteration if it does. **First check the correctness gate below** — this is the fix most likely to be unsafe to apply mechanically. See patterns.md Pattern 1. |
| Missing/incomplete span error status | Function can return non-nil error but no `SetStatus(codes.Error, ...)`; or `RecordError` without `SetStatus`; or no explicit `codes.Ok` on success; or the error surfaces via a side-effecting callback (e.g. `handleErr(err)`) instead of a return value | One `defer` right after span creation using `references/error-helper.go`'s `RecordErr`, reusing/exposing the function's `err` (name it `_` if there's already a local named `err`/`resp` to avoid collisions) — or, for the side-effecting-callback shape, a direct (non-deferred) `RecordErr(span, err)` call at the site where `err` is known. See patterns.md Pattern 2. |
| Missing span for an operation with real internal structure | An error-returning function does meaningful, isolable work but has no span at all, inconsistent with sibling functions that do | **Propose**, don't mechanically apply, a new span — gated on whether it would add signal (see the value gate in patterns.md Pattern 3). Always surface for explicit review; never batch-apply like the other two. |

## Workflow

1. **Audit**: scan the target package(s) for all three detectors (grep for
   `tracer.Start`/`.Start(ctx` near `for `/`range ` *and* trace call graphs
   through loop-invoked callbacks; grep for `defer span.End()` sites without
   a matching `SetStatus` in the same function; note functions with real
   internal structure and no span at all, especially next to siblings that
   have one). Bucket findings by package and pattern.
2. **Report** findings with file:line and which detector matched, before
   changing anything. For Pattern-1 candidates, note in the report whether
   the correctness gate (below) passed. For Pattern-3 candidates, note the
   value-gate reasoning (children/attributes/call-frequency) up front — this
   is a proposal, not a foregone fix.
3. **Correctness gate for Pattern 1 — check before collapsing, not after:**
   grep the whole function (and anything it calls with the per-iteration
   `ctx`) for `SpanFromContext(ctx)` or a nested `tracer.Start` reached
   through that `ctx`. If either exists, collapsing the per-iteration span
   changes what those consumers attach to or parent onto — a behavior
   change hiding behind what looks like "just tracing." Skip and flag
   instead of forcing it.
4. **Apply on request**: Pattern 1 and 2 fixes, one bucket at a time,
   respecting the tracing-only constraint above. Skip and flag (don't
   force) any site where the fix would require non-trivial control-flow
   changes — e.g. an `err` shadowed in a nested block, or a panic-recovery
   `defer` declared before the `span` variable it would need to close over
   (same rule: reordering to make it fit is out of scope, flag instead).
   Pattern 3 proposals are applied only one at a time, after the team
   confirms the specific instance is worth the added span.
5. Keep each applied bucket small enough to be its own PR (e.g. "tracing
   hygiene: collapse loop spans in `livestore`").

## Reference files

- `references/patterns.md` — detailed before/after for all three patterns,
  the correctness gate, the count-attribute-vs-span-event decision, the
  shadowed-`err` and panic-recovery-ordering caveats, and the Pattern 3
  value gate.
- `references/error-helper.go` — the local helper to copy into the target
  repo (not a dependency to import) combining `RecordError` + `SetStatus`.
