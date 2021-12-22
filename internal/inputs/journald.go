package inputs

import (
	"context"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/tracing"

	"github.com/rs/zerolog/log"
)

type JournalDReader interface {
	GetEntry(ctx context.Context) (clogger.Message, error)
	Close()
}

type CoreOSJournalDReader struct {
	reader *sdjournal.Journal
}

func NewCoreOSJournalDReader() (*CoreOSJournalDReader, error) {
	reader, err := sdjournal.NewJournal()
	if err != nil {
		return nil, err
	}

	err = reader.SeekTail()
	if err != nil {
		reader.Close()
		return nil, err
	}

	return &CoreOSJournalDReader{
		reader: reader,
	}, nil
}

func (c *CoreOSJournalDReader) Close() {
	c.reader.Close()
}

func (c *CoreOSJournalDReader) GetEntry(ctx context.Context) (clogger.Message, error) {
	_, span := tracing.GetTracer().Start(ctx, "CoreOSJournalDReader.GetEntry")
	defer span.End()

	var err error
	var i uint64

	i = 0

	for i <= 0 {
		c.reader.Wait(sdjournal.IndefiniteWait)
		i, err = c.reader.Next()

		if err != nil {
			return clogger.Message{}, err
		}
	}
	span.AddEvent("Finished Waiting")

	entry, err := c.reader.GetEntry()
	if err != nil {
		return clogger.Message{}, err
	}

	m2 := make(map[string]interface{}, len(entry.Fields))
	for k, v := range entry.Fields {
		m2[k] = v
	}

	return clogger.Message{
		MonoTimestamp: entry.MonotonicTimestamp,
		ParsedFields:  m2,
	}, nil
}

type JournalDInput struct {
	clogger.RecvConfig
	Reader JournalDReader
}

func NewJournalDInput(conf clogger.RecvConfig) (*JournalDInput, error) {
	reader, err := NewCoreOSJournalDReader()
	if err != nil {
		return nil, err
	}

	return &JournalDInput{
		RecvConfig: conf,
		Reader:     reader,
	}, nil
}

func (j *JournalDInput) Run(ctx context.Context, flushChan chan []clogger.Message) error {
	defer j.Reader.Close()
	for {
		msg, err := j.Reader.GetEntry(ctx)
		if err != nil {
			log.Err(err).Msg("Failed to read from JournalD")
			continue
		}

		flushChan <- []clogger.Message{msg}
	}

	return nil
}
