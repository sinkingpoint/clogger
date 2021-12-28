package filters

import (
	"context"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type Filter interface {
	Filter(ctx context.Context, msg *clogger.Message) (shouldDrop bool, err error)
}
