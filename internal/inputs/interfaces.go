package inputs

import (
	"context"
	"sync"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type MessageChan = chan clogger.Messages

type Inputter interface {
	Run(ctx context.Context, wg sync.WaitGroup) error
}

type Message struct {
	MonoTimestamp uint64
	ParsedFields  map[string]string
	RawMessage    string
}
