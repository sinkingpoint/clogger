package format

import (
	"fmt"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type Formatter interface {
	Format(m *clogger.Message) ([]byte, error)
}

func GetFormatterFromString(s string) (Formatter, error) {
	if s == "json" {
		return &JSONFormatter{}, nil
	}

	return nil, fmt.Errorf("no formatter named `%s` found", s)
}
