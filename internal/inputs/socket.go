package inputs

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"sync"

	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs/parse"
	"github.com/sinkingpoint/clogger/internal/tracing"
	"go.opentelemetry.io/otel/attribute"
)

type SocketInputType int

const (
	UNIX_SOCKET_INPUT SocketInputType = iota
	TCP_SOCKET_INPUT
)

const DEFAULT_SOCKET_PATH = "/run/clogger/clogger.sock"
const DEFAULT_LISTEN_ADDR = "localhost:4279"

type SocketInputConfig struct {
	RecvConfig
	ListenAddr string
	Type       SocketInputType
	Parser     parse.InputParser
}

func parseSocketConfigFromRaw(conf map[string]string, ty SocketInputType) (SocketInputConfig, error) {
	var err error
	var socketListen string
	if path, ok := conf["listen"]; ok {
		socketListen = path
	} else {
		switch ty {
		case UNIX_SOCKET_INPUT:
			socketListen = DEFAULT_SOCKET_PATH
		case TCP_SOCKET_INPUT:
			socketListen = DEFAULT_LISTEN_ADDR
		}
	}

	parser, _ := parse.GetParserFromString("newline", conf)
	if parserName, ok := conf["parser"]; ok {
		parser, err = parse.GetParserFromString(parserName, conf)
		if err != nil {
			return SocketInputConfig{}, err
		}
	}

	return SocketInputConfig{
		RecvConfig: NewRecvConfig(),
		ListenAddr: socketListen,
		Type:       ty,
		Parser:     parser,
	}, nil
}

type socketInput struct {
	conf SocketInputConfig
}

func NewSocketInput(c SocketInputConfig) *socketInput {
	return &socketInput{
		conf: c,
	}
}

func (s *socketInput) Kill() {
	s.conf.killChannel <- true
}

func (s *socketInput) handleConn(ctx context.Context, conn net.Conn, flush clogger.MessageChannel) {
	ctx, span := tracing.GetTracer().Start(ctx, "SocketInput.handleConn")
	defer span.End()
	defer conn.Close()
	if err := s.conf.Parser.ParseStream(ctx, conn, flush); err != nil {
		span.RecordError(err)
		log.Debug().Err(err).Msg("Failed to parse incoming stream")
	}
}

func (s *socketInput) Run(ctx context.Context, flushChan clogger.MessageChannel) error {
	ctx, span := tracing.GetTracer().Start(ctx, "SocketInput.Run")
	defer span.End()

	span.SetAttributes(attribute.String("socket_path", s.conf.ListenAddr))

	var ty string
	switch s.conf.Type {
	case UNIX_SOCKET_INPUT:
		ty = "unix"
	case TCP_SOCKET_INPUT:
		ty = "tcp"
	}

	l, err := net.Listen(ty, s.conf.ListenAddr)
	if err != nil {
		return err
	}

	go func() {
		<-s.conf.killChannel
		l.Close()
	}()

	wg := sync.WaitGroup{}
	for {
		conn, err := l.Accept()

		if err != nil {
			break
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			s.handleConn(ctx, conn, flushChan)
		}()
	}

	// Wait for all the connections to cleanly end
	log.Debug().Msg("Socket Inputter Closing... Waiting on child connections")
	l.Close()
	wg.Wait()
	return nil
}

func init() {
	inputsRegistry.Register("unix", func(rawConf map[string]string) (interface{}, error) {
		conf, err := parseSocketConfigFromRaw(rawConf, UNIX_SOCKET_INPUT)
		if err != nil {
			return nil, err
		}

		return conf, nil
	}, func(conf interface{}) (Inputter, error) {
		if c, ok := conf.(SocketInputConfig); ok {

			if _, err := os.Stat(c.ListenAddr); !errors.Is(err, os.ErrNotExist) {
				// Delete the existing socket so we can remake it
				log.Info().Str("socket_path", c.ListenAddr).Msg("Cleaning up left behind socket")
				if err = os.Remove(c.ListenAddr); err != nil {
					return nil, err
				}
			}

			return NewSocketInput(c), nil
		}

		return nil, fmt.Errorf("invalid config passed to socket input")
	})

	inputsRegistry.Register("tcp", func(rawConf map[string]string) (interface{}, error) {
		conf, err := parseSocketConfigFromRaw(rawConf, TCP_SOCKET_INPUT)
		if err != nil {
			return nil, err
		}

		return conf, nil
	}, func(conf interface{}) (Inputter, error) {
		if c, ok := conf.(SocketInputConfig); ok {
			return NewSocketInput(c), nil
		}

		return nil, fmt.Errorf("invalid config passed to socket input")
	})
}
