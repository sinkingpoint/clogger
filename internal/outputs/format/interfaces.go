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
		var err error
		newlines := false
		if n, ok := args["newlines"]; ok {
			newlines, err = strconv.ParseBool(n)
			if err != nil {
				return nil, fmt.Errorf("invalid bool `%s` for newline delimiting in JSON output - expected true or false", n)
			}
		}

		return &JSONFormatter{
			NewlineDelimited: newlines,
		}, nil
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
