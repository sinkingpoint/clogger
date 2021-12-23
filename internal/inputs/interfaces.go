package inputs

import (
	"context"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type Inputter interface {
	Run(ctx context.Context, flushChan chan []clogger.Message) error
}

type RecvConfig struct {
	KillChannel chan bool
}

func NewRecvConfig() RecvConfig {
	return RecvConfig{
		KillChannel: make(chan bool),
	}
}
