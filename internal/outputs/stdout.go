package outputs

import (
	"context"
	"os"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/outputs/format"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type StdOutputterConfig struct {
	formatter string
}

type StdOutputter struct {
	formatter format.Formatter
}

func NewStdOutputter(conf *StdOutputterConfig) (*StdOutputter, error) {
	if conf.formatter == "" {
		conf.formatter = "json"
	}

	formatter, err := format.GetFormatterFromString(conf.formatter)

	if err != nil {
		return nil, err
	}

	return &StdOutputter{
		formatter: formatter,
	}, nil
}

func (s *StdOutputter) Output(ctx context.Context, src *clogger.Messages) error {
	_, span := tracing.GetTracer().Start(ctx, "StdOutputter.Outputter")
	defer span.End()

	span.SetAttributes(attribute.Int("batch_size", len(src.Messages)))

	var firstError error

	for i := range src.Messages {
		msg := &src.Messages[i]
		s, err := s.formatter.Format(msg)
		if err != nil {
			if firstError == nil {
				return err
			}

			continue
		}

		// TODO: Pool these byte arrays
		os.Stdout.Write(s)
		os.Stdout.Write([]byte("\n"))
	}

	return firstError
}
