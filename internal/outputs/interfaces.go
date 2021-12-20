package outputs

import (
	"context"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type Outputter interface {
	Output(ctx context.Context, src *clogger.Messages) error
}

type Message struct {
	MonoTimestamp uint64
	ParsedFields  map[string]string
	RawMessage    string
}
