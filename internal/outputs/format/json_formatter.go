package format

import (
	"encoding/json"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type JSONFormatter struct {
	NewlineDelimited bool
}

func (j *JSONFormatter) Format(m *clogger.Message) ([]byte, error) {
	data, err := json.Marshal(m.ParsedFields)
	if err != nil {
		return nil, err
	}

	if j.NewlineDelimited {
		data = append(data, byte('\n'))
	}

	return data, nil
}
