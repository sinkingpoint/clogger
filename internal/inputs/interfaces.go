package inputs

import (
	"context"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type Inputter interface {
	Fetch(ctx context.Context, dst *clogger.Messages) error
}

type Message struct {
	MonoTimestamp uint64
	ParsedFields  map[string]string
	RawMessage    string
}
