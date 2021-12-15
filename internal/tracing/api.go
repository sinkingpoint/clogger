package tracing

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const TRACING_INSTRUMENTATION_NAME = "clogger"

func GetTracer() trace.Tracer {
	return otel.Tracer(TRACING_INSTRUMENTATION_NAME)
}
