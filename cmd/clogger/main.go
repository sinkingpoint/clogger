package main

import (
	"github.com/rs/zerolog/log"

	"github.com/sinkingpoint/clogger/cmd/clogger/build"
	"github.com/sinkingpoint/clogger/cmd/clogger/config"
)

func main() {
	log.Info().Str("version", build.GitHash).Msg("Started Clogger")

	// tracing.InitTracing(tracing.TracingConfig{
	// 	ServiceName:  "clogger",
	// 	SamplingRate: 1,
	// 	Debug:        true,
	// })

	pipeline, err := config.LoadConfigFile("config.dot")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	wg := pipeline.Run()
	wg.Wait()
}
