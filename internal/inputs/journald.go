package inputs

import (
	"context"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type JournalDReader interface {
	GetEntry() (Message, error)
}

type CoreOSJournalDReader struct {
	reader *sdjournal.Journal
}

func NewCoreOSJournalDReader() (*CoreOSJournalDReader, error) {
	reader, err := sdjournal.NewJournal()
	if err != nil {
		return nil, err
	}

	return &CoreOSJournalDReader{
		reader: reader,
	}, nil
}

func (c *CoreOSJournalDReader) GetEntry() (Message, error) {
	_, err := c.reader.Next()
	if err != nil {
		return Message{}, err
	}

	entry, err := c.reader.GetEntry()
	if err != nil {
		return Message{}, err
	}

	return Message{
		MonoTimestamp: entry.MonotonicTimestamp,
		ParsedFields:  entry.Fields,
	}, nil
}

type JournalDInputConfig struct {
	BatchSize int
}

type JournalDInput struct {
	Reader    JournalDReader
	BatchSize int
}

func NewJournalDInput(conf *JournalDInputConfig) (*JournalDInput, error) {
	reader, err := NewCoreOSJournalDReader()
	if err != nil {
		return nil, err
	}

	return &JournalDInput{
		BatchSize: conf.BatchSize,
		Reader:    reader,
	}, nil
}

func (j *JournalDInput) Fetch(ctx context.Context, dst *messageReaderContext) error {
	_, span := tracing.GetTracer().Start(ctx, "JournalDInput.Fetch")
	defer span.End()

	span.SetAttributes(
		attribute.Int("batch_size", j.BatchSize),
		attribute.Int("dest_buffer_size", len(dst.Messages)),
	)

	if cap(dst.Messages) < j.BatchSize {
		log.Fatal().
			Int("batch_size", j.BatchSize).
			Int("dest_size", len(dst.Messages)).
			Msg("TEST: Batch Size is bigger than the supplied buffer")
	}

	for i := 0; i < j.BatchSize; i++ {
		entry, err := j.Reader.GetEntry()
		if err != nil {
			span.RecordError(err)
			return err
		}

		dst.Messages = append(dst.Messages, entry)
	}

	return nil
}
