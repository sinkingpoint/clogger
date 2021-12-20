package main

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/sinkingpoint/clogger/cmd/clogger/build"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs"
)

func main() {
	log.Info().Str("version", build.GitHash).Msg("Started Clogger")

	input, err := inputs.NewJournalDInput(&inputs.JournalDInputConfig{
		BatchSize: 10,
	})

	if err != nil {
		log.Fatal().Err(err).Msg("Failed to init journald input")
	}

	dst := clogger.GetMessages(context.Background(), 10)
	defer clogger.PutMessages(context.Background(), dst)
	input.Fetch(context.Background(), dst)
	for _, m := range dst.Messages {
		fmt.Println(m)
	}
}
