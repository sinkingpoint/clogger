package inputs

import (
	"context"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/tracing"
)

type GoInput struct {
	c chan string
}

func NewGoInput() *GoInput {
	return &GoInput{
		c: make(chan string),
	}
}

func (s *GoInput) Kill() {
	close(s.c)
}

func (s *GoInput) Enqueue(msg string) {
	s.c <- msg
}

func (s *GoInput) Run(ctx context.Context, flushChan clogger.MessageChannel) error {
	_, span := tracing.GetTracer().Start(ctx, "GoInput.Run")
	defer span.End()

	for msg := range s.c {
		flushChan <- clogger.SizeOneBatch(
			clogger.Message{
				MonoTimestamp: time.Now().UnixNano(),
				ParsedFields: map[string]interface{}{
					clogger.MESSAGE_FIELD: msg,
				},
			},
		)
	}

	return nil
}
