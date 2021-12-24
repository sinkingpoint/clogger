package pipeline_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sinkingpoint/clogger/internal/clogger"
	"github.com/sinkingpoint/clogger/internal/inputs"
	"github.com/sinkingpoint/clogger/internal/outputs"
	"github.com/sinkingpoint/clogger/internal/pipeline"
	"github.com/sinkingpoint/clogger/testutils/mock_inputs"
	"github.com/sinkingpoint/clogger/testutils/mock_outputs"
)

func TestPipeline(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockInput := mock_inputs.NewMockInputter(ctrl)
	mockInput.EXPECT().Run(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, input chan []clogger.Message) error {
		input <- []clogger.Message{
			{
				MonoTimestamp: 0,
				RawMessage:    "test",
			},
			{
				MonoTimestamp: 1,
				RawMessage:    "test2",
			},
			{
				MonoTimestamp: 2,
				RawMessage:    "test3",
			},
		}

		return nil
	}).MaxTimes(1)

	mockOutput := mock_outputs.NewMockOutputter(ctrl)
	mockOutput.EXPECT().GetSendConfig().Return(outputs.SendConfig{
		FlushInterval: time.Millisecond * 100,
		BatchSize:     3,
	}).MinTimes(1).MaxTimes(1)

	mockOutput.EXPECT().FlushToOutput(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, messages []clogger.Message) error {
		if len(messages) != 3 {
			t.Errorf("Buffer wasn't completly flushed - expected 3 messages, got %d", len(messages))
		}
		return nil
	}).MinTimes(1).MaxTimes(1)

	pipeline := pipeline.NewPipeline(map[string]inputs.Inputter{
		"test_input": mockInput,
	}, map[string]outputs.Outputter{
		"test_output": mockOutput,
	}, map[string][]string{
		"test_input": {"test_output"},
	})

	pipeline.Run()

	// Wait for a bit to let the pipeline chug
	time.Sleep(time.Millisecond * 100)
}
