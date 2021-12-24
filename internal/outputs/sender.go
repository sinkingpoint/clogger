package outputs

import (
	"context"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

// Sender encapsulates the functionality that all Outputters get for free i.e. Buffering
type Sender struct {
	SendConfig
	sender        Outputter
	buffer        []clogger.Message
	lastFlushTime time.Time
}

func NewSender(config SendConfig, logic Outputter) *Sender {
	return &Sender{
		SendConfig:    config,
		buffer:        make([]clogger.Message, 0, config.BatchSize),
		lastFlushTime: time.Now(),
		sender:        logic,
	}
}

// queueMessages takes the given messages and appends them to the buffer,
// flushing as necessary
func (s *Sender) queueMessages(ctx context.Context, messages []clogger.Message) {
	ctx, span := tracing.GetTracer().Start(ctx, "Send.queueMessages")
	defer span.End()

	span.SetAttributes(attribute.Int("num_new_messages", len(messages)))

	for remainingRoom := cap(s.buffer) - len(s.buffer); remainingRoom < len(messages); {
		// Chunk the data into buffer sized pieces
		s.buffer = append(s.buffer, messages[:remainingRoom]...)
		s.Flush(ctx, false)
		messages = messages[remainingRoom:]
		s.buffer = s.buffer[:0]
	}

	s.buffer = append(s.buffer, messages...)
}

// Flush flushes the current buffer to the output stream
func (s *Sender) Flush(ctx context.Context, final bool) {
	ctx, span := tracing.GetTracer().Start(ctx, "Send.Flush")
	defer span.End()

	span.SetAttributes(
		attribute.Bool("final", final),
		attribute.String("last_flush_time", s.lastFlushTime.Format(time.RFC3339)),
		attribute.Int("messages_in_queue", len(s.buffer)),
	)
	// If we aren't at the buffer limit, and we have flushed recently, and this isn't the final flush, just shortcircuit
	if time.Since(s.lastFlushTime) < s.FlushInterval && len(s.buffer) < s.BatchSize && !final {
		span.AddEvent("Skipping Flush - not ready yet")
		return
	}

	if len(s.buffer) > 0 {
		s.sender.FlushToOutput(ctx, s.buffer)
		s.buffer = s.buffer[:0]
		s.lastFlushTime = time.Now()
	}
}
