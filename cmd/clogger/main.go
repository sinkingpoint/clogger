package main

import (
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/alecthomas/kong"
	"github.com/rs/zerolog/log"

	"github.com/sinkingpoint/clogger/cmd/clogger/build"
	"github.com/sinkingpoint/clogger/cmd/clogger/config"
	"github.com/sinkingpoint/clogger/internal/metrics"
	"github.com/sinkingpoint/clogger/internal/pipeline"
)

func closeHandler(p *pipeline.Pipeline) {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Info().Msg("Got SIGTERM, cleanly shutting down pipeline")
		p.Kill()

		// Call Goexit instead of os.Exit to run `defer`s
		// https://github.com/golang/go/issues/38261#issuecomment-609448473
		runtime.Goexit()
	}()
}

func RunServer() {
	// tracing.InitTracing(tracing.TracingConfig{
	// 	ServiceName:  "clogger",
	// 	SamplingRate: 0.1,
	// })

	metrics.InitMetrics(config.CLI.Server.MetricsAddress)

	pipeline, err := config.LoadConfigFile("config.dot")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load config")
	}

	closeHandler(pipeline)
	pipeline.Run()
	pipeline.Wait()
}

func main() {
	log.Info().Str("version", build.GitHash).Msg("Started Clogger")

	defer func() {
		log.Info().Msg("Clogger exiting...")
	}()

	ctx := kong.Parse(&config.CLI)

	switch ctx.Command() {
	case "server":
		RunServer()
	}
}
