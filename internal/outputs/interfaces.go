package outputs

import (
	"context"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type SendConfig struct {
	FlushInterval time.Duration
	BufferSize    int
	InputChan     chan []clogger.Message
}

type Sender interface {
	GetSendConfig() SendConfig
	FlushToOutput(ctx context.Context, messages []clogger.Message) error
}

type Send struct {
	SendConfig
	KillChannel   chan bool
	sender        Sender
	buffer        []clogger.Message
	lastFlushTime time.Time
}

func NewSend(config SendConfig, logic Sender) *Send {
	return &Send{
		SendConfig:    config,
		KillChannel:   make(chan bool),
		buffer:        make([]clogger.Message, 0, config.BufferSize),
		lastFlushTime: time.Now(),
		sender:        logic,
	}
}

func (s *Send) queueMessages(ctx context.Context, messages []clogger.Message) {
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
		s.sender.FlushToOutput(ctx, s.buffer)
		s.buffer = s.buffer[:0]
		s.lastFlushTime = time.Now()
	}
}

func StartOutputter(inputChan chan []clogger.Message, send Sender) {
	s := NewSend(send.GetSendConfig(), send)
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
