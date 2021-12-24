package outputs

import (
	"context"
	"strconv"
	"time"

	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/outputs/format"
)

const DEFAULT_BATCH_SIZE = 1000
const DEFAULT_FLUSH_INTERVAL = time.Millisecond * 100

type OutputResult int

const (
	// OUTPUT_SUCCESS indicates that the data was sucessfully sent to the output
	OUTPUT_SUCCESS OutputResult = iota

	// OUTPUT_TRANSIENT_FAILURE indicates that we failed to send data to the output, but should retry (with exponential backoff)
	OUTPUT_TRANSIENT_FAILURE

	// OUTPUT_LONG_FAILURE indicates that we failed to send data to the output and we should
	// buffer it to the buffer destination, if configured - this failure is likely to take a while to resolve
	OUTPUT_LONG_FAILURE
)

// SendConfig is a config that specifies the base fields
// for all outputs
type SendConfig struct {
	// FlushInterval is the maximum time to buffer messages before outputting
	FlushInterval time.Duration

	// BatchSize is the maximum number of messages to store in the buffer before outputting
	BatchSize int

	// Formatter is the method that converts Messages into byte streams to be piped downstream
	Formatter format.Formatter
}

// NewSendConfigFromRaw is a convenience method to construct SendConfigs from raw configs
// that might have been loaded from things like the config file
func NewSendConfigFromRaw(rawConf map[string]string) (SendConfig, error) {
	conf := SendConfig{
		FlushInterval: DEFAULT_FLUSH_INTERVAL,
		BatchSize:     DEFAULT_BATCH_SIZE,
		Formatter:     &format.JSONFormatter{},
	}

	var err error
	if s, ok := rawConf["flush_interval"]; ok {
		conf.FlushInterval, err = time.ParseDuration(s)
		if err != nil {
			return SendConfig{}, err
		}
	}

	if s, ok := rawConf["batch_size"]; ok {
		conf.BatchSize, err = strconv.Atoi(s)
		if err != nil {
			return SendConfig{}, err
		}
	}

	if s, ok := rawConf["format"]; ok {
		conf.Formatter, err = format.GetFormatterFromString(s, rawConf)
		if err != nil {
			return SendConfig{}, err
		}
	}

	return conf, nil
}

// An Outputter is a thing that can take messages and push them somewhere else
type Outputter interface {
	// GetSendConfig returns the base send config of this Outputter
	GetSendConfig() SendConfig

	// FlushToOutput takes a buffer of messages, and pushes them somewhere
	FlushToOutput(ctx context.Context, messages []clogger.Message) (OutputResult, error)
}

// StartOutputter starts up a go routine that handles all the input to the given output + buffering etc
func StartOutputter(inputChan chan []clogger.Message, send Outputter, killChan chan bool) {
	s := NewSender(send.GetSendConfig(), send)
	ticker := time.NewTicker(s.FlushInterval)
outer:
	for {
		select {
		case <-killChan:
			s.Flush(context.Background(), true)
			break outer
		case <-ticker.C:
			s.Flush(context.Background(), false)
		case messages := <-inputChan:
			s.queueMessages(context.Background(), messages)
		}
	}
}
