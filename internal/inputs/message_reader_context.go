package inputs

import (
	"context"
	"sync"
)

type messageReaderContext struct {
	Messages []Message
}

func (m *messageReaderContext) Reset(ctx context.Context) {
	m.Messages = m.Messages[:0]
}

var messageReadersPool sync.Pool

func GetMessageReader(ctx context.Context, numMessages int) *messageReaderContext {
	reader := messageReadersPool.Get()

	if reader == nil {
		reader = &messageReaderContext{
			Messages: make([]Message, 0, numMessages),
		}
	} else {
		castReader := reader.(*messageReaderContext)
		if cap(castReader.Messages) < numMessages {
			castReader.Messages = make([]Message, 0, numMessages)
		}
	}

	return reader.(*messageReaderContext)
}

func PutMessageReader(ctx context.Context, reader *messageReaderContext) {
	reader.Reset(ctx)
	messageReadersPool.Put(reader)
}
