package outputs

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/metrics"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type RetryConfig struct {
	MaxBackOffTries int
	BufferChannel   clogger.MessageChannel
	currentState    OutputResult
	lastRetryTime   time.Time
}

// Sender encapsulates the functionality that all Outputters get for free i.e. Buffering
type Sender struct {
	SendConfig
	RetryConfig
	name          string
	sender        Outputter
	buffer        *clogger.MessageBatch
	lastFlushTime time.Time
}

func NewSender(name string, logic Outputter) *Sender {
	config := logic.GetSendConfig()
	return &Sender{
		name:       name,
		SendConfig: config,
		RetryConfig: RetryConfig{
			MaxBackOffTries: 5, // arbitrary, just for testing. Must make this configurable
			currentState:    OUTPUT_SUCCESS,
		},
		buffer:        clogger.GetMessageBatch(config.BatchSize),
		lastFlushTime: time.Now(),
		sender:        logic,
	}
}

// QueueMessages takes the given messages and appends them to the buffer,
// flushing as necessary
func (s *Sender) QueueMessages(ctx context.Context, batch *clogger.MessageBatch) {
	ctx, span := tracing.GetTracer().Start(ctx, "Sender.QueueMessages")
	defer span.End()

	span.SetAttributes(
		attribute.Int("buffer_size", len(s.buffer.Messages)),
		attribute.Int("num_new_messages", len(batch.Messages)),
		attribute.Int("remaining_room", cap(s.buffer.Messages)-len(s.buffer.Messages)),
	)

	metrics.MessagesProcessed.WithLabelValues(s.name, "output").Add(float64(len(batch.Messages)))

	chunks := 1

	batchMessages := batch.Messages

	for remainingRoom := cap(s.buffer.Messages) - len(s.buffer.Messages); remainingRoom < len(batch.Messages); remainingRoom = cap(s.buffer.Messages) - len(s.buffer.Messages) {
		// Chunk the data into buffer sized pieces
		chunks += 1
		s.buffer.Messages = append(s.buffer.Messages, batchMessages[:remainingRoom]...)
		s.Flush(ctx, false)
		batchMessages = batchMessages[remainingRoom:]
	}

	span.SetAttributes(attribute.Int("chunks", chunks))

	clogger.PutMessageBatch(batch)

	s.buffer.Messages = append(s.buffer.Messages, batchMessages...)
}

func (s *Sender) transitionState(ctx context.Context, state OutputResult) {
	s.currentState = state

	for result, str := range allOutputs {
		if result == state {
			metrics.OutputState.WithLabelValues(s.name, str).Set(1)
		} else {
			metrics.OutputState.WithLabelValues(s.name, str).Set(0)
		}
	}
}

// handleLongFailure handles the buffer in the event that the main sender fails
func (s *Sender) handleLongFailure(ctx context.Context) error {
	_, span := tracing.GetTracer().Start(ctx, "Sender.handleLongFailure")
	defer span.End()
	span.SetAttributes(attribute.Bool("has_bufferchannel", s.BufferChannel != nil), attribute.Int("buffer_size", len(s.buffer.Messages)))

	s.transitionState(ctx, OUTPUT_LONG_FAILURE)
	metrics.OutputState.WithLabelValues(s.name, "success").Set(0)

	if s.BufferChannel != nil {
		s.BufferChannel <- clogger.CloneBatch(s.buffer)
	}

	s.buffer.Messages = s.buffer.Messages[:0]

	return nil
}

// doExponentialRetry handles the case where we have transient failures that can be retried
// Note: This has the potential to cause double counting of logs (at least once delivery)
func (s *Sender) doExponentialRetry(ctx context.Context) error {
	ctx, span := tracing.GetTracer().Start(ctx, "Sender.doExponentialRetry")
	defer span.End()

	span.SetAttributes(attribute.Int("buffer_size", len(s.buffer.Messages)))
	// start at one because we assume we've already done one attempt at flushing
	// to get here
	backoffTime := time.Millisecond * 100

	for i := 1; i < s.MaxBackOffTries; i++ {
		time.Sleep(backoffTime)

		result, err := s.sender.FlushToOutput(ctx, s.buffer)
		if err != nil {
			log.Debug().Err(err).Int("output_result", int(result)).Msg("Failed to flush output")
		}

		switch result {
		case OUTPUT_SUCCESS:
			span.SetAttributes(attribute.Int("success_after", i))
			s.buffer.Messages = s.buffer.Messages[:0]
			s.lastFlushTime = time.Now()

			s.transitionState(ctx, OUTPUT_SUCCESS)
			return nil
		case OUTPUT_TRANSIENT_FAILURE:
			backoffTime *= 2
			continue
		case OUTPUT_LONG_FAILURE:
			return s.handleLongFailure(ctx)
		}
	}

	return fmt.Errorf("did a backoff without success")
}

// Flush flushes the current buffer to the output stream
func (s *Sender) Flush(ctx context.Context, final bool) {
	ctx, span := tracing.GetTracer().Start(ctx, "Sender.Flush")
	defer span.End()

	span.SetAttributes(
		attribute.Bool("final", final),
		attribute.String("last_flush_time", s.lastFlushTime.Format(time.RFC3339)),
		attribute.Int("buffer_size", len(s.buffer.Messages)),
	)

	enoughTimeSinceLastFlush := time.Since(s.lastFlushTime) >= s.FlushInterval
	reachedBufferLimit := len(s.buffer.Messages) >= s.BatchSize
	if !enoughTimeSinceLastFlush && !reachedBufferLimit && !final {
		span.AddEvent("Skipping Flush - not ready yet")
		return
	}

	if len(s.buffer.Messages) > 0 {
		// Don't do any exponential backoff or anything if we know that we're in a long failure
		// just buffer it, but retry every minute or so incase we're back
		if s.currentState == OUTPUT_LONG_FAILURE && time.Since(s.lastRetryTime) < 30*time.Second {
			s.handleLongFailure(ctx)
			return
		}

		s.lastRetryTime = time.Now()

		result, err := s.sender.FlushToOutput(ctx, s.buffer)
		if err != nil {
			// We just log errors - retries etc should be controlled by the OutputResult return
			log.Debug().Err(err).Int("output_result", int(result)).Msg("Failed to flush output")
		}

		switch result {
		case OUTPUT_SUCCESS:
			s.buffer.Messages = s.buffer.Messages[:0]
			s.lastFlushTime = time.Now()
			s.transitionState(ctx, OUTPUT_SUCCESS)
		case OUTPUT_TRANSIENT_FAILURE:
			err := s.doExponentialRetry(ctx)
			if err != nil {
				log.Warn().Err(err).Msg("Fell through trying to do exponential backoff")
				s.handleLongFailure(ctx)
			}
		case OUTPUT_LONG_FAILURE:
			s.handleLongFailure(ctx)
		}
	}

	if final {
		clogger.PutMessageBatch(s.buffer)
		s.buffer = nil
	}
}
