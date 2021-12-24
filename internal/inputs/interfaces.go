package inputs

import (
	"context"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

// An Inputter is a thing that is able to read messages from somewhere
type Inputter interface {
	Run(ctx context.Context, flushChan chan []clogger.Message) error
	Kill()
}

type RecvConfig struct {
	KillChannel chan bool
}

func NewRecvConfig() RecvConfig {
	return RecvConfig{
		KillChannel: make(chan bool),
	}
}
