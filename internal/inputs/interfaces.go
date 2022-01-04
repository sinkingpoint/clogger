package inputs

import (
	"context"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

// An Inputter is a thing that is able to read messages from somewhere
type Inputter interface {
	Init(ctx context.Context) error
	GetBatch(ctx context.Context) (*clogger.MessageBatch, error)
	Close(ctx context.Context) error
}

type RecvConfig struct{}

func NewRecvConfig() RecvConfig {
	return RecvConfig{}
}
