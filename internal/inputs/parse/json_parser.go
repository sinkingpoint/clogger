package parse

import (
	"encoding/json"
	"io"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/clogger"
)

type JSONParser struct{}

func (j *JSONParser) ParseStream(bytes io.ReadCloser, flushChan chan []clogger.Message) error {
	dec := json.NewDecoder(bytes)
	for {
		rawMessage := map[string]interface{}{}
		err := dec.Decode(&rawMessage)
		if err != nil {
			if err != io.EOF {
				log.Err(err).Msg("Failed to read from stream")
			}
			break
		}

		message := clogger.NewMessage()
		message.ParsedFields = rawMessage
		message.MonoTimestamp = time.Now().UnixNano()

		flushChan <- []clogger.Message{message}
	}

	return nil
}
