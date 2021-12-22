package inputs

import (
	"context"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type Inputter interface {
	Run(ctx context.Context, flushChan chan []clogger.Message) error
}

type Message struct {
	MonoTimestamp uint64
	ParsedFields  map[string]string
	RawMessage    string
}
