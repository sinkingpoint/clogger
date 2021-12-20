package clogger

type Message struct {
	MonoTimestamp uint64
	ParsedFields  map[string]string
	RawMessage    string
}
