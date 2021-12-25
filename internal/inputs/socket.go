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

func (j *SocketInput) Kill() {
	j.conf.KillChannel <- true
}

func (s *SocketInput) handleConn(conn net.Conn, flush chan []clogger.Message) {
	defer conn.Close()
	s.conf.Parser.ParseStream(conn, flush)
}

func (s *SocketInput) Run(ctx context.Context, flushChan chan []clogger.Message) error {
	if _, err := os.Stat(s.conf.SocketPath); !errors.Is(err, os.ErrExist) {
		// Delete the existing socket so we can remake it
		log.Info().Str("socket_path", s.conf.SocketPath).Msg("Cleaning up left behind socket")
		if err = os.Remove(s.conf.SocketPath); err != nil {
			return err
		}
	}

	l, err := net.Listen("unix", s.conf.SocketPath)
	if err != nil {
		return err
	}

	defer l.Close()

	wg := sync.WaitGroup{}

outer:
	for {
		select {
		case <-s.conf.KillChannel:
			break outer
		default:
			conn, err := l.Accept()

			if err != nil {
				return err
			}

			wg.Add(1)
			go func() {
				defer wg.Done()
				s.handleConn(conn, flushChan)
			}()
		}
	}

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
