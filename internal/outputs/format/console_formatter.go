package format

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

const termReset = "\033[0m"
const termGreen = "\033[32m"
const termCyan = "\033[36m"

type ConsoleFormatter struct {
	color bool
}

func (j *ConsoleFormatter) ColorOutput(s interface{}, field string) string {
	sStr := fmt.Sprint(s)
	if !j.color || runtime.GOOS == "windows" {
		return sStr
	}

	switch field {
	case "key":
		return fmt.Sprintf("%s%s%s", termGreen, sStr, termReset)
	case "message":
		return fmt.Sprintf("%s%s%s", termCyan, sStr, termReset)
	default:
		return sStr
	}
}

func (j *ConsoleFormatter) Format(m *clogger.Message) ([]byte, error) {
	parts := make([]string, 0, len(m.ParsedFields))

	// Hoist the message field to the front
	if msg, ok := m.ParsedFields[clogger.MESSAGE_FIELD]; ok {
		parts = append(parts, j.ColorOutput(msg, "message"))
	}

	for k, v := range m.ParsedFields {
		if k == clogger.MESSAGE_FIELD {
			continue
		}

		parts = append(parts, fmt.Sprintf("%s=%s", j.ColorOutput(k, "key"), j.ColorOutput(v, "value")))
	}

	return []byte(strings.Join(parts, " ")), nil
}
