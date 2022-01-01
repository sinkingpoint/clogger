package outputs

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs"
)

type SocketOutputType int

const (
	UNIX_SOCKET_OUTPUT SocketOutputType = iota
	TCP_SOCKET_OUTPUT
)

func (s SocketOutputType) ToString() string {
	switch s {
	case UNIX_SOCKET_OUTPUT:
		return "unix"
	case TCP_SOCKET_OUTPUT:
		return "tcp"
	}

	log.Panic().Int("type", int(s)).Msg("BUG: Unimplemented ToString for a SocketOutputType")
	return "unreachable"
}

type SocketOutputConfig struct {
	SendConfig
	ListenAddr string
	TLS        *clogger.TLSConfig
	Type       SocketOutputType
}

func parseSocketConfigFromRaw(rawConf map[string]string, ty SocketOutputType) (SocketOutputConfig, error) {
	var err error
	var destination string

	conf, err := NewSendConfigFromRaw(rawConf)
	if err != nil {
		return SocketOutputConfig{}, err
	}

	if path, ok := rawConf["destination"]; ok {
		destination = path
	} else {
		switch ty {
		case UNIX_SOCKET_OUTPUT:
			destination = inputs.DEFAULT_SOCKET_PATH
		case TCP_SOCKET_OUTPUT:
			destination = inputs.DEFAULT_LISTEN_ADDR
		}
	}

	tls, err := clogger.NewTLSConfigFromRaw(rawConf)
	if err != nil {
		return SocketOutputConfig{}, err
	}

	return SocketOutputConfig{
		SendConfig: conf,
		ListenAddr: destination,
		Type:       ty,
		TLS:        &tls,
	}, nil
}

type socketOutput struct {
	conf SocketOutputConfig
	conn net.Conn
}

func NewSocketOutput(c SocketOutputConfig) *socketOutput {
	return &socketOutput{
		conf: c,
	}
}

func (s *socketOutput) reconnect() bool {
	network := s.conf.Type.ToString()
	conn, err := net.Dial(network, s.conf.ListenAddr)
	if err != nil {
		s.conn = nil
		return false
	}

	s.conn = conn
	return true
}

func (s *socketOutput) Close(ctx context.Context) error {
	return s.conn.Close()
}

func (s *socketOutput) GetSendConfig() SendConfig {
	return s.conf.SendConfig
}

func (s *socketOutput) FlushToOutput(ctx context.Context, messages *clogger.MessageBatch) (OutputResult, error) {
	if s.conn == nil {
		if !s.reconnect() {
			return OUTPUT_TRANSIENT_FAILURE, nil
		}
	}

	dataBuffer := []byte{}

	for _, msg := range messages.Messages {
		data, err := s.conf.Formatter.Format(&msg)
		if err != nil {
			log.Warn().Msg("Failed to format message")
			continue
		}

		dataBuffer = append(dataBuffer, data...)
	}

	_, err := s.conn.Write(dataBuffer)
	if err != nil {
		s.conn = nil
		return OUTPUT_TRANSIENT_FAILURE, nil
	}

	return OUTPUT_SUCCESS, nil
}

func init() {
	outputsRegistry.Register("unix", func(rawConf map[string]string) (interface{}, error) {
		conf, err := parseSocketConfigFromRaw(rawConf, UNIX_SOCKET_OUTPUT)
		if err != nil {
			return nil, err
		}

		return conf, nil
	}, func(conf interface{}) (Outputter, error) {
		if c, ok := conf.(SocketOutputConfig); ok {

			if _, err := os.Stat(c.ListenAddr); !errors.Is(err, os.ErrNotExist) {
				// Delete the existing socket so we can remake it
				log.Info().Str("socket_path", c.ListenAddr).Msg("Cleaning up left behind socket")
				if err = os.Remove(c.ListenAddr); err != nil {
					return nil, err
				}
			}

			return NewSocketOutput(c), nil
		}

		return nil, fmt.Errorf("invalid config passed to socket input")
	})

	outputsRegistry.Register("tcp", func(rawConf map[string]string) (interface{}, error) {
		conf, err := parseSocketConfigFromRaw(rawConf, TCP_SOCKET_OUTPUT)
		if err != nil {
			return nil, err
		}

		return conf, nil
	}, func(conf interface{}) (Outputter, error) {
		if c, ok := conf.(SocketOutputConfig); ok {

			if _, err := os.Stat(c.ListenAddr); !errors.Is(err, os.ErrNotExist) {
				// Delete the existing socket so we can remake it
				log.Info().Str("socket_path", c.ListenAddr).Msg("Cleaning up left behind socket")
				if err = os.Remove(c.ListenAddr); err != nil {
					return nil, err
				}
			}

			return NewSocketOutput(c), nil
		}

		return nil, fmt.Errorf("invalid config passed to socket input")
	})
}
