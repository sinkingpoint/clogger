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
	TLS        *clogger.TLSConfig
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

	tls, err := clogger.NewTLSConfigFromRaw(conf)
	if err != nil {
		return SocketInputConfig{}, err
	}

	return SocketInputConfig{
		RecvConfig: NewRecvConfig(),
		ListenAddr: socketListen,
		Type:       ty,
		Parser:     parser,
		TLS:        &tls,
	}, nil
}

type socketInput struct {
	conf         SocketInputConfig
	internalChan chan clogger.Message
	listener     net.Listener
	wg           sync.WaitGroup
}

func NewSocketInput(c SocketInputConfig) *socketInput {
	return &socketInput{
		conf:         c,
		internalChan: make(chan clogger.Message, 10),
		wg:           sync.WaitGroup{},
	}
}

func (s *socketInput) handleConn(ctx context.Context, conn net.Conn) {
	ctx, span := tracing.GetTracer().Start(ctx, "SocketInput.handleConn")
	defer span.End()
	defer conn.Close()
	if err := s.conf.Parser.ParseStream(ctx, conn, s.internalChan); err != nil {
		span.RecordError(err)
		log.Debug().Err(err).Msg("Failed to parse incoming stream")
	}
}

func (s *socketInput) Init(ctx context.Context) error {
	var ty string
	switch s.conf.Type {
	case UNIX_SOCKET_INPUT:
		ty = "unix"
	case TCP_SOCKET_INPUT:
		ty = "tcp"
	}

	listener, err := net.Listen(ty, s.conf.ListenAddr)
	if err != nil {
		return err
	}

	s.listener = s.conf.TLS.WrapListener(listener)

	go func() {
		for {
			conn, err := listener.Accept()

			if err != nil {
				break
			}

			s.wg.Add(1)
			go func() {
				defer s.wg.Done()
				s.handleConn(ctx, conn)
			}()
		}
	}()

	return nil
}

func (s *socketInput) Close(ctx context.Context) error {
	s.listener.Close()
	s.wg.Wait()
	close(s.internalChan)
	log.Debug().Msg("Socket Inputter Closing... Waiting on child connections")

	return nil
}

func (s *socketInput) GetBatch(ctx context.Context) (*clogger.MessageBatch, error) {
	_, span := tracing.GetTracer().Start(ctx, "SocketInput.GetBatch")
	defer span.End()

	span.SetAttributes(attribute.String("socket_path", s.conf.ListenAddr))

	select {
	case <-ctx.Done():
		return nil, nil
	case msg := <-s.internalChan:
		numMessages := len(s.internalChan) + 1
		batch := clogger.GetMessageBatch(numMessages)
		batch.Messages = append(batch.Messages, msg)
		for i := 0; i < numMessages-1; i++ {
			batch.Messages = append(batch.Messages, <-s.internalChan)
		}

		return batch, nil
	}
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
