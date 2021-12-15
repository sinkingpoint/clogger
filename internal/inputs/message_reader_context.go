package inputs

import (
	"context"
	"sync"
)

type messageReaderContext struct {
	messages []Message
}

func (m *messageReaderContext) Reset(ctx context.Context) {
	m.messages = m.messages[:0]
}

var messageReadersPool sync.Pool

func GetMessageReader(ctx context.Context, numMessages int) *messageReaderContext {
	reader := messageReadersPool.Get().(*messageReaderContext)
	if reader == nil {
		reader = &messageReaderContext{
			messages: make([]Message, 0, numMessages),
		}
	} else {
		if cap(reader.messages) < numMessages {
			reader.messages = make([]Message, 0, numMessages)
		}
	}

	return reader
}

func PutMessageReader(ctx context.Context, reader *messageReaderContext) {
	reader.Reset(ctx)
	messageReadersPool.Put(reader)
}
