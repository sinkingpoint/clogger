package main

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sinkingpoint/clogger/cmd/clogger/build"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
	"github.com/sinkingpoint/clogger/internal/pipeline"
	"github.com/sinkingpoint/clogger/internal/tracing"
)

func main() {
	log.Info().Str("version", build.GitHash).Msg("Started Clogger")

	tracing.InitTracing(tracing.TracingConfig{
		ServiceName:  "clogger",
		SamplingRate: 1,
		Debug:        true,
	})

	conf := inputs.NewRecvConfig()
	input, err := inputs.NewJournalDInput(conf)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create journald input")
	}

	output, err := outputs.NewStdOutputter(outputs.StdOutputterConfig{
		SendConfig: outputs.SendConfig{
			FlushInterval: time.Millisecond * 100,
			BufferSize:    1000,
		},
		Formatter: "json",
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create journald input")
	}

	pipeline := pipeline.NewPipeline([]inputs.Inputter{input}, []outputs.Sender{output})
	wg := pipeline.Run()
	wg.Wait()
}
