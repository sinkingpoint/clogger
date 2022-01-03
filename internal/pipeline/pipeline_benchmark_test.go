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
	"github.com/sinkingpoint/clogger/internal/tracing"
)

func BenchmarkPipeline(b *testing.B) {
	b.ReportAllocs()

	tracing.InitTracing(tracing.TracingConfig{
		ServiceName:  "clogger",
		SamplingRate: 1,
	})

	input := inputs.NewGoInput()
	outputter := outputs.DevNullOutput{
		SendConfig: outputs.SendConfig{
			FlushInterval: time.Second,
			BatchSize:     10,
			Formatter:     &format.ConsoleFormatter{},
		},
	}
	filter, err := filters.NewTengoFilterFromString([]byte(`
	shouldDrop := false`))

	if err != nil {
		b.Fatal(err)
	}

	pipeline := pipeline.NewPipeline(map[string]inputs.Inputter{
		"test_input": input,
	}, map[string]outputs.Outputter{
		"test_output": &outputter,
	}, map[string]filters.Filter{
		"test_filter": filter,
	}, map[string][]pipeline.Link{
		"test_input":  {pipeline.NewLink("test_filter")},
		"test_filter": {pipeline.NewLink("test_output")},
	})

	pipeline.Run()

	for i := 0; i < b.N; i++ {
		input.Enqueue(fmt.Sprintf("input %d", i))
	}

	pipeline.Kill()
}
