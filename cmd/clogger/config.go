package main

import (
	"context"

	"github.com/BurntSushi/toml"
	"github.com/sinkingpoint/clogger/internal/tracing"
)

type Config struct {
}

func LoadConfigFile(ctx context.Context, path string) (Config, error) {
	_, span := tracing.GetTracer().Start(ctx, "LoadConfigFile")
	config := Config{}
	toml.Unmarshal(&config)

	return config
}
