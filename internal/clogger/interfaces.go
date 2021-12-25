package clogger

import (
	"time"
)

const DEFAULT_BATCH_SIZE = 100
const DEFAULT_FLUSH_DURATION = 10 * time.Millisecond
const MESSAGE_FIELD = "message"

type Message struct {
	MonoTimestamp uint64                 `json:"timestamp"`
	ParsedFields  map[string]interface{} `json:,inline`
}
