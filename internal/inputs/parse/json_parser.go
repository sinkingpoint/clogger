package parse

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/tracing"
)

type JSONParser struct{}

func (j *JSONParser) ParseStream(ctx context.Context, bytes io.ReadCloser, flushChan chan []clogger.Message) error {
	_, span := tracing.GetTracer().Start(ctx, "JSONParser.ParseStream")
	defer span.End()

	dec := json.NewDecoder(bytes)
	for {
		rawMessage := map[string]interface{}{}
		err := dec.Decode(&rawMessage)
		if err != nil {
			if err != io.EOF {
				span.RecordError(err)
				return err
			}
			break
		}

		span.AddEvent("New Message")

		message := clogger.NewMessage()
		message.ParsedFields = rawMessage
		message.MonoTimestamp = time.Now().UnixNano()

		flushChan <- []clogger.Message{message}
	}

	return nil
}
