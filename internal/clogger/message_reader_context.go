package clogger

import (
	"context"
	"sync"
)

type Messages struct {
	Messages []Message
}

func (m *Messages) Reset(ctx context.Context) {
	m.Messages = m.Messages[:0]
}

var messageReadersPool sync.Pool

func GetMessages(ctx context.Context, numMessages int) *Messages {
	reader := messageReadersPool.Get()

	if reader == nil {
		reader = &Messages{
			Messages: make([]Message, 0, numMessages),
		}
	} else {
		castReader := reader.(*Messages)
		if cap(castReader.Messages) < numMessages {
			castReader.Messages = make([]Message, 0, numMessages)
		}
	}

	return reader.(*Messages)
}

func PutMessages(ctx context.Context, reader *Messages) {
	reader.Reset(ctx)
	messageReadersPool.Put(reader)
}
