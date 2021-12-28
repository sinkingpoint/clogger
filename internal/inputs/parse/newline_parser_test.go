package parse_test

import (
	"bytes"
	"context"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs/parse"
)

func TestNewLineParser(t *testing.T) {
	data := []string{
		"test",
		"test1",
		"test2",
	}
	reader := ioutil.NopCloser(bytes.NewReader([]byte(strings.Join(data, "\n"))))

	parser := parse.NewlineParser{}
	c := make(clogger.MessageChannel, 10)

	err := parser.ParseStream(context.Background(), reader, c)
	if err != nil {
		t.Fatalf("Error found when parsing input: %s", err.Error())
	}

	close(c)

	numMessages := 0
	for msgWrap := range c {
		for _, msg := range msgWrap {
			msgField := msg.ParsedFields["message"]
			if msgField != data[numMessages] {
				t.Errorf("Expected %s, got %s", data[numMessages], msgField)
			}
			numMessages += 1
		}
	}

	if numMessages != len(data) {
		t.Fatalf("Expected %d messages, got %d", len(data), numMessages)
	}
}
