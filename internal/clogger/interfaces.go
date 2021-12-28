package clogger

import (
	"time"
)

const DEFAULT_BATCH_SIZE = 100
const DEFAULT_FLUSH_DURATION = 10 * time.Millisecond
const MESSAGE_FIELD = "message"

type MessageChannel = chan []Message

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
