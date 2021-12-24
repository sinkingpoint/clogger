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
	SendConfig
	Formatter format.Formatter
}

type StdOutputter struct {
	SendConfig
	formatter format.Formatter
}

func NewStdOutputter(conf StdOutputterConfig) (*StdOutputter, error) {
	return &StdOutputter{
		SendConfig: conf.SendConfig,
		formatter:  conf.Formatter,
	}, nil
}

func (s *StdOutputter) GetSendConfig() SendConfig {
	return s.SendConfig
}

func (s *StdOutputter) FlushToOutput(ctx context.Context, messages []clogger.Message) error {
	_, span := tracing.GetTracer().Start(ctx, "StdOutputter.FlushToOutput")
	defer span.End()

	span.SetAttributes(attribute.Int("batch_size", len(messages)))

	var firstError error

	for i := range messages {
		msg := &messages[i]
		msg.ParsedFields["auth_timestamp"] = msg.MonoTimestamp
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

func (s *StdOutputter) Clone() (Outputter, error) {
	return &StdOutputter{
		SendConfig: s.SendConfig,
		formatter:  s.formatter,
	}, nil
}
