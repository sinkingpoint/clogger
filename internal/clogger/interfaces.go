package clogger

import (
	"context"
	"time"

	"github.com/sinkingpoint/clogger/internal/tracing"
)

const DEFAULT_BATCH_SIZE = 100
const DEFAULT_FLUSH_DURATION = 10 * time.Millisecond

type Message struct {
	MonoTimestamp uint64
	ParsedFields  map[string]interface{}
	RawMessage    string
}

type FlushFunction func(ctx context.Context, messages []Message) error

type RecvConfig struct {
	KillChannel chan bool
}

func NewRecvConfig() RecvConfig {
	return RecvConfig{
		KillChannel: make(chan bool),
	}
}

type SendConfig struct {
	FlushInterval time.Duration
	BufferSize    int
	InputChan     chan []Message
}

type Sender interface {
	Run(inputChan chan []Message)
}

type Send struct {
	SendConfig
	KillChannel   chan bool
	buffer        []Message
	lastFlushTime time.Time
	flushFunc     FlushFunction
}

func NewSend(config SendConfig) *Send {
	return &Send{
		SendConfig:    config,
		KillChannel:   make(chan bool),
		buffer:        make([]Message, 0, config.BufferSize),
		lastFlushTime: time.Now(),
	}
}

func (s *Send) queueMessages(ctx context.Context, messages []Message) {
	ctx, span := tracing.GetTracer().Start(ctx, "Send.queueMessages")
	defer span.End()
	// We don't have enough room in the buffer for all these messages
	// add what we can, flush, and then store the rest in the buffer
	remainingRoom := cap(s.buffer) - len(s.buffer)
	if remainingRoom < len(messages) {
		s.buffer = append(s.buffer, messages[:remainingRoom]...)
		s.Flush(ctx, false)
		s.buffer = append(s.buffer[:0], messages[remainingRoom:]...)
		return
	}

	// Otherwise, just add them to the buffer
	s.buffer = append(s.buffer, messages...)
}

func (s *Send) Flush(ctx context.Context, final bool) {
	// If we aren't at the buffer limit, and we have flushed recently, and this isn't the final flush, just shortcircuit
	if time.Since(s.lastFlushTime) < s.FlushInterval && len(s.buffer) < s.BufferSize && !final {
		return
	}

	s.flushFunc(ctx, s.buffer)
	s.buffer = s.buffer[0:]
	s.lastFlushTime = time.Now()
}

func Run(inputChan chan []Message, s Send, flushFunc FlushFunction) {
	s.flushFunc = flushFunc
	ticker := time.NewTicker(s.FlushInterval)
outer:
	for {
		select {
		case <-s.KillChannel:
			s.Flush(context.Background(), true)
			break outer
		case <-ticker.C:
			s.Flush(context.Background(), false)
		case messages := <-inputChan:
			s.queueMessages(context.Background(), messages)
		}
	}
}
