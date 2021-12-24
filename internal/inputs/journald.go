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
	reader      *sdjournal.Journal
	killChannel chan bool
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
		reader:      reader,
		killChannel: make(chan bool),
	}, nil
}

func (c *CoreOSJournalDReader) Close() {
	c.reader.Close()
}

func (c *CoreOSJournalDReader) GetEntry(ctx context.Context) (clogger.Message, error) {
	_, span := tracing.GetTracer().Start(ctx, "CoreOSJournalDReader.GetEntry")
	defer span.End()

	i, err := c.reader.Next()

	if err != nil {
		return clogger.Message{}, err
	}

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
	RecvConfig
	Reader JournalDReader
}

func NewJournalDInput(conf RecvConfig) (*JournalDInput, error) {
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
outer:
	for {
		select {
		case <-j.KillChannel:
			break outer
		default:
			msg, err := j.Reader.GetEntry(ctx)
			if msg.RawMessage != "" {
				if _, ok := msg.ParsedFields[clogger.MESSAGE_FIELD]; ok {
					msg.ParsedFields["raw_message"] = msg.RawMessage
				} else {
					msg.ParsedFields[clogger.MESSAGE_FIELD] = msg.RawMessage
				}
			}

			if textMsg, ok := msg.ParsedFields["MESSAGE"]; ok {
				// Normalise JournalD Formatted message field to our one
				msg.ParsedFields[clogger.MESSAGE_FIELD] = textMsg
				delete(msg.ParsedFields, "MESSAGE")
			}

			if err != nil {
				log.Err(err).Msg("Failed to read from JournalD")
				continue
			}

			flushChan <- []clogger.Message{msg}
		}
	}

	return nil
}

func (j *JournalDInput) Kill() {
	j.KillChannel <- true
}

func (j *JournalDInput) Clone() (Inputter, error) {
	return NewJournalDInput(j.RecvConfig)
}
