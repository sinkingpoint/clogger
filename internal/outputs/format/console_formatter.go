package format

import (
	"fmt"
	"strings"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type ConsoleFormatter struct{}

func (j *ConsoleFormatter) Format(m *clogger.Message) ([]byte, error) {
	parts := make([]string, 0, len(m.ParsedFields))
	if msg, ok := m.ParsedFields["message"]; ok {
		parts = append(parts, fmt.Sprintf("%s", msg))
	}

	for k, v := range m.ParsedFields {
		if k == "message" {
			continue
		}

		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}

	return []byte(strings.Join(parts, " ")), nil
}
