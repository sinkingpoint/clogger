package inputs

import (
	"context"
	"fmt"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/tracing"

	"github.com/rs/zerolog/log"
)

// JournalDReader is an interface that reads messages off the JournalD stream
type JournalDReader interface {
	// GetEntry reads a single message off of the end of the queue
	GetEntry(ctx context.Context) (clogger.Message, error)

	// Close is provided to clean up any sockets or anything when we exit
	Close()
}

// coreOSJournalDReader is the only journalDReader that we provide at the moment
// it uses github.com/coreos/go-systemd to read off of the JournalD stream
type coreOSJournalDReader struct {
	reader *sdjournal.Journal
}

// newCoreOSJournalDReader attempts to open a new reader on the journalD stream
// erroring if we fail (e.g. if we're not on a systemd machine)
func newCoreOSJournalDReader() (*coreOSJournalDReader, error) {
	reader, err := sdjournal.NewJournal()
	if err != nil {
		return nil, err
	}

	// SeekTail to push us to the end of the queue so that we only get new messages
	// and don't double count old ones
	err = reader.SeekTail()
	if err != nil {
		reader.Close()
		return nil, err
	}

	return &coreOSJournalDReader{
		reader: reader,
	}, nil
}

func (c *coreOSJournalDReader) Close() {
	c.reader.Close()
}

func (c *coreOSJournalDReader) GetEntry(ctx context.Context) (clogger.Message, error) {
	_, span := tracing.GetTracer().Start(ctx, "CoreOSJournalDReader.GetEntry")
	defer span.End()

	// Attempt to progress to the next thing in the queue
	i, err := c.reader.Next()

	if err != nil {
		return clogger.Message{}, err
	}

	// If there wasn't anything for us to read, just wait until there is
	for i <= 0 {
		// Indefinitely wait until we have a new message
		// Note: This blocks. Need to work out how to interrupt it
		c.reader.Wait(sdjournal.IndefiniteWait)
		i, err = c.reader.Next()

		if err != nil {
			return clogger.Message{}, err
		}
	}
	span.AddEvent("Finished Waiting")

	// Read the new message
	entry, err := c.reader.GetEntry()
	if err != nil {
		return clogger.Message{}, err
	}

	// Eugh. Turn the map[string]string into a map[string]interface{}
	// Should find a better way to do this that doesn't require a whole reallocation of the map
	m2 := make(map[string]interface{}, len(entry.Fields))
	for k, v := range entry.Fields {
		m2[k] = v
	}

	return clogger.Message{
		MonoTimestamp: int64(entry.MonotonicTimestamp * 1000),
		ParsedFields:  m2,
	}, nil
}

// JournalDInput is an Input that reads off of the JournalD stream
type JournalDInput struct {
	RecvConfig
	reader JournalDReader
}

// NewJournalDInput constructs a JournalDInput with the given RecvConf
// defaulting to the CoreOSJournalDReader
func NewJournalDInput(conf RecvConfig) (*JournalDInput, error) {
	reader, err := newCoreOSJournalDReader()
	if err != nil {
		return nil, err
	}

	return &JournalDInput{
		RecvConfig: conf,
		reader:     reader,
	}, nil
}

// Constructs a JournalDInput with the given reader, incase we have any others
// in the future
func NewJournalDInputWithReader(conf RecvConfig, reader JournalDReader) (*JournalDInput, error) {
	return &JournalDInput{
		RecvConfig: conf,
		reader:     reader,
	}, nil
}

// Runs the JournalDInput, piping the stream to the given flush channel
func (j *JournalDInput) Run(ctx context.Context, flushChan clogger.MessageChannel) error {
	defer j.reader.Close()
outer:
	for {
		select {
		case <-j.killChannel:
			break outer
		default:
			msg, err := j.reader.GetEntry(ctx)

			if err != nil {
				log.Err(err).Msg("Failed to read from JournalD")
				continue
			}

			if textMsg, ok := msg.ParsedFields["MESSAGE"]; ok {
				// Normalise JournalD Formatted message field to our one
				msg.ParsedFields[clogger.MESSAGE_FIELD] = textMsg
				delete(msg.ParsedFields, "MESSAGE")
			}

			flushChan <- []clogger.Message{msg}
		}
	}

	return nil
}

func (j *JournalDInput) Kill() {
	j.killChannel <- true
}

func init() {
	// JournalDInput that reads data from the journald stream
	inputsRegistry.Register("journald", func(rawConf map[string]string) (interface{}, error) {
		conf := NewRecvConfig()

		return conf, nil
	}, func(conf interface{}) (Inputter, error) {
		if c, ok := conf.(RecvConfig); ok {
			return NewJournalDInput(c)
		}

		return nil, fmt.Errorf("invalid config passed to journald input")
	})
}
