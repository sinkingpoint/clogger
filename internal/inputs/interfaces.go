package inputs

import (
	"context"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

// An Inputter is a thing that is able to read messages from somewhere
type Inputter interface {
	Run(ctx context.Context, flushChan clogger.MessageChannel) error
	Kill()
}

type RecvConfig struct {
	killChannel chan bool
}

func NewRecvConfig() RecvConfig {
	return RecvConfig{
		killChannel: make(chan bool, 1),
	}
}
