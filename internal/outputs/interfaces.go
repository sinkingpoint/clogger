package outputs

import (
	"context"
	"strconv"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/outputs/format"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const DEFAULT_BATCH_SIZE = 1000
const DEFAULT_FLUSH_INTERVAL = time.Millisecond * 100

// SendConfig is a config that specifies the base fields
// for all outputs
type SendConfig struct {
	// FlushInterval is the maximum time to buffer messages before outputting
	FlushInterval time.Duration

	// BatchSize is the maximum number of messages to store in the buffer before outputting
	BatchSize int

	// Formatter is the method that converts Messages into byte streams to be piped downstream
	Formatter format.Formatter
}

// NewSendConfigFromRaw is a convenience method to construct SendConfigs from raw configs
// that might have been loaded from things like the config file
func NewSendConfigFromRaw(rawConf map[string]string) (SendConfig, error) {
	conf := SendConfig{
		FlushInterval: DEFAULT_FLUSH_INTERVAL,
		BatchSize:     DEFAULT_BATCH_SIZE,
		Formatter:     &format.JSONFormatter{},
	}

	var err error
	if s, ok := rawConf["flush_interval"]; ok {
		conf.FlushInterval, err = time.ParseDuration(s)
		if err != nil {
			return SendConfig{}, err
		}
	}

	if s, ok := rawConf["batch_size"]; ok {
		conf.BatchSize, err = strconv.Atoi(s)
		if err != nil {
			return SendConfig{}, err
		}
	}

	if s, ok := rawConf["format"]; ok {
		conf.Formatter, err = format.GetFormatterFromString(s, rawConf)
		if err != nil {
			return SendConfig{}, err
		}
	}

	return conf, nil
}

// An Outputter is a thing that can take messages and push them somewhere else
type Outputter interface {
	// GetSendConfig returns the base send config of this Outputter
	GetSendConfig() SendConfig

	// FlushToOutput takes a buffer of messages, and pushes them somewhere
	FlushToOutput(ctx context.Context, messages []clogger.Message) error
}

// Sender encapsulates the functionality that all Outputters get for free i.e. Buffering
type Sender struct {
	SendConfig
	sender        Outputter
	buffer        []clogger.Message
	lastFlushTime time.Time
}

func NewSend(config SendConfig, logic Outputter) *Sender {
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

	// If the number of messages is too big for the buffer, chunk
	// it up into buffer sized bits, Flushing inbetween
	for remainingRoom := cap(s.buffer) - len(s.buffer); remainingRoom < len(messages); {
		s.buffer = append(s.buffer, messages[:remainingRoom]...)
		s.Flush(ctx, false)
		messages = messages[remainingRoom:]
		s.buffer = s.buffer[:0]
	}

	// Once we're under the buffer limit, just add the rest to the buffer
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

	// If we have messages, send them
	if len(s.buffer) > 0 {
		s.sender.FlushToOutput(ctx, s.buffer)
		s.buffer = s.buffer[:0]
		s.lastFlushTime = time.Now()
	}
}

// StartOutputter starts up a go routine that handles all the input to the given output + buffering etc
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
