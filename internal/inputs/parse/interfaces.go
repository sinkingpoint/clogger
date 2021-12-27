package parse

import (
	"context"
	"fmt"
	"io"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type InputParser interface {
	ParseStream(ctx context.Context, bytes io.ReadCloser, flushChan chan []clogger.Message) error
}

func GetParserFromString(s string, args map[string]string) (InputParser, error) {
	switch s {
	case "json":
		return &JSONParser{}, nil
	case "newline":
		return &NewlineParser{}, nil
	}

	return nil, fmt.Errorf("no formatter named `%s` found", s)
}
