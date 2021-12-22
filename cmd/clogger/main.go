package main

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sinkingpoint/clogger/cmd/clogger/build"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
	"github.com/sinkingpoint/clogger/internal/pipeline"
)

func main() {
	log.Info().Str("version", build.GitHash).Msg("Started Clogger")

	conf := clogger.NewRecvConfig()
	input, err := inputs.NewJournalDInput(conf)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create journald input")
	}

	output, err := outputs.NewStdOutputter(outputs.StdOutputterConfig{
		SendConfig: clogger.SendConfig{
			FlushInterval: time.Millisecond * 100,
			BufferSize:    1000,
		},
		Formatter: "json",
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create journald input")
	}

	pipeline := pipeline.NewPipeline([]inputs.Inputter{input}, []clogger.Sender{output})
	wg := pipeline.Run()
	wg.Wait()
}
