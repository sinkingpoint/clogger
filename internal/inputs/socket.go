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

const DEFAULT_SOCKET_PATH = "/run/clogger/clogger.sock"

type SocketInputConfig struct {
	RecvConfig
	SocketPath string
	Parser     parse.InputParser
}

func parseSocketConfigFromRaw(conf map[string]string) (SocketInputConfig, error) {
	var err error
	socketPath := DEFAULT_SOCKET_PATH
	if path, ok := conf["socket_path"]; ok {
		socketPath = path
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
		SocketPath: socketPath,
		Parser:     parser,
	}, nil
}

type SocketInput struct {
	conf SocketInputConfig
}

func NewSocketInput(c SocketInputConfig) *SocketInput {
	return &SocketInput{
		conf: c,
	}
}

func (s *SocketInput) Kill() {
	s.conf.killChannel <- true
}

func (s *SocketInput) handleConn(ctx context.Context, conn net.Conn, flush chan []clogger.Message) {
	ctx, span := tracing.GetTracer().Start(ctx, "SocketInput.handleConn")
	defer span.End()
	defer conn.Close()
	if err := s.conf.Parser.ParseStream(ctx, conn, flush); err != nil {
		span.RecordError(err)
		log.Debug().Err(err).Msg("Failed to parse incoming stream")
	} else {
		log.Debug().Msg("Connection exited cleanly")
	}
}

func (s *SocketInput) Run(ctx context.Context, flushChan chan []clogger.Message) error {
	ctx, span := tracing.GetTracer().Start(ctx, "SocketInput.Run")
	defer span.End()

	span.SetAttributes(attribute.String("socket_path", s.conf.SocketPath))

	if _, err := os.Stat(s.conf.SocketPath); !errors.Is(err, os.ErrNotExist) {
		// Delete the existing socket so we can remake it
		log.Info().Str("socket_path", s.conf.SocketPath).Msg("Cleaning up left behind socket")
		span.AddEvent("Deleting old socket")
		if err = os.Remove(s.conf.SocketPath); err != nil {
			return err
		}
	}

	l, err := net.Listen("unix", s.conf.SocketPath)
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
	// JournalDInput that reads data from the journald stream
	inputsRegistry.Register("socket", func(rawConf map[string]string) (interface{}, error) {
		return parseSocketConfigFromRaw(rawConf)
	}, func(conf interface{}) (Inputter, error) {
		if c, ok := conf.(SocketInputConfig); ok {
			return NewSocketInput(c), nil
		}

		return nil, fmt.Errorf("invalid config passed to journald input")
	})
}
