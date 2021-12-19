package inputs

import "context"

type Inputter interface {
	Fetch(ctx context.Context, dst *messageReaderContext) error
}

type Message struct {
	MonoTimestamp uint64
	ParsedFields  map[string]string
	RawMessage    string
}
