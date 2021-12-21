package clogger

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

const DEFAULT_BATCH_SIZE = 100
const DEFAULT_FLUSH_DURATION = 10 * time.Millisecond

type Message struct {
	MonoTimestamp uint64
	ParsedFields  map[string]string
	RawMessage    string
}

type Messages []Message

type SendRecvConfigBase struct {
	BatchSize    int
	BatchDelay   time.Duration
	FlushChannel chan Messages
	KillChannel  chan bool
}

func NewSendRecvConfigBase(batchSize int, batchDelay time.Duration, flushChannel chan Messages) SendRecvConfigBase {
	return SendRecvConfigBase{
		BatchSize:    batchSize,
		BatchDelay:   batchDelay,
		FlushChannel: flushChannel,
		KillChannel:  make(chan bool, 1),
	}
}

func NewSendRecvConfigBaseFromRaw(raw map[string]interface{}, flushChannel chan Messages) (SendRecvConfigBase, error) {
	var batchSize int
	batchSizeInterface, existOk := raw["batch_size"]
	if existOk {
		b, ok := batchSizeInterface.(int)
		if !ok {
			return SendRecvConfigBase{}, fmt.Errorf("invalid type for batch_size, expected int")
		}
		batchSize = b
	} else {
		batchSize = DEFAULT_BATCH_SIZE
	}

	var flushDuration time.Duration
	flushDurationInterface, existOk := raw["flush_duration"]
	if existOk {
		f, ok := flushDurationInterface.(string)
		if !ok {
			return SendRecvConfigBase{}, fmt.Errorf("invalid type for batch_size, expected int")
		}
		f2, err := time.ParseDuration(f)
		if err != nil {
			return SendRecvConfigBase{}, fmt.Errorf("error parsing flush duration: `%s`", f)
		}

		flushDuration = f2
	} else {
		flushDuration = DEFAULT_FLUSH_DURATION
	}

	return NewSendRecvConfigBase(batchSize, flushDuration, flushChannel), nil
}

type SendRecvBase struct {
	SendRecvConfigBase
	lastFlushTime time.Time
	buffer        Messages
}

func NewSendRecvBase(conf SendRecvConfigBase) SendRecvBase {
	return SendRecvBase{
		SendRecvConfigBase: conf,
		lastFlushTime:      time.Now(),
		buffer:             make(Messages, 0, conf.BatchSize),
	}
}

func (s *SendRecvBase) PushMessage(ctx context.Context, m Message) error {
	s.buffer = append(s.buffer, m)
	return s.Flush(ctx)
}

func (s *SendRecvBase) Run(ctx context.Context, wg sync.WaitGroup) error {
	timer := time.NewTicker(s.BatchDelay)

	wg.Add(1)
	go func() {
	outer:
		for {
			select {
			case <-s.KillChannel:
				break outer
			default:
				<-timer.C
				err := s.Flush(ctx)
				if err != nil {
				}
			}
		}

		wg.Done()
	}()

	return nil
}

func (s *SendRecvBase) Flush(ctx context.Context) error {
	_, span := tracing.GetTracer().Start(ctx, "SendRecvBase.Flush")
	defer span.End()

	if len(s.buffer) < s.BatchSize && time.Since(s.lastFlushTime) < s.BatchDelay {
		// We aren't at the buffer limit, and we haven't hit the time limit
		return nil
	}

	log.Info().Interface("message", s.buffer).Msg("Inserting Message")
	span.SetAttributes(attribute.Int("batch_size", len(s.buffer)))

	s.FlushChannel <- s.buffer
	s.buffer = s.buffer[0:]
	s.lastFlushTime = time.Now()

	return nil
}
