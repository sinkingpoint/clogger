package parse

import (
	"encoding/json"
	"io"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type JSONParser struct{}

func (j *JSONParser) ParseStream(bytes io.ReadCloser, flushChan chan []clogger.Message) error {
	dec := json.NewDecoder(bytes)
	message := clogger.Message{}
	for {
		err := dec.Decode(&message)
		if err != nil {
			break
		}

		flushChan <- []clogger.Message{message}
	}

	return nil
}
