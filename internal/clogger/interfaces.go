package clogger

import (
	"context"
	"time"

	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const DEFAULT_BATCH_SIZE = 100
const DEFAULT_FLUSH_DURATION = 10 * time.Millisecond

type Message struct {
	MonoTimestamp uint64
	ParsedFields  map[string]interface{}
	RawMessage    string
}

type FlushFunction func(ctx context.Context, messages []Message) error

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

	span.SetAttributes(attribute.Int("num_new_messages", len(messages)))

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
	ctx, span := tracing.GetTracer().Start(ctx, "Send.Flush")
	defer span.End()

	span.SetAttributes(
		attribute.Bool("final", final),
		attribute.String("last_flush_time", s.lastFlushTime.Format(time.RFC3339)),
		attribute.Int("messages_in_queue", len(s.buffer)),
	)

	// If we aren't at the buffer limit, and we have flushed recently, and this isn't the final flush, just shortcircuit
	if time.Since(s.lastFlushTime) < s.FlushInterval && len(s.buffer) < s.BufferSize && !final {
		span.AddEvent("Skipping Flush - not ready yet")
		return
	}

	if len(s.buffer) > 0 {
		s.flushFunc(ctx, s.buffer)
		s.buffer = s.buffer[:0]
		s.lastFlushTime = time.Now()
	}
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
