package format

import (
	"encoding/json"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type JSONFormatter struct{}

func (j *JSONFormatter) Format(m *clogger.Message) ([]byte, error) {
	return json.Marshal(m.ParsedFields)
}
