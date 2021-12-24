package outputs

import (
	"context"
	"strconv"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const DEFAULT_BATCH_SIZE = 1000
const DEFAULT_FLUSH_INTERVAL = time.Millisecond * 100

type SendConfig struct {
	FlushInterval time.Duration
	BufferSize    int
}

func NewSendConfigFromRaw(rawConf map[string]string) (SendConfig, error) {
	conf := SendConfig{
		FlushInterval: DEFAULT_FLUSH_INTERVAL,
		BufferSize:    DEFAULT_BATCH_SIZE,
	}

	var err error
	if s, ok := rawConf["flush_interval"]; ok {
		conf.FlushInterval, err = time.ParseDuration(s)
		if err != nil {
			return SendConfig{}, err
		}
	}

	if s, ok := rawConf["batch_size"]; ok {
		conf.BufferSize, err = strconv.Atoi(s)
		if err != nil {
			return SendConfig{}, err
		}
	}

	return conf, nil
}

type Outputter interface {
	GetSendConfig() SendConfig
	Clone() (Outputter, error)
	FlushToOutput(ctx context.Context, messages []clogger.Message) error
}

type Sender struct {
	SendConfig
	sender        Outputter
	buffer        []clogger.Message
	lastFlushTime time.Time
}

func NewSend(config SendConfig, logic Outputter) *Sender {
	return &Sender{
		SendConfig:    config,
		buffer:        make([]clogger.Message, 0, config.BufferSize),
		lastFlushTime: time.Now(),
		sender:        logic,
	}
}

func (s *Sender) queueMessages(ctx context.Context, messages []clogger.Message) {
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

func (s *Sender) Flush(ctx context.Context, final bool) {
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

func StartOutputter(inputChan chan []clogger.Message, send Outputter, killChan chan bool) {
	s := NewSend(send.GetSendConfig(), send)
	ticker := time.NewTicker(s.FlushInterval)
outer:
	for {
		select {
		case <-killChan:
			s.Flush(context.Background(), true)
			break outer
		case <-ticker.C:
			s.Flush(context.Background(), false)
		case messages := <-inputChan:
			s.queueMessages(context.Background(), messages)
		}
	}
}
