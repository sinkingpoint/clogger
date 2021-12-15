package inputs

import "context"

type Inputter interface {
	Fetch(ctx context.Context, dst *messageReaderContext)
}

type Message struct {
	ParsedFields map[string]string
	RawMessage   string
}
