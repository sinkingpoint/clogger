package pipeline_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/sinkingpoint/clogger/internal/filters"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
	"github.com/sinkingpoint/clogger/internal/outputs/format"
	"github.com/sinkingpoint/clogger/internal/pipeline"
)

func BenchmarkPipeline(b *testing.B) {
	b.ReportAllocs()

	input := inputs.NewGoInput()
	outputter := outputs.DevNullOutput{
		SendConfig: outputs.SendConfig{
			FlushInterval: time.Second,
			BatchSize:     10,
			Formatter:     &format.ConsoleFormatter{},
		},
	}

	pipeline := pipeline.NewPipeline(map[string]inputs.Inputter{
		"test_input": input,
	}, map[string]outputs.Outputter{
		"test_output": &outputter,
	}, map[string]filters.Filter{}, map[string][]string{
		"test_input": {"test_output"},
	})

	pipeline.Run()

	for i := 0; i < b.N; i++ {
		input.Enqueue(fmt.Sprintf("input %d", i))
	}

	pipeline.Kill()
}
