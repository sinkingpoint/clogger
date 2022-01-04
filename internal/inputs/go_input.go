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

func (g *GoInput) Init(ctx context.Context) error {
	return nil
}

func (s *GoInput) Close(ctx context.Context) error {
	close(s.c)
	return nil
}

func (s *GoInput) Enqueue(msg string) {
	s.c <- msg
}

func (g *GoInput) GetBatch(ctx context.Context) (*clogger.MessageBatch, error) {
	_, span := tracing.GetTracer().Start(ctx, "GoInput.Run")
	defer span.End()

	select {
	case <-ctx.Done():
		return nil, nil
	case msg := <-g.c:
		numMessages := len(g.c)
		batch := clogger.GetMessageBatch(numMessages + 1)
		batch.Messages = append(batch.Messages, clogger.Message{
			MonoTimestamp: time.Now().UnixNano(),
			ParsedFields: map[string]interface{}{
				clogger.MESSAGE_FIELD: msg,
			},
		})

		for i := 0; i < numMessages; i++ {
			batch.Messages = append(batch.Messages, clogger.Message{
				MonoTimestamp: time.Now().UnixNano(),
				ParsedFields: map[string]interface{}{
					clogger.MESSAGE_FIELD: msg,
				},
			})
		}

		return batch, nil
	}
}
