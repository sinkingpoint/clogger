package outputs

import (
	"context"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type DevNullOutput struct {
	SendConfig
}

func (d *DevNullOutput) GetSendConfig() SendConfig {
	return d.SendConfig
}

// FlushToOutput takes a buffer of messages, and pushes them somewhere
func (d *DevNullOutput) FlushToOutput(ctx context.Context, messages *clogger.MessageBatch) (OutputResult, error) {
	return OUTPUT_SUCCESS, nil
}
