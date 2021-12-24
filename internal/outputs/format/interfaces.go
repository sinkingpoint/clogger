package format

import (
	"fmt"
	"strconv"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type Formatter interface {
	Format(m *clogger.Message) ([]byte, error)
}

func GetFormatterFromString(s string, args map[string]string) (Formatter, error) {
	switch s {
	case "json":
		return &JSONFormatter{}, nil
	case "console":
		var err error
		color := false
		if c, ok := args["color"]; ok {
			color, err = strconv.ParseBool(c)
			if err != nil {
				return nil, fmt.Errorf("invalid bool `%s` for color in Console output - expected true or false", c)
			}
		}

		return &ConsoleFormatter{
			color,
		}, nil
	}

	return nil, fmt.Errorf("no formatter named `%s` found", s)
}
