package tracing

import (
	"context"
	"os"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	tracesdk "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
)

const TRACING_INSTRUMENTATION_NAME = "clogger"

func GetTracer() trace.Tracer {
	return otel.Tracer(TRACING_INSTRUMENTATION_NAME)
}

type TracingConfig struct {
	// The name of the service that is doing tracing
	ServiceName string

	// If true, only log spans, don't send them off (Default: false)
	Debug bool

	// The rate at which to sample (0 - 1), where 0 (default) is no sampling (send no spans)
	// and 1 is send all the spans
	SamplingRate float64

	// The span propagation format (Defaults to B2)
	Propagator propagation.TextMapPropagator
}

func InitTracing(config TracingConfig) *tracesdk.TracerProvider {
	var sampler tracesdk.Sampler

	// Treat > 1 as 1 and < 0 as 0
	if config.SamplingRate >= 1 {
		sampler = tracesdk.AlwaysSample()
	} else if config.SamplingRate <= 0 {
		sampler = tracesdk.NeverSample()
	} else {
		sampler = tracesdk.TraceIDRatioBased(config.SamplingRate)
	}

	// Default the propagator to B3
	if config.Propagator == nil {
		config.Propagator = b3.B3{}
	}

	otel.SetTextMapPropagator(config.Propagator)

	apiKey := os.Getenv("HONEYCOMB_API_KEY")
	dataset := os.Getenv("HONEYCOMB_DATASET")
	otlpEndpoint := os.Getenv("OTLP_ENDPOINT")
	var exporter tracesdk.SpanExporter
	var err error
	if apiKey != "" && dataset != "" && !config.Debug {
		log.Info().Bool("debug", config.Debug).Msg("Initializing honeycomb tracing")
		client := otlptracehttp.NewClient(
			otlptracehttp.WithHeaders(
				map[string]string{
					"x-honeycomb-team":    apiKey,
					"x-honeycomb-dataset": dataset,
				},
			),
			otlptracehttp.WithEndpoint("api.honeycomb.io:443"),
		)
		exporter, err = otlptrace.New(context.Background(), client)
	} else if otlpEndpoint != "" && !config.Debug {
		log.Info().Bool("debug", config.Debug).Msg("Initializing generic otlp tracing")
		client := otlptracehttp.NewClient(
			otlptracehttp.WithEndpoint(otlpEndpoint),
			otlptracehttp.WithInsecure(),
		)
		exporter, err = otlptrace.New(context.Background(), client)
	} else {
		log.Info().Bool("debug", config.Debug).Msg("Initializing stdout tracing")
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
	}

	if err != nil {
		log.Info().Bool("debug", config.Debug).Err(err).Msg("Error initializing tracing")
	}

	tp := tracesdk.NewTracerProvider(
		tracesdk.WithBatcher(exporter),
		tracesdk.WithSampler(sampler),
		tracesdk.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(config.ServiceName),
		)),
	)

	otel.SetTracerProvider(tp)

	return tp
}
