package parse

import (
	"bufio"
	"context"
	"io"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/tracing"
)

type NewlineParser struct{}

func (j *NewlineParser) ParseStream(ctx context.Context, bytes io.ReadCloser, flushChan clogger.MessageChannel) error {
	_, span := tracing.GetTracer().Start(ctx, "NewlineParser.ParseStream")
	defer span.End()

	scanner := bufio.NewScanner(bytes)
	for scanner.Scan() {
		line := scanner.Text()
		flushChan <- []clogger.Message{
			{
				MonoTimestamp: time.Now().UnixNano(),
				ParsedFields: map[string]interface{}{
					clogger.MESSAGE_FIELD: line,
				},
			},
		}
	}

	return scanner.Err()
}
