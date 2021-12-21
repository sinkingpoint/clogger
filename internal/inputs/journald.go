package inputs

import (
	"context"
	"sync"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/sinkingpoint/clogger/internal/clogger"

	"github.com/rs/zerolog/log"
)

type JournalDReader interface {
	GetEntry() (clogger.Message, error)
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

func (c *CoreOSJournalDReader) GetEntry() (clogger.Message, error) {
	_, err := c.reader.Next()
	if err != nil {
		return clogger.Message{}, err
	}

	entry, err := c.reader.GetEntry()
	if err != nil {
		return clogger.Message{}, err
	}

	return clogger.Message{
		MonoTimestamp: entry.MonotonicTimestamp,
		ParsedFields:  entry.Fields,
	}, nil
}

type JournalDInputConfig struct {
	clogger.SendRecvConfigBase
}

type JournalDInput struct {
	clogger.SendRecvBase
	Reader JournalDReader
}

func NewJournalDInput(conf *JournalDInputConfig) (*JournalDInput, error) {
	reader, err := NewCoreOSJournalDReader()
	if err != nil {
		return nil, err
	}

	return &JournalDInput{
		SendRecvBase: clogger.NewSendRecvBase(conf.SendRecvConfigBase),
		Reader:       reader,
	}, nil
}

func (j *JournalDInput) Run(ctx context.Context, wg sync.WaitGroup) error {
	// Start the flusher
	j.SendRecvBase.Run(ctx, wg)

	wg.Add(1)
	go func() {
		for {
			msg, err := j.Reader.GetEntry()
			if err != nil {
				log.Err(err).Msg("Failed to read from JournalD")
				continue
			}

			j.SendRecvBase.PushMessage(ctx, msg)
		}

		wg.Done()
	}()

	return nil
}
