package outputs

import (
	"context"
	"os"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// StdOutputterConfig is a shim around SendConfig for now
// mainly so that we can extend it in the future if necessary
type StdOutputterConfig struct {
	SendConfig
}

// StdOutputter is an Outputter that takes messages from the input stream
// and pushes them to stdout (fd 0)
type StdOutputter struct {
	SendConfig
}

// NewStdOutputter constructs a new StdOutputter from the given Config
func NewStdOutputter(conf StdOutputterConfig) (*StdOutputter, error) {
	return &StdOutputter{
		SendConfig: conf.SendConfig,
	}, nil
}

func (s *StdOutputter) GetSendConfig() SendConfig {
	return s.SendConfig
}

func (s *StdOutputter) FlushToOutput(ctx context.Context, messages []clogger.Message) (OutputResult, error) {
	_, span := tracing.GetTracer().Start(ctx, "StdOutputter.FlushToOutput")
	defer span.End()

	span.SetAttributes(attribute.Int("batch_size", len(messages)))

	var firstError error

	for i := range messages {
		msg := &messages[i]
		// Add in the timestamp so that it gets pushed
		msg.ParsedFields["auth_timestamp"] = msg.MonoTimestamp
		s, err := s.Formatter.Format(msg)
		if err != nil {
			if firstError == nil {
				firstError = err
			}

			continue
		}

		// TODO: Pool these byte arrays
		os.Stdout.Write(s)
		os.Stdout.Write([]byte("\n"))
	}

	// OUTPUT_SUCCESS here so that we don't retry - it's likely that the errors are bad data, or a bug in the formatter
	// either way, retrying would be pointless
	return OUTPUT_SUCCESS, firstError
}
