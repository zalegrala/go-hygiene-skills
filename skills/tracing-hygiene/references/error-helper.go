// Package tracing is a reference snippet, not a dependency to import.
// Copy this into the target repo (e.g. internal/tracing/errhandler.go) so
// no new external module is required.
package tracing

import (
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RecordErr marks span as errored (RecordError + SetStatus) when err is
// non-nil, or explicitly Ok when it is nil. Use from a defer, right after
// the span is created, with a named or closed-over err:
//
//	func doThing(ctx context.Context) (err error) {
//		ctx, span := tracer.Start(ctx, "doThing")
//		defer func() { tracing.RecordErr(span, err); span.End() }()
//		...
//		return err
//	}
//
// Both RecordError and SetStatus matter: RecordError attaches the error
// message/stack as an exception event; SetStatus is what flips the span's
// status field so `{ status = error }` TraceQL queries, error-rate
// dashboards, and automated trace analysis (e.g. Sift, or an LLM doing root
// cause analysis) can find it. RecordError alone does not mark a span as
// errored. Setting Ok explicitly (not leaving it Unset) makes the status
// field a complete, trustworthy signal instead of ambiguous on the success
// path.
func RecordErr(span trace.Span, err error) {
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return
	}
	span.SetStatus(codes.Ok, "")
}
