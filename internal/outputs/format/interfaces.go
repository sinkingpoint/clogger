package format

import (
	"fmt"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type Formatter interface {
	Format(m *clogger.Message) ([]byte, error)
}

func GetFormatterFromString(s string) (Formatter, error) {
	switch s {
	case "json":
		return &JSONFormatter{}, nil
	case "console":
		return &ConsoleFormatter{}, nil
	}

	return nil, fmt.Errorf("no formatter named `%s` found", s)
}
