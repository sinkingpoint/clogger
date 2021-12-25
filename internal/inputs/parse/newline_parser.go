package parse

import (
	"bufio"
	"io"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
)

type NewlineParser struct{}

func (j *NewlineParser) ParseStream(bytes io.ReadCloser, flushChan chan []clogger.Message) error {
	scanner := bufio.NewScanner(bytes)
	for scanner.Scan() {
		line := scanner.Text()
		flushChan <- []clogger.Message{
			{
				MonoTimestamp: time.Now().UnixNano(),
				ParsedFields: map[string]interface{}{
					clogger.MESSAGE_FIELD: line,
				},
			},
		}
	}

	return nil
}
