package outputs

import (
	"context"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/sinkingpoint/clogger/internal/clogger"
)

type FileOutputConfig struct {
	SendConfig
	Path string
}

type FileOutput struct {
	SendConfig
	file *os.File
}

func newFileOutputConfigFromRaw(rawConf map[string]string) (FileOutputConfig, error) {
	conf, err := NewSendConfigFromRaw(rawConf)
	if err != nil {
		return FileOutputConfig{}, err
	}

	if path, ok := rawConf["path"]; ok {
		return FileOutputConfig{
			SendConfig: conf,
			Path:       path,
		}, nil
	}

	return FileOutputConfig{}, fmt.Errorf("missing `path` required for FileOutput")
}

func NewFileOutput(conf FileOutputConfig) (*FileOutput, error) {
	file, err := os.Create(conf.Path)

	if err != nil {
		return nil, err
	}

	return &FileOutput{
		SendConfig: conf.SendConfig,
		file:       file,
	}, nil
}

func (f *FileOutput) Close(ctx context.Context) error {
	return f.file.Close()
}

func (f *FileOutput) GetSendConfig() SendConfig {
	return f.SendConfig
}

func (f *FileOutput) FlushToOutput(ctx context.Context, messages *clogger.MessageBatch) (OutputResult, error) {
	for _, msg := range messages.Messages {
		data, err := f.SendConfig.Formatter.Format(&msg)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to format message")
			continue
		}

		_, err = f.file.Write(data)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to write message")
			continue
		}
	}

	return OUTPUT_SUCCESS, nil
}

func init() {
	outputsRegistry.Register("file", func(rawConf map[string]string) (interface{}, error) {
		conf, err := newFileOutputConfigFromRaw(rawConf)
		if err != nil {
			return nil, err
		}

		return conf, nil
	}, func(conf interface{}) (Outputter, error) {
		if c, ok := conf.(FileOutputConfig); ok {
			return NewFileOutput(c)
		}

		return nil, fmt.Errorf("invalid config passed to file input")
	})
}
