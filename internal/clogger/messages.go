package clogger

import (
	"sync"
	"time"
)

const DEFAULT_BATCH_SIZE = 100
const DEFAULT_FLUSH_DURATION = 10 * time.Millisecond
const MESSAGE_FIELD = "message"

type MessageChannel = chan *MessageBatch
type MessageBatch struct {
	Messages []Message
}

type Message struct {
	MonoTimestamp int64
	ParsedFields  map[string]interface{}
}

func NewMessage() Message {
	return Message{
		MonoTimestamp: time.Now().UnixNano(),
		ParsedFields:  make(map[string]interface{}),
	}
}

func (m *Message) Reset() {
	for k := range m.ParsedFields {
		delete(m.ParsedFields, k)
	}
}

var batchPool sync.Pool

func GetMessageBatch(size int) (batch *MessageBatch) {
	b := batchPool.Get()
	if b == nil {
		b = &MessageBatch{
			Messages: make([]Message, size),
		}
	}

	batch = b.(*MessageBatch)
	if cap(batch.Messages) < size {
		batch.Messages = append(batch.Messages[:cap(batch.Messages)], make([]Message, size-cap(batch.Messages))...)
	}

	batch.Messages = batch.Messages[:0]

	return batch
}

func SizeOneBatch(m Message) *MessageBatch {
	batch := GetMessageBatch(1)
	batch.Messages = append(batch.Messages, m)

	return batch
}

func PutMessageBatch(m *MessageBatch) {
	batchPool.Put(m)
}
