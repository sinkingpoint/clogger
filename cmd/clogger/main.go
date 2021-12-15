package main

import (
	"github.com/rs/zerolog/log"

	"github.com/sinkingpoint/clogger/cmd/clogger/build"
)

func main() {
	log.Info().Str("version", build.GitHash).Msg("Started Clogger")
}
