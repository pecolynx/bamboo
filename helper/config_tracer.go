package helper

import (
	"errors"
	"io"
	"os"

	gcpexporter "github.com/GoogleCloudPlatform/opentelemetry-operations-go/exporter/trace"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
)

type JaegerConfig struct {
	Endpoint string `yaml:"endpoint" validate:"required"`
}

type TraceConfig struct {
	Exporter string        `yaml:"exporter" validate:"required"`
	Jaeger   *JaegerConfig `yaml:"jaeger"`
}

func initTracerExporter(traceConfig *TraceConfig) (sdktrace.SpanExporter, error) {
	switch traceConfig.Exporter {
	case "jaeger":
		// Create the Jaeger exporter
		return jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(traceConfig.Jaeger.Endpoint)))
	case "gcp":
		projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
		return gcpexporter.New(gcpexporter.WithProjectID(projectID))
	case "stdout":
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
			stdouttrace.WithWriter(os.Stderr),
		)
	case "none":
		return stdouttrace.New(
			stdouttrace.WithPrettyPrint(),
			stdouttrace.WithWriter(io.Discard),
		)
	default:
		return nil, errors.New("invalid argument")
	}
}

func InitTracerProvider(appName string, traceConfig *TraceConfig) (*sdktrace.TracerProvider, error) {
	exp, err := initTracerExporter(traceConfig)
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		// Always be sure to batch in production.
		sdktrace.WithBatcher(exp),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		// Record information about this application in a Resource.
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(appName),
		)),
	)

	return tp, nil
}
